# Pipeline Agent

## High level story

- The pipeline agent provides a registry for task & link agents.
- The pipeline agent receives annotated values from outputs of egress tasks.
  The means for storing these annotated values is implementation dependent.
- The pipeline agent serves a web UI for the pipeline.
- The pipeline agent receives statistics from task & link agents.
- The Pod that contains the pipeline agent also contains an annotated value
  registry (in a second container). This registry is not covered in this document.

## Program structure

### Program entry

- The pipeline agent starts its `main` in `cmd/agents/main.go`
- From `main` it executes a `pipeline` command (`cmdPipelineAgentRun`) in `cmd/agents/pipeline_agent.go`
- The `pipeline` command collects arguments and prepares the environment.
  It creates an API builder depending on the selected storage implementation (`--queue=...`).
  It then creates a service found in `pkg/agent/pipeline`, passes the API builder and runs it.

### Service operations

- See `pkg/agent/pipeline/service.go`
- When the service is created, it loads the `Pipeline` (see `pkg/apis/koalja/v1alpha1`)
  from the environment and kubernetes.
- When the pipeline agent service is created, it uses a provided API builder to create
  API implementations for an annotated value publisher, an agent registry, a statistics sink
  and a frontend server.
- When the pipeline agent service runs, it creates go-routines to run various internal services.

### AnnotatedValuePublisher

- See `pkg/annotatedvalue.annotatedvalue.proto`
- This API is used by the output publisher of egress task agents to send annotated values
  that are considered "output of the pipeline".
- Rate limiting provided by the `CanPublish` function.
- The actual implementation of this service depends of the storage system
  that was choosen. Currently there is only one storage implementation: `stub`.
  It stores annotated values in an in-memory queue. See `pkg/agent/pipeline/stub`.
- Note that this API is also implemented by link agents for the output
  of non-egress tasks.

### AgentRegistry

- See `pkg/agent/pipeline/agent_api.proto#AgentRegistry`
- This API is used by the task & link agents to register themselves at regular
  intervals.
  This provides the pipeline agent with an "view" of which agents are
  currently active.

### Frontend API

- See `pkg/agent/pipeline/agent_api.proto#Frontend`
- This API is used by the frontend (javascript) to get the information it
  needs to display the pipeline, its current status and annotated values
  that are created during the execution of the pipeline.
- This API is sed both as a GRPC service, as well as an HTTP service (see `pkg/agent/pipeline/server_frontend.go`).
  The HTTP server is generated from options in the GRPC service and
  forwards all requests to the GRPC service implementation.

### Frontend

- See `frontend/*`
- The frontend is a React application that connects to the frontend API (of the pipeline agent)
  using HTTP/JSON.
- The frontend is compiled to javascript using webpack. See `frontend/Dockerfile.build`.
- The compiled application resources are converted to binary assets that are included
  in the go code of the pipeline agent. See `frontend/assets.go` & `Makefile#frontend/assets.go`.
- The binary assets are served over the same HTTP server that also serves the HTTP frontend API
  (see `pkg/agent/pipeline/server_frontend.go`).

### Frontend hub

- See `pkg/agent/pipeline/hub`
- The frontend React application opens a websocket to the pipeline agent as listens
  to incoming messages.
- Upon an incoming message the React application reloads the data from the pipeline agent.
  This ensures that the web UI is updated very quickly after a change in the overall
  pipeline status occurs.
- The statistics sink uses the go side of this hub (`StatisticsChanged`) to trigger
  a change notification.

### Statistics Sink

- See `pkg/tracking/statistics.proto`
- This API is used by task & link agents to publish statistics such as the current
  number of annotated values waiting in a link or the amount of executions of
  a task agent.
- The frontend displays these statistics in the web UI.
