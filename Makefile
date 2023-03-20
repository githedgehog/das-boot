VERSION := $(shell git describe --all --tags --long)

MKFILE_DIR := $(shell echo $(dir $(abspath $(lastword $(MAKEFILE_LIST)))) | sed 's#/$$##')
BUILD_DIR := $(MKFILE_DIR)/build
BUILD_ARTIFACTS_DIR := $(BUILD_DIR)/artifacts
BUILD_COVERAGE_DIR := $(BUILD_DIR)/coverage
DEV_DIR := $(MKFILE_DIR)/dev

SRC_COMMON := $(shell find $(MKFILE_DIR)/pkg -type f -name "*.go")
SRC_HHDEVID := $(shell find $(MKFILE_DIR)/cmd/hhdevid -type f -name "*.go")
SRC_STAGE0 := $(shell find $(MKFILE_DIR)/cmd/stage0 -type f -name "*.go")
SRC_STAGE1 := $(shell find $(MKFILE_DIR)/cmd/stage1 -type f -name "*.go")
SRC_STAGE2 := $(shell find $(MKFILE_DIR)/cmd/stage2 -type f -name "*.go")
SRC_SEEDER := $(shell find $(MKFILE_DIR)/cmd/seeder -type f -name "*.go")

SEEDER_ARTIFACTS_DIR := $(MKFILE_DIR)/pkg/seeder/artifacts/embedded/artifacts

SEEDER_DEPS := $(SEEDER_ARTIFACTS_DIR)/stage0-amd64  $(SEEDER_ARTIFACTS_DIR)/stage0-arm64  $(SEEDER_ARTIFACTS_DIR)/stage0-arm
SEEDER_DEPS += $(SEEDER_ARTIFACTS_DIR)/stage1-amd64  $(SEEDER_ARTIFACTS_DIR)/stage1-arm64  $(SEEDER_ARTIFACTS_DIR)/stage1-arm
SEEDER_DEPS += $(SEEDER_ARTIFACTS_DIR)/stage2-amd64  $(SEEDER_ARTIFACTS_DIR)/stage2-arm64  $(SEEDER_ARTIFACTS_DIR)/stage2-arm

DEV_FILES := $(DEV_DIR)/seeder/client-ca-cert.pem
DEV_FILES += $(DEV_DIR)/seeder/client-ca-key.pem
DEV_FILES += $(DEV_DIR)/seeder/config-ca-cert.pem
DEV_FILES += $(DEV_DIR)/seeder/config-ca-key.pem
DEV_FILES += $(DEV_DIR)/seeder/config-cert.pem
DEV_FILES += $(DEV_DIR)/seeder/config-key.pem
DEV_FILES += $(DEV_DIR)/seeder/seeder.yaml
DEV_FILES += $(DEV_DIR)/seeder/server-ca-cert.pem
DEV_FILES += $(DEV_DIR)/seeder/server-ca-key.pem
DEV_FILES += $(DEV_DIR)/seeder/server-cert.pem
DEV_FILES += $(DEV_DIR)/seeder/server-key.pem

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

all: generate build ## Runs 'generate' and 'build' targets

build: hhdevid stage0 stage1 stage2 seeder ## Builds all golang binaries for all platforms: hhdevid, stage0, stage1, stage2 and seeder

clean: hhdevid-clean stage0-clean stage1-clean stage2-clean seeder-clean ## Cleans all golang binaries for all platforms: hhdevid, stage0, stage1, stage2 and seeder

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

seeder:  $(BUILD_ARTIFACTS_DIR)/seeder ## Builds the 'seeder' for x86_64

$(BUILD_ARTIFACTS_DIR)/seeder: $(SRC_COMMON) $(SRC_SEEDER) $(SEEDER_DEPS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/seeder -ldflags="-w -s -buildmode=pie -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/seeder

.PHONY: seeder-clean
seeder-clean: ## Cleans the 'seeder' x86_64 golang binary
	rm -v $(BUILD_ARTIFACTS_DIR)/seeder || true

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

.PHONY: install-deps
install-deps: ## Installs development tool dependencies
	@echo "Installing mockgen..."
	go install github.com/golang/mock/mockgen@latest

dev-init-seeder: $(DEV_FILES) ## Generates development files (keys, certs, etc.pp.) for running the seeder locally

$(DEV_FILES) &:
	$(MKFILE_DIR)/scripts/init_seeder_dev.sh

.PHONY: dev-run-seeder
dev-run-seeder: dev-init-seeder seeder ## Runs the seeder locally
	$(BUILD_ARTIFACTS_DIR)/seeder --config $(DEV_DIR)/seeder/seeder.yaml