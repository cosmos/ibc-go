#!/usr/bin/make -f

PACKAGES_NOSIMULATION=$(shell go list ./... | grep -v '/simulation')
PACKAGES_SIMTEST=$(shell go list ./... | grep '/simulation')
CHANGED_GO_FILES := $(shell git diff --name-only | grep .go$$ | grep -v pb.go)
ALL_GO_FILES := $(shell find . -regex ".*\.go$$" | grep -v pb.go)
VERSION := $(shell echo $(shell git describe --always) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
LEDGER_ENABLED ?= true
BINDIR ?= $(GOPATH)/bin
BUILDDIR ?= $(CURDIR)/build
SIMAPP = ./simapp
MOCKS_DIR = $(CURDIR)/tests/mocks
HTTPS_GIT := https://github.com/cosmos/ibc-go.git
DOCKER := $(shell which docker)
PROJECT_NAME = $(shell git remote get-url origin | xargs basename -s .git)

export GO111MODULE = on

# process build tags

build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += gcc
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=sim \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=simd \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)"

# DB backend selection
ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
endif
ifeq (badgerdb,$(findstring badgerdb,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=badgerdb
endif
# handle rocksdb
ifeq (rocksdb,$(findstring rocksdb,$(COSMOS_BUILD_OPTIONS)))
  CGO_ENABLED=1
  BUILD_TAGS += rocksdb
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=rocksdb
endif
# handle boltdb
ifeq (boltdb,$(findstring boltdb,$(COSMOS_BUILD_OPTIONS)))
  BUILD_TAGS += boltdb
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=boltdb
endif

ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -w -s
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))
BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
# check for nostrip option
ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif

#? all: Run tools build lint test
all: build lint test

# The below include contains the tools and runsim targets.
include contrib/devtools/Makefile

###############################################################################
###                                  Build                                  ###
###############################################################################

BUILD_TARGETS := build install

#? tidy-all: Run go mod tidy for all modules
tidy-all:
	./scripts/go-mod-tidy-all.sh

#? build: Build simapp and build_test_matrix
build: BUILD_ARGS=-o $(BUILDDIR)/

#? build-linux: Build simapp and build_test_matrix for GOOS=linux GOARCH=amd64
build-linux:
	GOOS=linux GOARCH=amd64 LEDGER_ENABLED=false $(MAKE) build

$(BUILD_TARGETS): go.sum $(BUILDDIR)/
	cd simapp && go $@ -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./...

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

.PHONY: build build-linux

#? distclean: Run `make clean`
distclean: clean 

#? clean: Clean some auto generated directories
clean:
	rm -rf \
    $(BUILDDIR)/ \
    artifacts/ \
    tmp-swagger-gen/

.PHONY: distclean clean

#? build-docker-wasm: Build wasm simapp with specified tag.
build-docker-wasm:
	./scripts/build-wasm-simapp-docker.sh $(tag)

build-docker-local:
	docker build -t ghcr.io/cosmos/ibc-go-simd:local --build-arg IBC_GO_VERSION=local . 

.PHONY: build-docker-wasm

###############################################################################
###                          Tools & Dependencies                           ###
###############################################################################

go.sum: go.mod
	echo "Ensure dependencies have not been modified ..." >&2
	go mod verify
	go mod tidy

#? python-install-deps: Install python dependencies
python-install-deps:
	@echo "Installing python dependencies..."
	@pip3 install --upgrade pip
	@pip3 install -r requirements.txt

###############################################################################
###                              Documentation                              ###
###############################################################################

#? godocs: Generate go documentation
godocs:
	@echo "--> Wait a few seconds and visit http://localhost:6060/pkg/github.com/cosmos/cosmos-sdk/types"
	godoc -http=:6060

#? build-docs: Build documentation
build-docs:
	@cd docs && npm ci && npm run build

#? serve-docs: Run docs server
serve-docs:
	@cd docs && npm run serve

# If the DOCS_VERSION variable is not set, display an error message and exit
ifndef DOCS_VERSION
#? tag-docs-version: Tag the docs version
tag-docs-version:
	@echo "Error: DOCS_VERSION is not set. Use 'make tag-docs-version DOCS_VERSION=<version>' to set it. For example: 'make tag-docs-version DOCS_VERSION=v8.0.x'"
	@exit 1
else
tag-docs-version:
	@cd docs && npm run docusaurus docs:version $(DOCS_VERSION)
endif

check-docs-links:
	@command -v lychee >/dev/null 2>&1 || { echo "ERROR: lychee is not installed (https://lychee.cli.rs/installation/)" >&2; exit 1; }
	@echo "Checking links in documentation..."
	@lychee --root-dir $(CURDIR)/docs/docs \
		--cache \
		--cache-exclude-status 429 \
		--max-cache-age 1w \
		--retry-wait-time 30 \
		--max-retries 25 \
		--max-concurrency 25 \
		--remap '($(CURDIR)/docs)(/docs/)(architecture/|events/)([^#]+?)(#[^#]+)?$$ $$1/$$3/$$4.md' \
		'./docs/docs'

lint-docs:
	@command -v markdownlint-cli2 >/dev/null 2>&1 || { \
	  echo "ERROR: markdownlint-cli2 is not installed" >&2; exit 1; \
	}
	@echo "Linting documentation..."
	@find docs/docs -type f -name '*.md' | xargs markdownlint-cli2

.PHONY: build-docs serve-docs tag-docs-version

###############################################################################
###                           Tests & Simulation                            ###
###############################################################################

# make init-simapp initializes a single local node network
# it is useful for testing and development
# Usage: make install && make init-simapp && simd start
# Warning: make init-simapp will remove all data in simapp home directory
#? init-simapp: Run scripts/init-simapp.sh
init-simapp:
	./scripts/init-simapp.sh

#? test: Run make test-unit
test: test-unit

#? test-all: Run all test
test-all: test-unit test-ledger-mock test-race test-cover

TEST_PACKAGES=./...
TEST_TARGETS := test-unit test-unit-amino test-unit-proto test-ledger-mock test-race test-ledger test-race

# Test runs-specific rules. To add a new test target, just add
# a new rule, customise ARGS or TEST_PACKAGES ad libitum, and
# append the new rule to the TEST_TARGETS list.
test-unit: ARGS=-tags='cgo ledger test_ledger_mock test_e2e'
test-unit-amino: ARGS=-tags='ledger test_ledger_mock test_amino'
test-ledger: ARGS=-tags='cgo ledger'
test-ledger-mock: ARGS=-tags='ledger test_ledger_mock'
test-race: ARGS=-race -tags='cgo ledger test_ledger_mock'
test-race: TEST_PACKAGES=$(PACKAGES_NOSIMULATION)
$(TEST_TARGETS): run-tests

# check-* compiles and collects tests without running them
# note: go test -c doesn't support multiple packages yet (https://github.com/golang/go/issues/15513)
CHECK_TEST_TARGETS := check-test-unit check-test-unit-amino
check-test-unit: ARGS=-tags='cgo ledger test_ledger_mock'
check-test-unit-amino: ARGS=-tags='ledger test_ledger_mock test_amino'
$(CHECK_TEST_TARGETS): EXTRA_ARGS=-run=none
$(CHECK_TEST_TARGETS): run-tests

ARGS += -tags "$(test_tags)"
#? run-tests: Runs the go test command for all modules
run-tests: 
	@ARGS="$(ARGS)" TEST_PACKAGES=$(TEST_PACKAGES) EXTRA_ARGS="$(EXTRA_ARGS)" python3 ./scripts/go-test-all.py

.PHONY: run-tests test test-all $(TEST_TARGETS)

#? test-sim-nondeterminism: Run non-determinism test for simapp
test-sim-nondeterminism:
	@echo "Running non-determinism test..."
	@go test -mod=readonly $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
		-NumBlocks=100 -BlockSize=200 -Commit=true -Period=0 -v -timeout 24h

test-sim-custom-genesis-fast:
	@echo "Running custom genesis simulation..."
	@echo "By default, ${HOME}/.gaiad/config/genesis.json will be used."
	@go test -mod=readonly $(SIMAPP) -run TestFullAppSimulation -Genesis=${HOME}/.gaiad/config/genesis.json \
		-Enabled=true -NumBlocks=100 -BlockSize=200 -Commit=true -Seed=99 -Period=5 -v -timeout 24h

test-sim-import-export: runsim
	@echo "Running application import/export simulation. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestAppImportExport

test-sim-after-import: runsim
	@echo "Running application simulation-after-import. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestAppSimulationAfterImport

test-sim-custom-genesis-multi-seed: runsim
	@echo "Running multi-seed custom genesis simulation..."
	@echo "By default, ${HOME}/.gaiad/config/genesis.json will be used."
	@$(BINDIR)/runsim -Genesis=${HOME}/.gaiad/config/genesis.json -SimAppPkg=$(SIMAPP) -ExitOnFail 400 5 TestFullAppSimulation

test-sim-multi-seed-long: runsim
	@echo "Running long multi-seed application simulation. This may take awhile!"
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 500 50 TestFullAppSimulation

test-sim-multi-seed-short: runsim
	@echo "Running short multi-seed application simulation. This may take awhile!"
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 10 TestFullAppSimulation

test-sim-benchmark-invariants:
	@echo "Running simulation invariant benchmarks..."
	@go test -mod=readonly $(SIMAPP) -benchmem -bench=BenchmarkInvariants -run=^$ \
	-Enabled=true -NumBlocks=1000 -BlockSize=200 \
	-Period=1 -Commit=true -Seed=57 -v -timeout 24h

.PHONY: \
test-sim-nondeterminism \
test-sim-custom-genesis-fast \
test-sim-import-export \
test-sim-after-import \
test-sim-custom-genesis-multi-seed \
test-sim-multi-seed-short \
test-sim-multi-seed-long \
test-sim-benchmark-invariants

SIM_NUM_BLOCKS ?= 500
SIM_BLOCK_SIZE ?= 200
SIM_COMMIT ?= true

#? test-sim-benchmark: Run application benchmark
test-sim-benchmark:
	@echo "Running application benchmark for numBlocks=$(SIM_NUM_BLOCKS), blockSize=$(SIM_BLOCK_SIZE). This may take awhile!"
	@go test -mod=readonly -benchmem -run=^$$ $(SIMAPP) -bench ^BenchmarkFullAppSimulation$$  \
		-Enabled=true -NumBlocks=$(SIM_NUM_BLOCKS) -BlockSize=$(SIM_BLOCK_SIZE) -Commit=$(SIM_COMMIT) -timeout 24h

#? test-sim-profile: Run application benchmark and output cpuprofile, memprofile
test-sim-profile:
	@echo "Running application benchmark for numBlocks=$(SIM_NUM_BLOCKS), blockSize=$(SIM_BLOCK_SIZE). This may take awhile!"
	@go test -mod=readonly -benchmem -run=^$$ $(SIMAPP) -bench ^BenchmarkFullAppSimulation$$ \
		-Enabled=true -NumBlocks=$(SIM_NUM_BLOCKS) -BlockSize=$(SIM_BLOCK_SIZE) -Commit=$(SIM_COMMIT) -timeout 24h -cpuprofile cpu.out -memprofile mem.out

.PHONY: test-sim-profile test-sim-benchmark

#? test-cover: Run contrib/test_cover.sh
test-cover:
	@export VERSION=$(VERSION); bash -x contrib/test_cover.sh
.PHONY: test-cover

#? benchmark: Run benchmark tests
benchmark:
	@go test -mod=readonly -bench=. $(PACKAGES_NOSIMULATION)
.PHONY: benchmark

###############################################################################
###                                Linting                                  ###
###############################################################################

#? lint: Run golangci-lint on all modules
lint:
	@echo "--> Running linter"
	@./scripts/go-lint-all.sh --timeout=15m

#? lint-fix: Run golangci-lint and fix issues on all modules
lint-fix:
	@echo "--> Running linter"
	@./scripts/go-lint-all.sh --fix

#? format: Run gofumpt
format:
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./tests/mocks/*" -not -name '*.pb.go' -not -name '*.pb.gw.go' | xargs gofumpt -w
.PHONY: format

.PHONY: lint lint-fix format

###############################################################################
###                                Protobuf                                 ###
###############################################################################

protoVer=0.17.1
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

#? proto-all: Format, lint and generate Protobuf files
proto-all: proto-format proto-lint proto-gen

#? proto-gen: Generate Protobuf files
proto-gen:
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh

#? proto-swagger-gen: Generate Protobuf Swagger
proto-swagger-gen:
	@echo "Generating Protobuf Swagger"
	@$(protoImage) sh ./scripts/protoc-swagger-gen.sh

#? proto-format: Format Protobuf files
proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

#? proto-lint: Lint Protobuf files
proto-lint:
	@$(protoImage) buf lint --error-format=json

#? proto-check-breaking: Check if Protobuf file contains breaking changes
proto-check-breaking:
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

#? proto-update-deps: Update Protobuf dependencies
proto-update-deps:
	@echo "Updating Protobuf dependencies"
	$(DOCKER) run --rm -v $(CURDIR)/proto:/workspace --workdir /workspace $(protoImageName) buf mod update

.PHONY: proto-all proto-gen proto-gen-any proto-swagger-gen proto-format proto-lint proto-check-breaking proto-update-deps

#? help: Get more info on make commands
help: Makefile
	@echo " Choose a command run in "$(PROJECT_NAME)":"
	@sed -n 's/^#?//p' $< | column -t -s ':' |  sort | sed -e 's/^/ /'
.PHONY: help
