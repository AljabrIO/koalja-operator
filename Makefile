# Various simple defines
VERSION ?= dev
GOOS ?= linux
GOARCH ?= amd64
GOMOD=github.com/AljabrIO/koalja-operator

# Image URL to use all building/pushing image targets
MANAGERIMG ?= $(DOCKERNAMESPACE)/koalja-operator:$(VERSION)
PIPELINEAGENTIMG ?= $(DOCKERNAMESPACE)/koalja-pipeline-agent:$(VERSION)

all: check-vars build test

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

# Build programs
build: manager pipeline_agent

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
	@kustomize build config/default/$(VERSION) | kubectl delete -f - || true
	kustomize build config/default/$(VERSION) | kubectl apply -f -

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

# Build & push all docker images
docker: docker-build docker-push docker-patch-config

# Build the docker image for the programs
docker-build: check-vars manager pipeline_agent
	docker build --build-arg=GOARCH=$(GOARCH) -f ./cmd/manager/Dockerfile -t $(MANAGERIMG) .
	docker build --build-arg=GOARCH=$(GOARCH) -f ./cmd/pipeline_agent/Dockerfile -t $(PIPELINEAGENTIMG) .

# Push docker images
docker-push: docker-build
	docker push $(MANAGERIMG)
	docker push $(PIPELINEAGENTIMG)

# Set image IDs in patch files
docker-patch-config:
	mkdir -p config/default/$(VERSION)
	sed -e 's!image: .*!image: '"$(shell docker inspect --format="{{index .RepoDigests 0}}" $(MANAGERIMG))"'!' ./config/default/manager_image_patch.yaml > ./config/default/$(VERSION)/manager_image_patch.yaml
	sed -e 's!image: .*!image: '"$(shell docker inspect --format="{{index .RepoDigests 0}}" $(PIPELINEAGENTIMG))"'!' ./config/default/pipeline_agent_image_patch.yaml > ./config/default/$(VERSION)/pipeline_agent_image_patch.yaml
	cd config/default/$(VERSION) && echo "namespace: koalja-$(VERSION)" > kustomization.yaml && kustomize edit add base ".." && kustomize edit add patch "*_patch.yaml"
