SCRIPTDIR := $(shell pwd)
ROOTDIR := $(shell cd $(SCRIPTDIR) && pwd)

# Various simple defines
VERSION ?= dev
GOOS ?= linux
GOARCH ?= amd64
GOMOD=github.com/AljabrIO/koalja-operator

# Image URL to use all building/pushing image targets
OPERATORIMG ?= $(DOCKERNAMESPACE)/koalja-operator:$(VERSION)
AGENTSIMG ?= $(DOCKERNAMESPACE)/koalja-agents:$(VERSION)
SERVICESIMG ?= $(DOCKERNAMESPACE)/koalja-services:$(VERSION)
TASKSIMG ?= $(DOCKERNAMESPACE)/koalja-tasks:$(VERSION)

# Frontend defines
FRONTENDDIR := $(ROOTDIR)/frontend
FRONTENDBUILDIMG := koalja-operator-frontend-builder
FRONTENDSOURCES := $(shell find $(FRONTENDDIR)/src -name '*.js' -not -path './test/*')

# Tools
GOASSETSBUILDER := $(shell go env GOPATH)/bin/go-assets-builder$(shell go env GOEXE)

# Sources
SOURCES := $(shell find . -name '*.go') $(shell find . -name '*.proto')

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
build: manager agents services tasks

# Build manager binary
manager: bin/$(GOOS)/$(GOARCH)/manager

bin/$(GOOS)/$(GOARCH)/manager: $(SOURCES) generate fmt vet
	mkdir -p bin/$(GOOS)/$(GOARCH)/
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(GOOS)/$(GOARCH)/manager $(GOMOD)/cmd/manager

# Build agents binary
agents: bin/$(GOOS)/$(GOARCH)/agents
 
bin/$(GOOS)/$(GOARCH)/agents: $(SOURCES) generate fmt vet frontend/assets.go
	mkdir -p bin/$(GOOS)/$(GOARCH)/
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(GOOS)/$(GOARCH)/agents $(GOMOD)/cmd/agents

# Build services binary
services: bin/$(GOOS)/$(GOARCH)/services

bin/$(GOOS)/$(GOARCH)/services: $(SOURCES) generate fmt vet
	mkdir -p bin/$(GOOS)/$(GOARCH)/
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(GOOS)/$(GOARCH)/services $(GOMOD)/cmd/services

# Build tasks binary
tasks: bin/$(GOOS)/$(GOARCH)/tasks

bin/$(GOOS)/$(GOARCH)/tasks: $(SOURCES) generate fmt vet
	mkdir -p bin/$(GOOS)/$(GOARCH)/
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(GOOS)/$(GOARCH)/tasks $(GOMOD)/cmd/tasks

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
	go-to-protobuf \
		--keep-gogoproto \
		--proto-import="vendor" \
		--proto-import="third_party/googleapis" \
		--apimachinery-packages -k8s.io/apimachinery/pkg/util/intstr,-k8s.io/apimachinery/pkg/api/resource,-k8s.io/apimachinery/pkg/runtime/schema,-k8s.io/apimachinery/pkg/runtime,-k8s.io/apimachinery/pkg/apis/meta/v1,-k8s.io/apimachinery/pkg/apis/meta/v1beta1,-k8s.io/apimachinery/pkg/apis/testapigroup/v1,+sigs.k8s.io/controller-runtime/pkg/runtime/scheme,-k8s.io/api/core/v1 \
		--packages=github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1
	go generate ./pkg/... ./cmd/...

frontend/assets.go: $(FRONTENDSOURCES) $(FRONTENDDIR)/Dockerfile.build
	cd $(FRONTENDDIR) && docker build -t $(FRONTENDBUILDIMG) -f Dockerfile.build $(FRONTENDDIR)
	@mkdir -p $(FRONTENDDIR)/build
	docker run --rm \
		-u $(shell id -u):$(shell id -g) \
		-v $(FRONTENDDIR)/build:/usr/code/build \
		-v $(FRONTENDDIR)/public:/usr/code/public:ro \
		-v $(FRONTENDDIR)/src:/usr/code/src:ro \
		$(FRONTENDBUILDIMG)
	$(GOASSETSBUILDER) -s /frontend/build/ -o frontend/assets.go -p frontend frontend/build

# Build & push all docker images
docker: docker-build docker-push docker-patch-config

# Build the docker image for the programs
docker-build: check-vars build
	docker build --build-arg=GOARCH=$(GOARCH) -f ./docker/agents/Dockerfile -t $(AGENTSIMG) .
	docker build --build-arg=GOARCH=$(GOARCH) -f ./docker/operator/Dockerfile -t $(OPERATORIMG) .
	docker build --build-arg=GOARCH=$(GOARCH) -f ./docker/services/Dockerfile -t $(SERVICESIMG) .
	docker build --build-arg=GOARCH=$(GOARCH) -f ./docker/tasks/Dockerfile -t $(TASKSIMG) .

# Push docker images
docker-push: docker-build
	docker push $(AGENTSIMG)
	docker push $(OPERATORIMG)
	docker push $(SERVICESIMG)
	docker push $(TASKSIMG)

# Set image IDs in patch files
docker-patch-config:
	mkdir -p config/default/$(VERSION)
	sed -e 's!image: .*!image: '"$(shell docker inspect --format="{{index .RepoDigests 0}}" $(AGENTSIMG))"'!' ./config/default/pipeline_agent_image_patch.yaml > ./config/default/$(VERSION)/pipeline_agent_image_patch.yaml
	sed -e 's!image: .*!image: '"$(shell docker inspect --format="{{index .RepoDigests 0}}" $(AGENTSIMG))"'!' ./config/default/stub_link_agent_image_patch.yaml > ./config/default/$(VERSION)/stub_link_agent_image_patch.yaml
	sed -e 's!image: .*!image: '"$(shell docker inspect --format="{{index .RepoDigests 0}}" $(AGENTSIMG))"'!' ./config/default/task_agent_image_patch.yaml > ./config/default/$(VERSION)/task_agent_image_patch.yaml
	sed -e 's!image: .*!image: '"$(shell docker inspect --format="{{index .RepoDigests 0}}" $(OPERATORIMG))"'!' ./config/default/manager_image_patch.yaml > ./config/default/$(VERSION)/manager_image_patch.yaml
	sed -e 's!image: .*!image: '"$(shell docker inspect --format="{{index .RepoDigests 0}}" $(SERVICESIMG))"'!' ./config/default/stub_event_registry_image_patch.yaml > ./config/default/$(VERSION)/stub_event_registry_image_patch.yaml
	sed -e 's!image: .*!image: '"$(shell docker inspect --format="{{index .RepoDigests 0}}" $(SERVICESIMG))"'!' ./config/default/local_fs_service_image_patch.yaml > ./config/default/$(VERSION)/local_fs_service_image_patch.yaml
	sed -e 's!image: .*!image: '"$(shell docker inspect --format="{{index .RepoDigests 0}}" $(TASKSIMG))"'!' ./config/default/filedrop_executor_image_patch.yaml > ./config/default/$(VERSION)/filedrop_executor_image_patch.yaml
	cd config/default/$(VERSION) && echo "namespace: koalja-$(VERSION)" > kustomization.yaml && kustomize edit add base ".." && kustomize edit add patch "*_patch.yaml"

bootstrap:
	go get github.com/jessevdk/go-assets-builder
