# Koalja Pipeline Operator

## Building

```bash
export DOCKERNAMESPACE=<your-docker-hub-account-name>
make docker-manager
```

## Deployment

```bash
# Install the CRD
make install
# Deploy operator
make deploy
```
