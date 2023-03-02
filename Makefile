VERSION := $(shell git describe --all --tags --long)

MKFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
BUILD_DIR := $(MKFILE_DIR)/build
BUILD_ARTIFACTS_DIR := $(BUILD_DIR)/artifacts
BUILD_COVERAGE_DIR := $(BUILD_DIR)/coverage

SRC_COMMON := $(shell find $(MKFILE_DIR)/pkg)
SRC_HHDEVID := $(shell find $(MKFILE_DIR)/cmd/hhdevid)
SRC_STAGE0 := $(shell find $(MKFILE_DIR)/cmd/stage0)
SRC_STAGE1 := $(shell find $(MKFILE_DIR)/cmd/stage1)
SRC_STAGE2 := $(shell find $(MKFILE_DIR)/cmd/stage2)
SRC_SEEDER := $(shell find $(MKFILE_DIR)/cmd/seeder)

SEEDER_DEPS := $(BUILD_ARTIFACTS_DIR)/stage0-amd64  $(BUILD_ARTIFACTS_DIR)/stage0-arm64  $(BUILD_ARTIFACTS_DIR)/stage0-arm
SEEDER_DEPS += $(BUILD_ARTIFACTS_DIR)/stage1-amd64  $(BUILD_ARTIFACTS_DIR)/stage1-arm64  $(BUILD_ARTIFACTS_DIR)/stage1-arm
SEEDER_DEPS += $(BUILD_ARTIFACTS_DIR)/stage2-amd64  $(BUILD_ARTIFACTS_DIR)/stage2-arm64  $(BUILD_ARTIFACTS_DIR)/stage2-arm

all: generate build

build: hhdevid stage0 stage1 stage2 seeder

clean: hhdevid-clean stage0-clean stage1-clean stage2-clean seeder-clean

hhdevid:  $(BUILD_ARTIFACTS_DIR)/hhdevid-amd64  $(BUILD_ARTIFACTS_DIR)/hhdevid-arm64  $(BUILD_ARTIFACTS_DIR)/hhdevid-arm

 $(BUILD_ARTIFACTS_DIR)/hhdevid-amd64: $(SRC_COMMON) $(SRC_HHDEVID)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/hhdevid-amd64 -ldflags="-buildmode=pie -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/hhdevid

 $(BUILD_ARTIFACTS_DIR)/hhdevid-arm64: $(SRC_COMMON) $(SRC_HHDEVID)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_ARTIFACTS_DIR)/hhdevid-arm64 -ldflags="-buildmode=pie -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/hhdevid

 $(BUILD_ARTIFACTS_DIR)/hhdevid-arm: $(SRC_COMMON) $(SRC_HHDEVID)
# -buildmode=pie breaks here? Why? 
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o $(BUILD_ARTIFACTS_DIR)/hhdevid-arm -ldflags="-X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/hhdevid

.PHONY: hhdevid-clean
hhdevid-clean:
	rm -v $(BUILD_ARTIFACTS_DIR)/hhdevid-amd64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/hhdevid-arm64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/hhdevid-arm || true

stage0:  $(BUILD_ARTIFACTS_DIR)/stage0-amd64  $(BUILD_ARTIFACTS_DIR)/stage0-arm64  $(BUILD_ARTIFACTS_DIR)/stage0-arm

 $(BUILD_ARTIFACTS_DIR)/stage0-amd64: $(SRC_COMMON) $(SRC_STAGE0)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/stage0-amd64 -ldflags="-buildmode=pie -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage0

 $(BUILD_ARTIFACTS_DIR)/stage0-arm64: $(SRC_COMMON) $(SRC_STAGE0)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_ARTIFACTS_DIR)/stage0-arm64 -ldflags="-buildmode=pie -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage0

 $(BUILD_ARTIFACTS_DIR)/stage0-arm: $(SRC_COMMON) $(SRC_STAGE0)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o $(BUILD_ARTIFACTS_DIR)/stage0-arm -ldflags="-X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage0

.PHONY: stage0-clean
stage0-clean:
	rm -v $(BUILD_ARTIFACTS_DIR)/stage0-amd64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage0-arm64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage0-arm || true

stage1:  $(BUILD_ARTIFACTS_DIR)/stage1-amd64  $(BUILD_ARTIFACTS_DIR)/stage1-arm64  $(BUILD_ARTIFACTS_DIR)/stage1-arm

 $(BUILD_ARTIFACTS_DIR)/stage1-amd64: $(SRC_COMMON) $(SRC_STAGE1)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/stage1-amd64 -ldflags="-buildmode=pie -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage1

 $(BUILD_ARTIFACTS_DIR)/stage1-arm64: $(SRC_COMMON) $(SRC_STAGE1)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_ARTIFACTS_DIR)/stage1-arm64 -ldflags="-buildmode=pie -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage1

 $(BUILD_ARTIFACTS_DIR)/stage1-arm: $(SRC_COMMON) $(SRC_STAGE1)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o $(BUILD_ARTIFACTS_DIR)/stage1-arm -ldflags="-X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage1

.PHONY: stage1-clean
stage1-clean:
	rm -v $(BUILD_ARTIFACTS_DIR)/stage1-amd64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage1-arm64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage1-arm || true

stage2:  $(BUILD_ARTIFACTS_DIR)/stage2-amd64  $(BUILD_ARTIFACTS_DIR)/stage2-arm64  $(BUILD_ARTIFACTS_DIR)/stage2-arm

 $(BUILD_ARTIFACTS_DIR)/stage2-amd64: $(SRC_COMMON) $(SRC_STAGE2)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/stage2-amd64 -ldflags="-buildmode=pie -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage2

 $(BUILD_ARTIFACTS_DIR)/stage2-arm64: $(SRC_COMMON) $(SRC_STAGE2)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_ARTIFACTS_DIR)/stage2-arm64 -ldflags="-buildmode=pie -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage2

 $(BUILD_ARTIFACTS_DIR)/stage2-arm: $(SRC_COMMON) $(SRC_STAGE2)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o $(BUILD_ARTIFACTS_DIR)/stage2-arm -ldflags="-X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/stage2

.PHONY: stage2-clean
stage2-clean:
	rm -v $(BUILD_ARTIFACTS_DIR)/stage2-amd64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage2-arm64 || true
	rm -v $(BUILD_ARTIFACTS_DIR)/stage2-arm || true

seeder:  $(BUILD_ARTIFACTS_DIR)/seeder

 $(BUILD_ARTIFACTS_DIR)/seeder: $(SRC_COMMON) $(SRC_SEEDER) $(SEEDER_DEPS)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_ARTIFACTS_DIR)/seeder -ldflags="-buildmode=pie -X 'go.githedgehog.com/dasboot/pkg/version.Version=$(VERSION)'" ./cmd/seeder

.PHONY: seeder-clean
seeder-clean:
	rm -v $(BUILD_ARTIFACTS_DIR)/seeder || true

# Use this target only for local linting. In CI we use a dedicated github action
.PHONY: lint
lint:
	golangci-lint run --verbose ./...

.PHONY: test
test:
	@echo "Running tests with race detector..."
	go test -race ./cmd/... ./pkg/...
	@echo
	@echo "Running tests for code coverage..."
	go test -cover -covermode=count -coverprofile $(BUILD_COVERAGE_DIR)/coverage.profile ./cmd/... ./pkg/...
	go tool cover -func=$(BUILD_COVERAGE_DIR)/coverage.profile -o=$(BUILD_COVERAGE_DIR)/coverage.out
	go tool cover -html=$(BUILD_COVERAGE_DIR)/coverage.profile -o=$(BUILD_COVERAGE_DIR)/coverage.html
	@echo
	@echo -n "Total Code Coverage: $(shell tail -n 1 $(BUILD_COVERAGE_DIR)/coverage.out | awk '{ print $$3 }')"
	@echo

.PHONY: generate
generate:
	go generate -v ./...

.PHONY: install-deps
install-deps:
	@echo "Installing mockgen..."
	go install github.com/golang/mock/mockgen@latest