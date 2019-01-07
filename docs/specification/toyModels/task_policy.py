
#import sys

#####################################################################################
# When there is a fixed "data drop"

linkA = [ "A1", "",  "A2", "A3", "",   "",  "", "", "A4", "A5", "",  "A6",  "",   "A7", "A8", "A9", "A10","A11" ]
linkB = [ "B1", "B2","B3", "B4", "B5", "B6","", "", "B7", "B8", "B9","B11", "",   "",   "",   "B11","B11","B11"]
linkC = [ "C1", "",  "",   "",   "",   "",  "", "", "C2", "",   "",  "",    "C3", "C4", "",   "",   "C7", "C8" ]

 # thin out the data modulo here in the smart link delivery

#####################################################################################
# Fixed input records, tetris block
# schema
 
requireA = 2
requireB = 3
requireC = 1

#####################################################################################
# one of each, all fresh
print("One of each, simple, effectively a sliding window because keeps last for empty --------------------")

for index in range(0,16):
    if (linkA[index] != ""):
        A = linkA[index]
    if (linkB[index] != ""):
        B = linkB[index]
    if (linkC[index] != ""):
        C = linkC[index]

    # always ready to advance
    print (index , ":= " + "(" + A + "," + B + "," + C + ")")

# Now with unequal longitudinal sets, partitioned
print ("Seq, aggregation Unequal requirements, partitioned non-overlapping sets --------")

#

AA = []
BB = []
CC = []
Xindex = 0
Afull = Bfull = Cfull = 0
indexA = indexB = indexC = 0

# begin

for index in range(0,30):
    
    if (not(Afull) and linkA[indexA] != ""):
        AA.append(linkA[indexA])

    if (not(Bfull) and linkB[indexB] != ""):
        BB.append(linkB[indexB])

    if (not(Cfull) and linkC[indexC] != ""):
        CC.append(linkC[indexC])

    #print "(",indexA,indexB,indexC, ") " ,index, ":" , AA , " ; " , BB , " ; ",  CC

    if (Afull and Bfull and Cfull):
        print (Xindex , ":> " + "(" , AA , "," , BB , "," , CC , ")")
        AA.clear()
        BB.clear()
        CC.clear()
        Afull = Bfull = Cfull = 0
        Xindex += 1

    if (len(AA) < requireA  and indexA < 16):
        indexA += 1
    else:
        Afull = 1

    if (len(BB) < requireB and indexB < 16):
        indexB += 1
    else:
        Bfull = 1
        
    if (len(CC) < requireC  and indexC < 16):
        indexC += 1
    else:
        Cfull = 1

# end
#####################################################################################
# Now with unequal longitudinal sets, sliding by 1
print ("Unequal requirements, sliding overlapping sets ----------")

AA = []
BB = []
CC = []
Xindex = 0
Afull = Bfull = Cfull = 0
indexA = indexB = indexC = 0

# begin

for index in range(0,30):
    
    if (not(Afull) and linkA[indexA] != ""):
        AA.append(linkA[indexA])

    if (not(Bfull) and linkB[indexB] != ""):
        BB.append(linkB[indexB])

    if (not(Cfull) and linkC[indexC] != ""):
        CC.append(linkC[indexC])

    #print "(",indexA,indexB,indexC, ") " ,index, ":" , AA , " ; " , BB , " ; ",  CC

    if (Afull and Bfull and Cfull):
        print (Xindex , ":> " + "(" , AA , "," , BB , "," , CC , ")")
        if len(AA) > 0:
            del AA[0]
        if len(BB) > 0:
            del BB[0]
        if len(CC) > 0:
            del CC[0]
        Afull = Bfull = Cfull = 0
        Xindex += 1

    if (len(AA) < requireA  and indexA < 16):    # This AND leads to a bug in type formatting at the end
        indexA += 1
    else:
        Afull = 1

    if (len(BB) < requireB and indexB < 16):
        indexB += 1
    else:
        Bfull = 1
        
    if (len(CC) < requireC  and indexC < 16):
        indexC += 1
    else:
        Cfull = 1

# end

#####################################################################################
# Now with unequal longitudinal sets, no sliding, timeout
print ("Min/Max, timeout, partitions ----------")

AA = []
BB = []
CC = []
Xindex = 0
Aready = Bready = Cready = 0
indexA = indexB = indexC = 0
RESET = 3
timeout = RESET

# If no definite shape, e.g. a histogram, then limits might be set by array sized or memory say

minA = 1
maxA = 1
minB = 1
maxB = 1    # b's are big, so we can max out on 4
minC = 1
maxC = 1

# e.g. format a page with max 4 photos and 10 comments/scores - a union type

# begin

for index in range(0,40):

    if (len(AA) < maxA):
        if (linkA[indexA] != ""):
            AA.append(linkA[indexA])
        indexA += 1

    if (len(BB) < maxB):
        if (linkB[indexB] != ""):
            BB.append(linkB[indexB])
        indexB += 1

    if (len(CC)< maxC):
        if (linkC[indexC] != ""):
            CC.append(linkC[indexC])
        indexC += 1

    # Common timeout, else the different channel constraints will conflict
    timeout -= 1
    
    all_ready = (Aready and Bready and Cready)

    if (timeout == 0):
        print ("TIMEOUT!")
        
    if ((timeout == 0) or all_ready):
        print (Xindex , ":> " + "(" , AA , "," , BB , "," , CC , ")")
        AA.clear()
        BB.clear()
        CC.clear()
        Aready = Bready = Cready = 0
        timeout = RESET
        Xindex += 1

    if (len(AA) >= minA):
        Aready = 1        

    if (len(BB) >= minB):
        Bready = 1        

    if (len(CC) >= minC):
        Cready = 1        

# end

