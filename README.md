# Koalja Pipeline Operator

## Building

```bash
export DOCKERNAMESPACE=<your-docker-hub-account-name>
make build-image docker
```

## Kubernetes cluster preparations

```bash
# Install heptio contour
kubectl apply -f https://j.hept.io/contour-deployment-rbac
```

Once Contour is deployed, lookup the loadbalancer `Service` in the
`heptio-contour` namespace. Make sure to configure DNS appropriately for
all domains (see [Domain management](#Domain-management)) such that
these domains are mapped to the Contour loadbalancer.

## Deployment

```bash
# Install the CRD
make install
# Deploy operator
make deploy
```

## Domain management

For every pipeline, a hostname is created and served on the Contour ingress load-balancer.
The hostname is `<pipeline-name>.<namespace>.<domain-suffix>`.

The `domain-suffix` can be configured using a `ConfigMap` in the namespace of the pipeline
(or globally in the `koalja-system` namespace).

Using the `ConfigMap` you can specify one or more `domain-suffixes`. The suffix that is
used is selected using a label selector for every suffix. The suffix with the most specific
label selector that matches the labels of the `Pipeline` will be used.

The `ConfigMap` must be named `koalja-domain-config` and requires the following data format.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: koalja-domain-config
data:
  # These are example settings of domain.
  # example.org will be used for pipelines having app=prod.
  example.org: |
    selector:
      app: prod
  # Default value for domain, for pipelines that does not have app=prod labels.
  # Although it will match all pipelines, it is the least-specific rule so it
  # will only be used if no other domain matches.
  example.com: ""
```

## Service management

The Koalja Pipeline resource controller will launch multiple agents and services
for every pipeline resource. The containers for those agents and services are
configured using custom resources (see [config/agents](./config/agents/))
and a `ConfigMap` to select which resource will be used.

This way, Koalja can support multiple implementations of services & agents
and you can choose which one to use. For example you can choose between
(default) in memory implementations of ArangoDB database based implementations.

To select non-default implementations, create a `ConfigMap` like below and
add it to the namespace in which you're deploying your pipeline resources.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: koalja-services-config
data:
  config: |
    annotated-value-registry-name: arangodb-annotatedvalue-registry
```

## Storage Management

The Koalja Operator deploys pipeline components that rely on a FileSystem Storage
Service for storing long term data assets.
This service is deployed separately from the operator itself.

There are multiple implementations of this service.

- Local: An test only implementation that stores data on the FS of the k8s Nodes.
- S3: An implementation that stores data in a S3 compatible object store.

The selection between these implementation is made using an environment
variable named `STORAGETYPE` which can be set to either `local` or `s3`.

The `make deploy` target willuses this variable to deploy the appropriate resources.

### S3 Storage Service Configuration

The S3 storage service must be configured before it can be used.
To do so, deploy a yaml file like this:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: koalja-s3-default-secrets
  namespace: koalja-system
data:
  access-key: <base64 encoded access key>
  secret-key: <base64 encoded secret key>
type: aljabrio/koalja-flex-s3

---

apiVersion: v1
kind: ConfigMap
metadata:
  name: koalja-s3-storage-config
  namespace: koalja-system
data:
  default: |
    name: <name of the s3 bucket>
    endpoint: <endpoint of the s3 server>
    secretName: koalja-s3-default-secrets
```

The `ConfigMap` may contain multiple object storages, only a `default`
object storage is required.

### S3 Storage Considerations on Kubernetes Platforms

- On GKE:
  - Ensure that the nodes support `fusermount` (e.g. use ubuntu images)
  - `kubectl apply -f config/platforms/gke`

## Directories

- [`cmd`](./cmd/) Package containing entry points (`main` functions) for the binaries of this project.
- [`config`](./config/) Directory containing (customizable) deployment configuration.
- [`docker`](./docker/) Directory containing docker image build files.
- [`docs`](./docs/) Directory containing design & specification documents and experiments.
- [`examples`](./examples/) Example pipelines.
- [`frontend`](./frontend/) Pipeline frontend (react application).
- [`hack`](./hack/) Kubebuilder related template files.
- [`pkg`](./pkg/) Package root for all koalja libraries & services.
- [`third_party`](./third_party) Directory containing 3rd party code mainly related to GRPC & protocol buffers.
- [`vendor`](./vendor/) External go libraries.
