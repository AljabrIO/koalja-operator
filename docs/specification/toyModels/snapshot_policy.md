
# Links and their Annotated Values

A link delivers an Annotated Value (AV) as a single text file to a task, handing it as file to argv.
Since Tasks are containers, there is no other natural communications channel for passing values.
The source and destination for AVs may be as files, or in any kind of database or message queue.
Smart links marshall the data as files for the task code.
A task specification has the form:

``` (input1 input2 ....) taskname (output1, output2 ...)
```

The inputs and outputs represent tuples of files that are collected as a `snapshot' set.
A named input may have a buffer size, representing the minimum number of AVs required to
execute the container. 

``` (input1[5] input2 ....) taskname (output1, output2 ...)
```

This increases the number of files in the task's argv[]. 
Outputs do not have buffers.

# Snapshot policy

A snapshot is a set of input files to be substituted for argv in the task container.
Even on a single input, there are two ways of aggregating data: 

- Each input (or input buffer) reads new incoming data, using them only once to produce an output. 

- An input maintains a sliding window in which a subset of values are replaced, like a queue.

When several inputs are combined, they form a tuple of inputs, each of which might be a buffer:

``` (input1[5] input2[2] ....) taskname (output1, output2 ...)
```

Then, a snapshot, leading to a set of argv values and executed outputs, needs a policy
to decide how to advance the AVs arriving on each of the links independently to form the tuples.

- Some of the inputs have new data, but we want to recompute immediately;
e.g. when compiling software, changes to only a few files are common, but all files
are needed to built the output software. So then, older values of some inputs are combined
with new values for changed files. The result is a mixed policy (partial swap of new for old).

- Wait for a sufficient number of new values to arrive on all incoming links before recomputing an output.
This has several variations (min/max or exact required number), with different consequences for
what the task code will be fed.

Snapshot policy needs some kind of rate control to avoid needless unintended recomputation.

In practice, only a few of these policies are likley to be common.

