# Service Container Injection Config

## Annotated Value Registry

The annotated value registry maintains a list of all AnnotatedValue's that
are created in a Pipeline.

It runs, as a separate container, in the same Pod as the Pipeline agent.

The following annotated-value-registry implementations are available:

- `stub-annotatedvalue-registry` The default, in-memory implementation.
- `arangodb-annotatedvalue-registry` An implementation that stores values in an ArangoDB database.

The `arangodb-annotatedvalue-registry` implementation requires some additional
configuration in the form of a `ConfigMap` and a `Secret` for database authentication
credentials.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: arangodb-annotatedvalue-registry-config
data:
  endpoints: "https://<hostname-of-database>:8529"
  database: "<nameOfDatabase>"
```

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: arangodb-annotatedvalue-registry-auth
data:
  username: <base64 encoded username>
  password: <base64 password>
```

## FileSystem Service

The filesystem service provides Task Executors with a volume to store
files on.

The following filesystem service implementations are available:

- `local-fs` The default implementation that stores files on the disks of the nodes of the kubernetes cluster.
- `s3-fs` An implementation that stores files in an S3 compatible object store.
