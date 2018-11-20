#!/usr/bin/python3

#####################################################################################################
# defs
#####################################################################################################

import sys

#####################################################################################################
# Data incoming samples , dummy data stream
#####################################################################################################

maxlinks = 3

# model of the incoming pipe links, and input buffer

link     = [ "1" , "2" , "maxlinks"] # the channel
snapshot = [ "1" , "2" , "maxlinks"] # channel buffer

link[0] = [ "A1", "",  "A2", "A3", "",   "",  "", "", "A4", "A5", "",  "A6",  "",   "A7", "A8", "A9", "A10","A11" ]
link[1] = [ "B1", "B2","B3", "B4", "B5", "B6","", "", "B7", "B8", "B9","B11", "",   "",   "",   "B11","B11","B11"]
link[2] = [ "C1", "",  "",   "",   "",   "",  "", "", "C2", "",   "",  "",    "C3", "C4", "",   "",   "C7", "C8" ]

#####################################################################################################
# Internal work space
# data buffers for each channel
#####################################################################################################

# internal state - counters/flags

cursor   = [1,2,maxlinks] # channel pointer
required = [1,2,maxlinks] # schema
slide    = [1,2,maxlinks] # schema
ready    = [1,2,maxlinks] # channel trigger criterion
minfill  = [1,2,maxlinks] # channel trigger criterion
maxfill  = [1,2,maxlinks] # channel trigger criterion

snapshot[0] = []
snapshot[1] = []
snapshot[2] = []

# environment

clock_time = 0
last_commit = 0
cursor   = [0,0,0]
ready    = [False,False,False]
timed_out = False

#####################################################################################################
# policy for schema, how many to waitfor
#####################################################################################################

# TODO
# align timestamps, match by time
# drop too many in shapshotting advance cursor to drop

#
# Main TASK policy switch
#

policy = sys.argv[1] # [ "swap_new4old", "all_new", "sliding_window", "align_stamps" ]

#
# Additional TASK policy tuning "knobs"
#

required[0] = 3
required[1] = 2
required[2] = 1

#

minfill[0] = 3    # need to sanity check minmax against required
maxfill[0] = 3
minfill[1] = 2
maxfill[1] = 2
minfill[2] = 2
maxfill[2] = 2

#

slide[0] = 1
slide[1] = 1
slide[2] = 1

#

aggregation_timeout = 5


#####################################################################################################
# Helper fns
#####################################################################################################

def CheckPolicy():

    for index in range(0,maxlinks):

        if required[index] > 0:
            print("required policy set, determines minfill/maxfill for link ",index)
            minfill[index] = maxfill[index] = required[index]

        if maxfill[index] < minfill[index]:
            print("Error, maxfill[] policy is less than minfill[] for link ",index)
            return False

        if policy == "sliding_window":
            if slide[index] > minfill[index]:
                print("Error, sliding_window policy asks for slide[] window bigger than schema")
                return False

        if policy == "swap_new4old":
            if required[index] != 1 or minfill[index] != 1:
                print("Error, swap_old4new policy is ambiguous when required[] / minfill[] partition bigger 1")
                return False

    print("*")
    print("* POLICY = " + policy)
    print("*  required ", required)
    print("*  minfill  ", minfill)
    print("*  maxfill  ", maxfill)
    if policy == "sliding_window":
        print("*  slide    ", slide)
    print("*")

    return True

###########################

def UpdateAnnotatedValue():

    # poll the links for data
    for index in range(0,maxlinks):

        if ready[index]:
            continue

        data = link[index][cursor[index]] 

        if (len(snapshot[index]) <= maxfill[index]):
            cursor[index] += 1

        # If empty swap_new4old, leave snapshot alone

        if policy == "swap_new4old" and data == "":
            ready[index] = True
            continue

        # All other cases, update snapshot
        
        if data != "":

            if policy == "swap_new4old":
                # don't append, reset
                snapshot[index].clear()
                ready[index] = True

            snapshot[index].append(data)

            if (len(snapshot[index]) >= minfill[index]):
                ready[index] = True

###########################

def AcceptanceCriteria(ready):

    # involuntary timeout if aggregation takes too long
    
    global clock_time, last_commit, timed_out
    clock_time += 1
    elapsed = clock_time - last_commit

    if elapsed > aggregation_timeout:
        timed_out = True
        return False
            
    # normal
    
    allready = True

    for index in range(0,maxlinks):
        allready &= ready[index]

    return allready

###########################

def CommitAnnotatedValue():

    global last_commit, clock_time, timed_out

    # conditions met for full commit, or timed_out (incomplete)
    
    last_commit = clock_time

    if timed_out:

        # This print is really RUN CONTAINER
        print (clock_time , "commit/exec (timeout)" + "(" , snapshot , ")")
        timed_out = False # reset

    else:

        # This print is really RUN CONTAINER
        print (clock_time , "commit/exec (filled)" + "(" , snapshot , ")")

    # clear previous data from links according to policy
    
    for index in range(0,maxlinks):
            
        ready[index] = False

        if policy == "swap_new4old":
            # don't wipe old value
            continue
        
        if policy == "all_new":
            snapshot[index].clear()
            
        if policy == "sliding_window":
            if len(snapshot[index]) > 0:
                del snapshot[index][0]    # todo: slide by N > 1

#####################################################################################################
# BEGIN
#####################################################################################################

if CheckPolicy():

    while True:
    
        UpdateAnnotatedValue()

        allready = AcceptanceCriteria(ready)

        if (allready or timed_out):
            CommitAnnotatedValue()

