# Koalja Pipeline Operator

## Building

```bash
export DOCKERNAMESPACE=<your-docker-hub-account-name>
make docker
```

## Deployment

```bash
# Install the CRD
make install
# Prepare dev namespace
kubectl create ns koalja-dev
# Deploy operator
make deploy
```
