# Task Agent

## High level story

- The task agent loads the `Pipeline` resource (specification) and the name of
  the task it is responsible for from the environment (and kubernetes).
- The task agent queries incoming links for annotated values (See input loop).
- The annotated values are processed into snapshots (See input loop)
  according to a SnapshotPolicy specified in the pipeline.
- Snapshots are passed to an executor service. That will prepare a Pod
  for executing on a snapshot. Preparation involves making data, from annotated values
  in the snapshot, available to the pod, preparing space for resulting data to be
  written to and creating the Pod specification (e.g. commandline, environment vars).
  Once prepared, the Pod is launched and the task agent waits until it is finished.
- Once the task executor Pod is finished, it leaves resuling data in prepared locations.
  URI's are created for all of the resulting data items and from those URI's annotated values
  are created and published to the outgoing links (See output publisher service).

## Program structure

### Pipeline specification

- The task agents uses the pipeline specification (see `pkg/apis/koalja/v1alpha1/pipeline_types.go`).

### Program entry

- The task agent starts its `main` in `cmd/agents/main.go`
- From `main` it executes a `task` command (`cmdTaskAgentRun`) in `cmd/agents/task_agent.go`
- The `task` command collects arguments and prepares the environment.
  It then creates a service found in `pkg/agent/task` and runs it.

### Service operations

- See `pkg/agent/task/service.go`
- When the service is created, it loads the `Pipeline` (see `pkg/apis/koalja/v1alpha1`)
  from the environment and kubernetes.
- When the task agent service runs, it first launches a registration go-routine.
  This go-routine registers the task agent with the pipeline agent at regular intervals.
- Once the first registration has succeeded, it starts go-routines for various
  internal services:
  - A kubernetes resource cache; this is just a cache helping to speed up kubernetes resource lookups
  - The input loop; this polls incoming links for annotated values
  - The snapshot service; this provides an API for custom task executors to collect snapshots
  - The executor service; this executes the task executor for snapshots
  - The output publisher service; this sends created annotated values to outgoing links
  - The Pod garbage collector; this cleans up left of executor pods
  - The GRPC server; this runs a network service, serving several GRPC API's
- The service then waits forever.

### Input loop

- See `pkg/agent/task/input_loop.go`
- The input loop launches go-routines to watch all inputs of the task.
  Every watcher (`watchInput`) keeps pulling annotated values from its incoming link
  and sends it on for processing (`processAnnotatedValue`).
- In `processAnnotatedValue` the annotated value is inserted into the "current"
  snapshot. If the annotated value cannot be inserted into the "current"
  snapshot yet, the called is blocked until there is room for it (according to the limits specified
  in `TaskInputSpecification`).
  Then the "current" snapshot is being checked and when it is ready to be executed
  it is cloned and the clone is put into an execution channel. The "current" snapshot
  is prepared (according to a `SnapshotPolicy`) for receiving new annotated values.
- A separate go-routine (`processExecQueue`) is watching the execution channel
  for incoming snapshots. Depending on the launch policy of the task it will
  - execute the snapshot (Auto launch policy) or
  - cancel an ongoing execution and then execute the snapshot (Restart launch policy) or
  - pass the snapshot to the snapshot service for retrieval by the custom executor (Custom launch policy)

### Snapshot service

- See `pkg/agent/task/snapshot_service.go`
- This service implements a GRPC service used by custom task executors (see `pkg/task/task.proto#SnapshotService`)
  The service uses a channel to synchronize snapshots and to ensure that only one executes at a time
- Part of the GRPC service is a function to execute go templates (`ExecuteTemplate`).
  This function is used by the custom executor to fill a template (that the custom executor provides)
  with data from a snapshot.

### Executor service

- See `pkg/agent/task/executor.go`
- This service primary function is `Execute`, that (given a snapshot) prepares and launches
  a Pod to execute the task executor with that snapshot.
- The `Execute` function uses `ExecutorInputBuilder` implementations (specific to the scheme of the URI in an annotated value)
  to prepare a Volume for accessing annotated value of the snapshot for a specific task input.
  Besides the Volume, these input builders also provide data to put into a template (e.g. to build commandline arguments).
  The input builders are implemented in `pkg/agent/task/builders/input`.
- The `Execute` function uses `ExecutorOutputBuilder` implementations (currently only for File scheme)
  to prepare a Volume where the task executor can store data for a specific task output.
  Besides the Volume, these output builders also provide data to put into a template (e.g. to build commandline arguments)
  and a set of output processors to do something with the data once the executor has finished.
- Once the Pod is specified, it will be launched and the `Execute` functions waits for the Pod to finish.
- When the Pod has finished succesfully, the output processors are called to collect the data and
  send it as newly created annotated values to the output publisher.

### Output publisher service

- See `pkg/agent/task/output_publisher.go`
- This service it used to send annotated values (resulting from task execution) to outgoing
  links.
- In its `Publish` function is will fill any missing fields of the annotated value,
  link it with the annotated values from the snapshot it was created for and then
  send it to the outgoing link(s).
- Since a task output can be connected to multiple links, the `Publish` function
  publishes the annotated value into internal channels, one for every outgoing
  link. Go-routines are waiting on these internal channels to actually send
  annotated values to individual links.

### Pod garbage collector

- See `pkg/agent/task/pod_gc.go`
- This service will remove all executor Pods that have finished.
- Pods that have finished succesfully are removed right away.
- Pods that resulted in an error are put in a queue.
  Once there are more than 5 pods in the queue, the oldest Pod is removed.
  This is done to allowed the operator to inspect failing Pods.
