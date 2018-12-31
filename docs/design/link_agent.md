# Link Agent

## High level story

- The link agent receives annotated values from an outputs of a task (push) and
  makes them available to inputs of another task (pull).
- Between the time that the link agent has received the annotated value
  and the time when the annotated value is acknowledged by the task
  on the outgoing side of the link, the link will store the annotated value.
  The means of storage is implementation dependent.
  It is up to the operator of the Koalja system to decide which storage implementation
  to use.

## Program structure

### Program entry

- The link agent starts its `main` in `cmd/agents/main.go`
- From `main` it executes a `link` command (`cmdLinkAgentRun`) in `cmd/agents/link_agent.go`
- The `link` command collects arguments and prepares the environment.
  It creates an API builder depending on the selected storage implementation (`--queue=...`).
  It then creates a service found in `pkg/agent/link`, passes the API builder and runs it.

### Service operations

- See `pkg/agent/link/service.go`
- When the link agent service is created, it uses a provided API builder to create
  API implementations for an annotated value publisher & source.
- When the link agent service runs, it first launches a registration go-routine.
  This go-routine registers the link agent with the pipeline agent at regular intervals.
- Once the first registration has succeeded, it runs a GRPC server for the GRPC API's
  that are provided by the link agent (annotated value publisher & source).

### AnnotatedValuePublisher

- See `pkg/annotatedvalue.annotatedvalue.proto`
- This API is used by the output publisher of the task agent to send annotated values
  into a link.
- Rate limiting provided by the `CanPublish` function.
- The actual implementation of this service depends of the storage system
  that was choosen. Currently there is only one storage implementation: `stub`.
  It stores annotated values in an in-memory queue. See `pkg/agent/link/stub`.
- Note that this API is also implemented by the pipeline agent for the output
  of egress tasks.

### AnnotatedValueSource

- See `pkg/annotatedvalue.annotatedvalue.proto`
- This API is used by the input loop of the task agent to query for annotated values
  from a link.
- This API uses a two-step approach. The user will first request a `Subscription`
  and then called `Next` requests using that subscription to pull annotated values.
  This indirection is there to deal with scaling of task agents and failing task agents.
- An annotated value is return from a `Next` request in only one subscription.
  This subscription keeps a record of the annotated values it has returned.
  If the subscription is considered failing (because task agent is gone)
  then all the annotated values it has returned but are not yet acknowledged
  will be passed to other subscriptions.
- Users of API must call `Ping` requests to indicate that their subscription
  is still valid.
