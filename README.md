# Koalja Pipeline Operator

## Building

```bash
export DOCKERNAMESPACE=<your-docker-hub-account-name>
make docker
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
