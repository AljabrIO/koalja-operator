# Various simple defines
VERSION ?= dev
GOOS ?= linux
GOARCH ?= amd64

# Image URL to use all building/pushing image targets
MANAGERIMG ?= $(DOCKERNAMESPACE)/koalja-operator:$(VERSION)

all: check-vars test manager

# Check given variables
.PHONY: check-vars
check-vars:
ifndef DOCKERNAMESPACE
	@echo "DOCKERNAMESPACE must be set"
	@exit 1
endif
	@echo "Using docker namespace: $(DOCKERNAMESPACE)"

# Run tests
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	mkdir -p bin/$(GOOS)/$(GOARCH)/
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(GOOS)/$(GOARCH)/manager github.com/AljabrIO/koalja-operator/cmd/manager

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./cmd/manager/main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	@kustomize build config/default | kubectl delete -f - || true
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
	go generate ./pkg/... ./cmd/...

# Build the docker image for the manager (aka operator)
docker-manager: check-vars manager
	docker build --build-arg=GOARCH=$(GOARCH) -f ./cmd/manager/Dockerfile -t $(MANAGERIMG) .
	docker push $(MANAGERIMG)
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"$(MANAGERIMG)"'@' ./config/default/manager_image_patch.yaml
