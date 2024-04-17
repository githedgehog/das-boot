VERSION ?= $(shell git describe --tags --dirty --always)

# using latest for now to keep compatible with other scripts
DOCKER_VERSION ?= latest

# using 0.1.0 for now to keep compatible with other scripts
HELM_CHART_VERSION ?= 0.1.0

DOCKER_REPO ?= registry.local:31000/githedgehog/das-boot
DOCKER_REPO_SEEDER ?= $(DOCKER_REPO)/das-boot-seeder
DOCKER_REPO_REGISTRATION_CONTROLLER ?= $(DOCKER_REPO)/das-boot-registration-controller

HELM_CHART_REPO ?= registry.local:31000/githedgehog/helm-charts

MKFILE_DIR := $(shell echo $(dir $(abspath $(lastword $(MAKEFILE_LIST)))) | sed 's\#/$$\#\#')
BUILD_DIR := $(MKFILE_DIR)/build
BUILD_ARTIFACTS_DIR := $(BUILD_DIR)/artifacts
BUILD_COVERAGE_DIR := $(BUILD_DIR)/coverage
BUILD_DOCKER_SEEDER_DIR := $(BUILD_DIR)/docker/seeder
BUILD_DOCKER_REGISTRATION_CONTROLLER_DIR := $(BUILD_DIR)/docker/registration-controller
BUILD_HELM_DIR := $(BUILD_DIR)/helm
DEV_DIR := $(MKFILE_DIR)/dev
HHFAB_DIR ?= $(MKFILE_DIR)/.hhfab

SRC_COMMON := $(shell find $(MKFILE_DIR)/pkg -type f -name "*.go")
SRC_K8S_COMMON := $(shell find $(MKFILE_DIR)/pkg/k8s -type f -name "*.go")
SRC_HHDEVID := $(shell find $(MKFILE_DIR)/cmd/hhdevid -type f -name "*.go")
SRC_STAGE0 := $(shell find $(MKFILE_DIR)/cmd/stage0 -type f -name "*.go")
SRC_STAGE1 := $(shell find $(MKFILE_DIR)/cmd/stage1 -type f -name "*.go")
SRC_STAGE2 := $(shell find $(MKFILE_DIR)/cmd/stage2 -type f -name "*.go")
SRC_HHAGENTPROV := $(shell find $(MKFILE_DIR)/cmd/hedgehog-agent-provisioner -type f -name "*.go")
SRC_SEEDER := $(shell find $(MKFILE_DIR)/cmd/seeder -type f -name "*.go")
SRC_REGISTRATION_CONTROLLER := $(shell find $(MKFILE_DIR)/cmd/registration-controller -type f -name "*.go")

SEEDER_ARTIFACTS_DIR := $(MKFILE_DIR)/pkg/seeder/artifacts/embedded/artifacts

SEEDER_DEPS := $(SEEDER_ARTIFACTS_DIR)/stage0-amd64  $(SEEDER_ARTIFACTS_DIR)/stage0-arm64  $(SEEDER_ARTIFACTS_DIR)/stage0-arm
SEEDER_DEPS += $(SEEDER_ARTIFACTS_DIR)/stage1-amd64  $(SEEDER_ARTIFACTS_DIR)/stage1-arm64  $(SEEDER_ARTIFACTS_DIR)/stage1-arm
SEEDER_DEPS += $(SEEDER_ARTIFACTS_DIR)/stage2-amd64  $(SEEDER_ARTIFACTS_DIR)/stage2-arm64  $(SEEDER_ARTIFACTS_DIR)/stage2-arm
SEEDER_DEPS += $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-amd64  $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm64  $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm

DEV_SEEDER_FILES := $(DEV_DIR)/seeder/client-ca-cert.pem
DEV_SEEDER_FILES += $(DEV_DIR)/seeder/client-ca-key.pem
DEV_SEEDER_FILES += $(DEV_DIR)/seeder/config-ca-cert.pem
DEV_SEEDER_FILES += $(DEV_DIR)/seeder/config-ca-key.pem
DEV_SEEDER_FILES += $(DEV_DIR)/seeder/config-cert.pem
DEV_SEEDER_FILES += $(DEV_DIR)/seeder/config-key.pem
DEV_SEEDER_FILES += $(DEV_DIR)/seeder/seeder.yaml
DEV_SEEDER_FILES += $(DEV_DIR)/seeder/server-ca-cert.pem
DEV_SEEDER_FILES += $(DEV_DIR)/seeder/server-ca-key.pem
DEV_SEEDER_FILES += $(DEV_DIR)/seeder/server-cert.pem
DEV_SEEDER_FILES += $(DEV_DIR)/seeder/server-key.pem

DEV_OCI_REPO_CERT_FILES := $(DEV_DIR)/oci/oci-repo-ca-key.pem
DEV_OCI_REPO_CERT_FILES += $(DEV_DIR)/oci/oci-repo-ca-cert.pem
DEV_OCI_REPO_CERT_FILES += $(DEV_DIR)/oci/server-key.pem
DEV_OCI_REPO_CERT_FILES += $(DEV_DIR)/oci/server-cert.pem

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

all: generate build ## Runs 'generate' and 'build' targets

build: hhdevid stage0 stage1 stage2 hedgehog-agent-provisioner seeder registration-controller ## Builds all golang binaries for all platforms: hhdevid, stage0, stage1, stage2, hedgehog-agent-provisioner, seeder and registration-controller

clean: hhdevid-clean stage0-clean stage1-clean stage2-clean hedgehog-agent-provisioner-clean seeder-clean registration-controller-clean docker-clean helm-clean ## Cleans all golang binaries for all platforms: hhdevid, stage0, stage1, stage2, hedgehog-agent-provisioner, seeder and registration-controller, as well as the seeder docker image and the packaged helm chart

hhdevid:  $(BUILD_ARTIFACTS_DIR)/hhdevid-amd64  $(BUILD_ARTIFACTS_DIR)/hhdevid-arm64  $(BUILD_ARTIFACTS_DIR)/hhdevid-arm ## Builds 'hhdevid' for all platforms

$(BUILD_ARTIFACTS_DIR)/hhdevid-amd64: $(SRC_COMMON) $(SRC_HHDEVID)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/hhdevid-amd64 -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/hhdevid

$(BUILD_ARTIFACTS_DIR)/hhdevid-arm64: $(SRC_COMMON) $(SRC_HHDEVID)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_ARTIFACTS_DIR)/hhdevid-arm64 -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/hhdevid

$(BUILD_ARTIFACTS_DIR)/hhdevid-arm: $(SRC_COMMON) $(SRC_HHDEVID)
# breaks here? Why?
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o $(BUILD_ARTIFACTS_DIR)/hhdevid-arm -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/hhdevid

.PHONY: hhdevid-clean
hhdevid-clean: ## Cleans all 'hhdevid' golang binaries
	rm -v $(BUILD_ARTIFACTS_DIR)/hhdevid-amd64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/hhdevid-arm64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/hhdevid-arm || true

stage0: $(SEEDER_ARTIFACTS_DIR)/stage0-amd64 $(SEEDER_ARTIFACTS_DIR)/stage0-arm64 $(SEEDER_ARTIFACTS_DIR)/stage0-arm ## Builds 'stage0' for all platforms

$(BUILD_ARTIFACTS_DIR)/stage0-amd64: $(SRC_COMMON) $(SRC_STAGE0)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/stage0-amd64 -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage0

$(BUILD_ARTIFACTS_DIR)/stage0-arm64: $(SRC_COMMON) $(SRC_STAGE0)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_ARTIFACTS_DIR)/stage0-arm64 -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage0

$(BUILD_ARTIFACTS_DIR)/stage0-arm: $(SRC_COMMON) $(SRC_STAGE0)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o $(BUILD_ARTIFACTS_DIR)/stage0-arm -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage0

$(SEEDER_ARTIFACTS_DIR)/stage0-amd64: $(BUILD_ARTIFACTS_DIR)/stage0-amd64
	cp -v $(BUILD_ARTIFACTS_DIR)/stage0-amd64 $(SEEDER_ARTIFACTS_DIR)/stage0-amd64

$(SEEDER_ARTIFACTS_DIR)/stage0-arm64: $(BUILD_ARTIFACTS_DIR)/stage0-arm64
	cp -v $(BUILD_ARTIFACTS_DIR)/stage0-arm64 $(SEEDER_ARTIFACTS_DIR)/stage0-arm64

$(SEEDER_ARTIFACTS_DIR)/stage0-arm: $(BUILD_ARTIFACTS_DIR)/stage0-arm
	cp -v $(BUILD_ARTIFACTS_DIR)/stage0-arm $(SEEDER_ARTIFACTS_DIR)/stage0-arm

.PHONY: stage0-clean
stage0-clean: ## Cleans all 'stage0' golang binaries
	rm -v $(SEEDER_ARTIFACTS_DIR)/stage0-amd64 || true
	rm -v $(SEEDER_ARTIFACTS_DIR)/stage0-arm64 || true
	rm -v $(SEEDER_ARTIFACTS_DIR)/stage0-arm || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage0-amd64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage0-arm64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage0-arm || true

stage1: $(SEEDER_ARTIFACTS_DIR)/stage1-amd64 $(SEEDER_ARTIFACTS_DIR)/stage1-arm64 $(SEEDER_ARTIFACTS_DIR)/stage1-arm ## Builds 'stage1' for all platforms

$(BUILD_ARTIFACTS_DIR)/stage1-amd64: $(SRC_COMMON) $(SRC_STAGE1)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/stage1-amd64 -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage1

$(BUILD_ARTIFACTS_DIR)/stage1-arm64: $(SRC_COMMON) $(SRC_STAGE1)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_ARTIFACTS_DIR)/stage1-arm64 -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage1

$(BUILD_ARTIFACTS_DIR)/stage1-arm: $(SRC_COMMON) $(SRC_STAGE1)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o $(BUILD_ARTIFACTS_DIR)/stage1-arm -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage1

$(SEEDER_ARTIFACTS_DIR)/stage1-amd64: $(BUILD_ARTIFACTS_DIR)/stage1-amd64
	cp -v $(BUILD_ARTIFACTS_DIR)/stage1-amd64 $(SEEDER_ARTIFACTS_DIR)/stage1-amd64

$(SEEDER_ARTIFACTS_DIR)/stage1-arm64: $(BUILD_ARTIFACTS_DIR)/stage1-arm64
	cp -v $(BUILD_ARTIFACTS_DIR)/stage1-arm64 $(SEEDER_ARTIFACTS_DIR)/stage1-arm64

$(SEEDER_ARTIFACTS_DIR)/stage1-arm: $(BUILD_ARTIFACTS_DIR)/stage1-arm
	cp -v $(BUILD_ARTIFACTS_DIR)/stage1-arm $(SEEDER_ARTIFACTS_DIR)/stage1-arm

.PHONY: stage1-clean
stage1-clean: ## Cleans all 'stage1' golang binaries
	rm -v $(SEEDER_ARTIFACTS_DIR)/stage1-amd64 || true
	rm -v $(SEEDER_ARTIFACTS_DIR)/stage1-arm64 || true
	rm -v $(SEEDER_ARTIFACTS_DIR)/stage1-arm || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage1-amd64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage1-arm64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage1-arm || true

stage2: $(SEEDER_ARTIFACTS_DIR)/stage2-amd64 $(SEEDER_ARTIFACTS_DIR)/stage2-arm64 $(SEEDER_ARTIFACTS_DIR)/stage2-arm ## Builds 'stage2' for all platforms

$(BUILD_ARTIFACTS_DIR)/stage2-amd64: $(SRC_COMMON) $(SRC_STAGE2)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/stage2-amd64 -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage2

$(BUILD_ARTIFACTS_DIR)/stage2-arm64: $(SRC_COMMON) $(SRC_STAGE2)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_ARTIFACTS_DIR)/stage2-arm64 -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage2

$(BUILD_ARTIFACTS_DIR)/stage2-arm: $(SRC_COMMON) $(SRC_STAGE2)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o $(BUILD_ARTIFACTS_DIR)/stage2-arm -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage2

$(SEEDER_ARTIFACTS_DIR)/stage2-amd64: $(BUILD_ARTIFACTS_DIR)/stage2-amd64
	cp -v $(BUILD_ARTIFACTS_DIR)/stage2-amd64 $(SEEDER_ARTIFACTS_DIR)/stage2-amd64

$(SEEDER_ARTIFACTS_DIR)/stage2-arm64: $(BUILD_ARTIFACTS_DIR)/stage2-arm64
	cp -v $(BUILD_ARTIFACTS_DIR)/stage2-arm64 $(SEEDER_ARTIFACTS_DIR)/stage2-arm64

$(SEEDER_ARTIFACTS_DIR)/stage2-arm: $(BUILD_ARTIFACTS_DIR)/stage2-arm
	cp -v $(BUILD_ARTIFACTS_DIR)/stage2-arm $(SEEDER_ARTIFACTS_DIR)/stage2-arm

.PHONY: stage2-clean
stage2-clean: ## Cleans all 'stage2' golang binaries
	rm -v $(SEEDER_ARTIFACTS_DIR)/stage2-amd64 || true
	rm -v $(SEEDER_ARTIFACTS_DIR)/stage2-arm64 || true
	rm -v $(SEEDER_ARTIFACTS_DIR)/stage2-arm || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage2-amd64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage2-arm64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage2-arm || true

hedgehog-agent-provisioner: $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-amd64 $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm64 $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm ## Builds 'hedgehog-agent-provisioner' for all platforms

$(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-amd64: $(SRC_COMMON) $(SRC_HHAGENTPROV)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-amd64 -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/hedgehog-agent-provisioner

$(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm64: $(SRC_COMMON) $(SRC_HHAGENTPROV)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm64 -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/hedgehog-agent-provisioner

$(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm: $(SRC_COMMON) $(SRC_HHAGENTPROV)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/hedgehog-agent-provisioner

$(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-amd64: $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-amd64
	cp -v $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-amd64 $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-amd64

$(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm64: $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm64
	cp -v $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm64 $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm64

$(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm: $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm
	cp -v $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm

.PHONY: hedgehog-agent-provisioner-clean
hedgehog-agent-provisioner-clean: ## Cleans all 'hedgehog-agent-provisioner' golang binaries
	rm -v $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-amd64 || true
	rm -v $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm64 || true
	rm -v $(SEEDER_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm || true
	rm -v $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-amd64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/hedgehog-agent-provisioner-arm || true

seeder: $(BUILD_ARTIFACTS_DIR)/seeder $(BUILD_DOCKER_SEEDER_DIR)/seeder ## Builds the 'seeder' for x86_64

# TODO: removing "-buildmode=pie" from the ldflags for now, as it requires a dynamic linker
$(BUILD_ARTIFACTS_DIR)/seeder: $(SRC_COMMON) $(SRC_SEEDER) $(SEEDER_DEPS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/seeder -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/seeder

$(BUILD_DOCKER_SEEDER_DIR)/seeder: $(BUILD_ARTIFACTS_DIR)/seeder
	cp -v $(BUILD_ARTIFACTS_DIR)/seeder $(BUILD_DOCKER_SEEDER_DIR)/seeder

.PHONY: seeder-clean
seeder-clean: ## Cleans the 'seeder' x86_64 golang binary
	rm -v $(BUILD_ARTIFACTS_DIR)/seeder || true
	rm -v $(BUILD_DOCKER_SEEDER_DIR)/seeder || true

registration-controller: $(BUILD_ARTIFACTS_DIR)/registration-controller $(BUILD_DOCKER_REGISTRATION_CONTROLLER_DIR)/registration-controller ## Builds the 'registration-controller' for x86_64

$(BUILD_ARTIFACTS_DIR)/registration-controller: $(SRC_K8S_COMMON) $(SRC_REGISTRATION_CONTROLLER)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/registration-controller -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/registration-controller

$(BUILD_DOCKER_REGISTRATION_CONTROLLER_DIR)/registration-controller: $(BUILD_ARTIFACTS_DIR)/registration-controller
	cp -v $(BUILD_ARTIFACTS_DIR)/registration-controller $(BUILD_DOCKER_REGISTRATION_CONTROLLER_DIR)/registration-controller

.PHONY: registration-controller-clean
registration-controller-clean: ## Cleans the 'registration-controller' x86_64 golang binary
	rm -v $(BUILD_ARTIFACTS_DIR)/registration-controller || true
	rm -v $(BUILD_DOCKER_REGISTRATION_CONTROLLER_DIR)/registration-controller || true

dev-init-oci-certs: $(DEV_OCI_REPO_CERT_FILES) ## Generates a local CA and server certificate to use for our docker registry

$(DEV_OCI_REPO_CERT_FILES) &:
	$(MKFILE_DIR)/scripts/init_repo_certs.sh

.PHONY: docker
docker: docker-seeder docker-registration-controller ## Builds all docker images

.PHONY: docker-clean
docker-clean: docker-seeder-clean docker-registration-controller-clean ## Removes all docker images from the local docker images

.PHONY: docker-push
docker-push: docker-seeder-push docker-registration-controller-push ## Builds AND pushes all docker images

.PHONY: docker-seeder
docker-seeder: seeder ## Builds a docker images for the seeder
	cd $(BUILD_DOCKER_SEEDER_DIR) && docker build -t $(DOCKER_REPO_SEEDER):$(DOCKER_VERSION) .

.PHONY: docker-seeder-clean
docker-seeder-clean: ## Removes the docker image from the local docker images
	docker rmi $(DOCKER_REPO_SEEDER):$(DOCKER_VERSION) || true

.PHONY: docker-seeder-push
docker-seeder-push: docker ## Builds AND pushes a docker image for the seeder
	skopeo copy --dest-tls-verify=false docker-daemon:$(DOCKER_REPO_SEEDER):$(DOCKER_VERSION) docker://$(DOCKER_REPO_SEEDER):$(DOCKER_VERSION)

.PHONY: docker-registration-controller
docker-registration-controller: registration-controller ## Builds a docker images for the registration-controller
	cd $(BUILD_DOCKER_REGISTRATION_CONTROLLER_DIR) && docker build -t $(DOCKER_REPO_REGISTRATION_CONTROLLER):$(DOCKER_VERSION) .

.PHONY: docker-registration-controller-clean
docker-registration-controller-clean: ## Removes the docker image from the local docker images
	docker rmi $(DOCKER_REPO_REGISTRATION_CONTROLLER):$(DOCKER_VERSION) || true

.PHONY: docker-registration-controller-push
docker-registration-controller-push: docker ## Builds AND pushes a docker image for the registration-controller
	skopeo copy --dest-tls-verify=false docker-daemon:$(DOCKER_REPO_REGISTRATION_CONTROLLER):$(DOCKER_VERSION) docker://$(DOCKER_REPO_REGISTRATION_CONTROLLER):$(DOCKER_VERSION)

.PHONY: helm
helm: ## Builds a helm chart for the seeder
	helm lint $(BUILD_HELM_DIR)/crds
	helm lint $(BUILD_HELM_DIR)/registration-controller
	helm lint $(BUILD_HELM_DIR)/seeder
# TODO: at some point we need valid app versions too
#	helm package $(BUILD_HELM_DIR) --version $(HELM_CHART_VERSION) --app-version $(VERSION) -d $(BUILD_ARTIFACTS_DIR)
	helm package $(BUILD_HELM_DIR)/crds --version $(HELM_CHART_VERSION) --app-version $(HELM_CHART_VERSION) -d $(BUILD_ARTIFACTS_DIR)
	helm package $(BUILD_HELM_DIR)/registration-controller --version $(HELM_CHART_VERSION) --app-version $(HELM_CHART_VERSION) -d $(BUILD_ARTIFACTS_DIR)
	helm package $(BUILD_HELM_DIR)/seeder --version $(HELM_CHART_VERSION) --app-version $(HELM_CHART_VERSION) -d $(BUILD_ARTIFACTS_DIR)

.PHONY: helm-clean
helm-clean: ## Cleans the packaged helm chart for the seeder from the artifacts build directory
	rm -v $(BUILD_ARTIFACTS_DIR)/das-boot-crds-$(HELM_CHART_VERSION).tgz || true
	rm -v $(BUILD_ARTIFACTS_DIR)/das-boot-registration-controller-$(HELM_CHART_VERSION).tgz || true
	rm -v $(BUILD_ARTIFACTS_DIR)/das-boot-seeder-$(HELM_CHART_VERSION).tgz || true

.PHONY: helm-push
helm-push: helm ## Builds AND pushes the helm chart for the seeder
	helm push --insecure-skip-tls-verify $(BUILD_ARTIFACTS_DIR)/das-boot-crds-$(HELM_CHART_VERSION).tgz oci://$(HELM_CHART_REPO)
	helm push --insecure-skip-tls-verify $(BUILD_ARTIFACTS_DIR)/das-boot-registration-controller-$(HELM_CHART_VERSION).tgz oci://$(HELM_CHART_REPO)
	helm push --insecure-skip-tls-verify $(BUILD_ARTIFACTS_DIR)/das-boot-seeder-$(HELM_CHART_VERSION).tgz oci://$(HELM_CHART_REPO)

.PHONY: update-third-party-helm
update-third-party-helm: ## Builds AND pushes the 3rd party helm charts
	$(MKFILE_DIR)/scripts/update_third_party_helm.sh

# Use this target only for local linting. In CI we use a dedicated github action
.PHONY: lint
lint: ## Runs golangci-lint (NOTE: target for local development only, used through github action in CI)
	golangci-lint run --verbose ./...

test: test-race test-cover ## Runs golang unit tests twice: for code coverage, and the second time with race detector

.PHONY: test-race
test-race: ## Runs golang unit tests with race detector
	@echo "Running tests with race detector..."
	CGO_ENABLED=1 go test -race ./cmd/... ./pkg/...
	@echo

.PHONY: test-cover
test-cover: ## Runs golang unit tests and generates code coverage information
	@echo "Running tests for code coverage..."
	go test -cover -covermode=count -coverprofile $(BUILD_COVERAGE_DIR)/coverage.profile ./cmd/... ./pkg/...
	go tool cover -func=$(BUILD_COVERAGE_DIR)/coverage.profile -o=$(BUILD_COVERAGE_DIR)/coverage.out
	go tool cover -html=$(BUILD_COVERAGE_DIR)/coverage.profile -o=$(BUILD_COVERAGE_DIR)/coverage.html
	@echo
	@echo -n "Total Code Coverage: "; tail -n 1 $(BUILD_COVERAGE_DIR)/coverage.out | awk '{ print $$3 }'
	@echo

.PHONY: generate
generate: ## Runs 'go generate'
	go generate -v ./...

.PHONY: k8s-manifests
k8s-manifests: ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	controller-gen rbac:roleName=registration-controller-role crd webhook paths="./..." \
		output:crd:artifacts:config=$(MKFILE_DIR)/build/helm/crds/templates \
		output:rbac:artifacts:config=$(MKFILE_DIR)/build/helm/registration-controller/templates

.PHONY: k8s-generate
k8s-generate: ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	controller-gen object:headerFile="docs/boilerplate.go.txt" paths="./..."

.PHONY: k8s-docs ## Build simple markdown documentation for all CRDs to be used as API docs
k8s-docs:
	crd-ref-docs --source-path=./pkg/k8s/api/ --config=./pkg/k8s/api/docs-config.yaml --renderer=markdown --output-path=./docs/k8s-api.md

.PHONY: install-deps
install-deps: ## Installs development tool dependencies
	@echo "Installing mockgen..."
	go install github.com/golang/mock/mockgen@latest
	@echo "Installing crd-ref-docs..."
	go install github.com/elastic/crd-ref-docs@latest
	@echo "Installing controller-gen..."
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

dev-init-seeder: $(DEV_SEEDER_FILES) ## Generates development files (keys, certs, etc.pp.) for running the seeder locally

$(DEV_SEEDER_FILES) &:
	$(MKFILE_DIR)/scripts/init_seeder_dev.sh

.PHONY: dev-run-seeder
dev-run-seeder: dev-init-seeder seeder ## Runs the seeder locally
	$(BUILD_ARTIFACTS_DIR)/seeder --config $(DEV_DIR)/seeder/seeder.yaml

.PHONY: init-vlab
init-vlab: ## Prepares a VLAB using our hhfab utility
ifneq "$(wildcard $(HHFAB_DIR) )" ""
	$(error "VLAB already initialized...")
endif
	hhfab wiring sample spine-leaf --chain-control-link=true > $(MKFILE_DIR)/wiring.yaml
	hhfab init -p vlab -w $(MKFILE_DIR)/wiring.yaml
	hhfab build

.PHONY: clean-vlab
clean-vlab: ## Deletes the VLAB and its supporting files
	rm -rvf $(HHFAB_DIR) || true
	rm -vf $(MKFILE_DIR)/wiring.yaml || true

.PHONY: run-vlab
run-vlab: ## Runs the VLAB using our hhfab utility
	hhfab vlab up

access-vlab-kubeconfig: ## Displays the kubeconfig to use to be able to reach the Kubernetes cluster (NOTE: 127.0.0.1 is fine, port-forwarding is used)
	@chmod 600 $(HHFAB_DIR)/vlab-vms/kubeconfig.yaml
	@cat $(HHFAB_DIR)/vlab-vms/kubeconfig.yaml
	@echo
	@echo "Run the following command in your shell to get access to it immediately:" 1>&2
	@echo 1>&2
	@echo "export KUBECONFIG=\"$(HHFAB_DIR)/vlab-vms/kubeconfig.yaml\"" 1>&2

.PHONY: access-vlab-ssh
access-vlab-ssh: ## SSH into the VLAB with VM selector
	hhfab vlab ssh

.PHONY: access-vlab-serial
access-vlab-serial: ## Access the serial console of the VLAB with VM selector
	hhfab vlab serial
