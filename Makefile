VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_FLAGS := -ldflags "\
  -X github.com/liftedinit/manifest-node-exporter/cmd/manifest-node-exporter.Version=$(VERSION) \
  -X github.com/liftedinit/manifest-node-exporter/cmd/manifest-current-supply-exporter.Version=$(VERSION)" \
  -tags manifest

#### Build ####
build: ## Build the binary
	@echo "--> Building development binary (version: $(VERSION))"
	@go build $(BUILD_FLAGS) -tags manifest_node_exporter -o bin/manifest-node-exporter ./cmd-bin/manifest-node-exporter/main.go
	@go build $(BUILD_FLAGS) -tags manifest_current_supply_exporter -o bin/manifest-current-supply-exporter ./cmd-bin/manifest-current-supply-exporter/main.go

.PHONY: build

#### Test ####
test: ## Run tests
	@echo "--> Running tests"
	@go test -v -short -race ./...

.PHONY: test

#### Coverage ####
COV_ROOT="/tmp/manifest-node-exporter-coverage"
COV_UNIT="${COV_ROOT}/unit"
COV_E2E="${COV_ROOT}/e2e"
COV_PKG="github.com/liftedinit/manifest-node-exporter/..."

coverage: ## Run tests with coverage
	@echo "--> Creating GOCOVERDIR"
	@mkdir -p ${COV_UNIT} ${COV_E2E}
	@echo "--> Cleaning up coverage files, if any"
	@rm -rf ${COV_UNIT}/* ${COV_E2E}/*
	@echo "--> Running short tests with coverage"
	@go test -v -short -timeout 30m -race -covermode=atomic -cover -cpu=$$(nproc) -coverpkg=${COV_PKG} ./... -args -test.gocoverdir="${COV_UNIT}"
	@echo "--> Running end-to-end tests with coverage"
	@go test -v -race -timeout 30m -race -covermode=atomic -cover -cpu=$$(nproc) -coverpkg=${COV_PKG} -args -test.gocoverdir="${COV_E2E}"
	@echo "--> Merging coverage reports"
	@go tool covdata merge -i=${COV_UNIT},${COV_E2E} -o ${COV_ROOT}
	@echo "--> Converting binary coverage report to text format"
	@go tool covdata textfmt -i=${COV_ROOT} -o ${COV_ROOT}/coverage-merged.out
	@echo "--> Generating coverage report"
	@go tool cover -func=${COV_ROOT}/coverage-merged.out
	@echo "--> Generating HTML coverage report"
	@go tool cover -html=${COV_ROOT}/coverage-merged.out -o coverage.html
	@echo "--> Coverage report available at coverage.html"
	@echo "--> Cleaning up coverage files"
	@rm -rf ${COV_UNIT}/* ${COV_E2E}/*
	@echo "--> Running coverage complete"

####  Linting  ####
golangci_lint_cmd=golangci-lint
golangci_version=v2.1.6

lint:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run ./... --timeout 15m

lint-fix:
	@echo "--> Running linter and fixing issues"
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run ./... --fix --timeout 15m

.PHONY: lint lint-fix

#### FORMAT ####
goimports_version=v0.32.0

format: ## Run formatter (goimports)
	@echo "--> Running goimports"
	@go install golang.org/x/tools/cmd/goimports@$(goimports_version)
	@find . -name '*.go' -exec goimports -w -local github.com/liftedinit/manifest-node-exporter {} \;

.PHONY: format

#### GOVULNCHECK ####
govulncheck_version=v1.1.3

govulncheck: ## Run govulncheck
	@echo "--> Running govulncheck"
	@go install golang.org/x/vuln/cmd/govulncheck@$(govulncheck_version)
	@govulncheck ./...

.PHONY: govulncheck
