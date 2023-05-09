VERSION ?= $(shell git describe --tags --long --always)
# increment this in a commit or PR when you make changes to the helm chart
HELM_CHART_VERSION ?= 0.1.0

DOCKER_REPO ?= registry.local:5000/githedgehog/das-boot
HELM_CHART_REPO ?= registry.local:5000/githedgehog/helm-charts

MKFILE_DIR := $(shell echo $(dir $(abspath $(lastword $(MAKEFILE_LIST)))) | sed 's#/$$##')
BUILD_DIR := $(MKFILE_DIR)/build
BUILD_ARTIFACTS_DIR := $(BUILD_DIR)/artifacts
BUILD_COVERAGE_DIR := $(BUILD_DIR)/coverage
BUILD_DOCKER_DIR := $(BUILD_DIR)/docker
BUILD_HELM_DIR := $(BUILD_DIR)/helm
DEV_DIR := $(MKFILE_DIR)/dev

SRC_COMMON := $(shell find $(MKFILE_DIR)/pkg -type f -name "*.go")
SRC_HHDEVID := $(shell find $(MKFILE_DIR)/cmd/hhdevid -type f -name "*.go")
SRC_STAGE0 := $(shell find $(MKFILE_DIR)/cmd/stage0 -type f -name "*.go")
SRC_STAGE1 := $(shell find $(MKFILE_DIR)/cmd/stage1 -type f -name "*.go")
SRC_STAGE2 := $(shell find $(MKFILE_DIR)/cmd/stage2 -type f -name "*.go")
SRC_HHAGENTPROV := $(shell find $(MKFILE_DIR)/cmd/hedgehog-agent-provisioner -type f -name "*.go")
SRC_SEEDER := $(shell find $(MKFILE_DIR)/cmd/seeder -type f -name "*.go")

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

build: hhdevid stage0 stage1 stage2 hedgehog-agent-provisioner seeder ## Builds all golang binaries for all platforms: hhdevid, stage0, stage1, stage2, hedgehog-agent-provisioner and seeder

clean: hhdevid-clean stage0-clean stage1-clean stage2-clean hedgehog-agent-provisioner-clean seeder-clean docker-clean helm-clean ## Cleans all golang binaries for all platforms: hhdevid, stage0, stage1, stage2, hedgehog-agent-provisioner and seeder, as well as the seeder docker image and the packaged helm chart

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

seeder: $(BUILD_ARTIFACTS_DIR)/seeder $(BUILD_DOCKER_DIR)/seeder ## Builds the 'seeder' for x86_64

# TODO: removing "-buildmode=pie" from the ldflags for now, as it requires a dynamic linker
$(BUILD_ARTIFACTS_DIR)/seeder: $(SRC_COMMON) $(SRC_SEEDER) $(SEEDER_DEPS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/seeder -ldflags="-w -s -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/seeder

$(BUILD_DOCKER_DIR)/seeder: $(BUILD_ARTIFACTS_DIR)/seeder
	cp -v $(BUILD_ARTIFACTS_DIR)/seeder $(BUILD_DOCKER_DIR)/seeder

.PHONY: seeder-clean
seeder-clean: ## Cleans the 'seeder' x86_64 golang binary
	rm -v $(BUILD_ARTIFACTS_DIR)/seeder || true
	rm -v $(BUILD_DOCKER_DIR)/seeder || true

dev-init-oci-certs: $(DEV_OCI_REPO_CERT_FILES) ## Generates a local CA and server certificate to use for our docker registry

$(DEV_OCI_REPO_CERT_FILES) &:
	$(MKFILE_DIR)/scripts/init_repo_certs.sh

.PHONY: run-docker-registry
run-docker-registry: dev-init-oci-certs ## Runs a local docker registry in a docker container. NOTE: This is forwarded to the control plane as well!
	$(MKFILE_DIR)/scripts/run_registry.sh

.PHONY: docker
docker: seeder ## Builds a docker image for the seeder
	cd $(BUILD_DOCKER_DIR) && docker build -t $(DOCKER_REPO):latest .

.PHONY: docker-clean
docker-clean: ## Removes the docker image from the local docker images
	docker rmi $(DOCKER_REPO):latest || true

.PHONY: docker-push
docker-push: docker ## Builds AND pushes a docker image for the seeder
	@echo
	@[ "$(DOCKER_REPO)" = "registry.local:5000/githedgehog/das-boot" ] && $(MKFILE_DIR)/scripts/run_registry.sh || echo "Not trying to run local registry, different docker repository..."
	@echo
	docker push $(DOCKER_REPO):latest

.PHONY: helm
helm: ## Builds a helm chart for the seeder
	helm lint $(BUILD_HELM_DIR)
# TODO: at some point we need valid app versions too
#	helm package $(BUILD_HELM_DIR) --version $(HELM_CHART_VERSION) --app-version $(VERSION) -d $(BUILD_ARTIFACTS_DIR)
	helm package $(BUILD_HELM_DIR) --version $(HELM_CHART_VERSION) --app-version $(HELM_CHART_VERSION) -d $(BUILD_ARTIFACTS_DIR)

.PHONY: helm-clean
helm-clean: ## Cleans the packaged helm chart for the seeder from the artifacts build directory
	rm -v $(BUILD_ARTIFACTS_DIR)/das-boot-seeder-$(HELM_CHART_VERSION).tgz || true

.PHONY: helm-push
helm-push: helm ## Builds AND pushes the helm chart for the seeder
	helm push $(BUILD_ARTIFACTS_DIR)/das-boot-seeder-$(HELM_CHART_VERSION).tgz oci://$(HELM_CHART_REPO)

# Use this target only for local linting. In CI we use a dedicated github action
.PHONY: lint
lint: ## Runs golangci-lint (NOTE: target for local development only, used through github action in CI)
	golangci-lint run --verbose ./...

test: test-race test-cover ## Runs golang unit tests twice: for code coverage, and the second time with race detector

.PHONY: test-race
test-race: ## Runs golang unit tests with race detector
	@echo "Running tests with race detector..."
	go test -race ./cmd/... ./pkg/...
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
	controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=configs/crd/bases

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

.PHONY: init-control-node
init-control-node: ## Prepares a QEMU VM to run the control node
	$(MKFILE_DIR)/scripts/init_control_node.sh

.PHONY: run-control-node
run-control-node: ## Runs the control node VM
	$(MKFILE_DIR)/scripts/run_control_node.sh

.PHONY: run-control-node-tpm
run-control-node-tpm: ## Runs the software TPM for the control node VM (NOTE: not needed to run separately, will be started automatically)
	$(MKFILE_DIR)/scripts/run_control_node_tpm.sh

.PHONE: clean-control-node
clean-control-node: ## Deletes the control node VM and its supporting files
	rm -rvf $(DEV_DIR)/control-node-1 || true

.PHONY: access-control-node-kubeconfig
access-control-node-kubeconfig: ## Displays the kubeconfig to use to be able to reach the Kubernetes cluster (NOTE: 127.0.0.1 is fine, port-forwarding is used)
	@ssh -o GlobalKnownHostsFile=/dev/null -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i $(DEV_DIR)/control-node-1/core-ssh-key -p 2201 core@127.0.0.1 "sudo kubectl config view --raw=true" | tee $(DEV_DIR)/control-node-1/kubeconfig
	@chmod 600 $(DEV_DIR)/control-node-1/kubeconfig
	@echo
	@echo "NOTE: a copy is also stored now at $(DEV_DIR)/control-node-1/kubeconfig" 1>&2
	@echo "Run the following command in your shell to get access to it immediately:" 1>&2
	@echo 1>&2
	@echo "export KUBECONFIG=\"$(DEV_DIR)/control-node-1/kubeconfig\"" 1>&2

.PHONY: access-control-node-ssh
access-control-node-ssh: ## SSH into control node VM
	ssh -o GlobalKnownHostsFile=/dev/null -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i $(DEV_DIR)/control-node-1/core-ssh-key -p 2201 core@127.0.0.1

.PHONY: access-control-node-serial
access-control-node-serial: ## Access the serial console of the control node VM
	@echo "Use ^] to disconnect from serial console"
	socat -,rawer,escape=0x1d unix-connect:$(DEV_DIR)/control-node-1/serial.sock

.PHONY: access-control-node-vnc
access-control-node-vnc: ## Access the VGA display of the control node VM
	vncviewer unix $(DEV_DIR)/control-node-1/vnc.sock

.PHONY: access-control-node-monitor
access-control-node-monitor: ## Access the QEMU monitor (control interface) of the control node VM
	nc -U $(DEV_DIR)/control-node-1/monitor.sock

.PHONY: access-control-node-qnp
access-control-node-qnp:
	nc -U $(DEV_DIR)/control-node-1/qnp.sock

.PHONY: init-switch1
init-switch1: ## Prepares a QEMU VM to run switch1
	SSH_PORT="2211" NETDEVS="devid=eth0 mac=0c:20:12:fe:01:00 devid=eth1 mac=0c:20:12:fe:01:01 local_port=127.0.0.1:21011 dest_port=127.0.0.1:21001 devid=eth2 mac=0c:20:12:fe:01:02 local_port=127.0.0.1:21012 dest_port=127.0.0.1:21031" $(MKFILE_DIR)/scripts/init_switch.sh switch1

.PHONY: run-switch1
run-switch1: ## Runs the VM for switch1
	$(MKFILE_DIR)/scripts/run_switch.sh switch1

.PHONY: run-switch1-tpm
run-switch1-tpm: ## Runs the software TPM for th switch1 VM (NOTE: not needed to run separately, will be started automatically)
	$(MKFILE_DIR)/scripts/run_switch_tpm.sh switch1

.PHONE: clean-switch1
clean-switch1: ## Deletes the switch1 VM and its supporting files
	rm -rvf $(DEV_DIR)/switch1 || true

.PHONY: access-switch1-serial
access-switch1-serial: ## Access the serial console of the switch1 VM
	@echo "Use ^] to disconnect from serial console"
	socat -,rawer,escape=0x1d unix-connect:$(DEV_DIR)/switch1/serial.sock

.PHONY: access-switch1-ssh
access-switch1-ssh: ## SSH into switch1 VM (NOTE: requires a successful SONiC installation)
	@echo "Use password 'githedgehog' for our own SONiC VS builds (default)."
	@echo "Change the username in the Makefile to 'admin' for upstream SONiC VS builds. Password for this is 'YourPaSsWoRd'."
	ssh -o GlobalKnownHostsFile=/dev/null -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -p 2211 githedgehog@127.0.0.1

.PHONY: access-switch1-monitor
access-switch1-monitor: ## Access the QEMU monitor (control interface) of the switch1 VM
	nc -U $(DEV_DIR)/switch1/monitor.sock

.PHONY: access-switch1-qnp
access-switch1-qnp:
	nc -U $(DEV_DIR)/switch1/qnp.sock