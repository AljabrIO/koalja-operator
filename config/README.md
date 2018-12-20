# Configuration files

All files under this directory are used when deploying Koalja on a Kubernetes
cluster.

The root [Makefile](../Makefile) uses [`kustomize`](https://sigs.k8s.io/kustomize)
to generate up to date YAML files for deployment.

The `kustomization` part is done through patch files.
Currently on the container images of the various deployments are customized
to their latest SHA256 tags. With this customization we can ensure that
a development deployment will always use the latest build code versions.

## Directories

- `agents` Contains resources that register various types of agents.
- `crds` Contains CustomResourceDefinition resources. These are auto-generated during make.
- `manager` Contains deployment resources for the Koalja operator itself.
- `namespaces` Contains namespace resource for `koalja-system` namespace.
- `operator` Contains `kustomize` files for deploying the operator and all related components.
- `patches` Contains `kustomize` patch files used for updating container images.
- `platforms` Contains resources for specific Kubernetes platforms.
- `rbac` Contains role & rolebinding resources. These are auto-generated during make.
- `samples` Contains auto-generated (by kubebuilder) samples. These are not used.
- `services` Contains deployment resources for pipeline services (e.g. FileSystem).
- `storage` Contains `kustomize` files for deploying storage components.
- `tasks` Contains template resources for all custom tasks.
