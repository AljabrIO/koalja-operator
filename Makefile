# Various simple defines
VERSION ?= dev
GOOS ?= linux
GOARCH ?= amd64
GOMOD=github.com/AljabrIO/koalja-operator

# Image URL to use all building/pushing image targets
MANAGERIMG ?= $(DOCKERNAMESPACE)/koalja-operator:$(VERSION)
PIPELINEAGENTIMG ?= $(DOCKERNAMESPACE)/koalja-pipeline-agent:$(VERSION)

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
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(GOOS)/$(GOARCH)/manager $(GOMOD)/cmd/manager

# Build pipeline_agent binary
pipeline_agent: generate fmt vet
	mkdir -p bin/$(GOOS)/$(GOARCH)/
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(GOOS)/$(GOARCH)/pipeline_agent $(GOMOD)/cmd/pipeline_agent

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

# Build the docker image for the pipeline agent
docker-pipeline_agent: check-vars pipeline_agent
	docker build --build-arg=GOARCH=$(GOARCH) -f ./cmd/pipeline_agent/Dockerfile -t $(PIPELINEAGENTIMG) .
	docker push $(PIPELINEAGENTIMG)
	#@echo "updating kustomize image patch file for pipeline_agent resource"
	#sed -i'' -e 's@image: .*@image: '"$(PIPELINEAGENTIMG)"'@' ./config/default/manager_image_patch.yaml
