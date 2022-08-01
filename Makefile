NAME ?= app-migrator
OUTPUT = ./bin/$(NAME)
GO_SOURCES = $(shell find . -type f -name '*.go')
GOBIN ?= $(shell go env GOPATH)/bin
VERSION ?= $(shell ./hack/next-version)
GITSHA = $(shell git rev-parse HEAD)
GITDIRTY = $(shell git diff --quiet HEAD || echo "dirty")
LDFLAGS_VERSION = -X github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli.cliName=$(NAME) \
				  -X github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli.cliVersion=$(VERSION) \
				  -X github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli.cliGitSHA=$(GITSHA) \
				  -X github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli.cliGitDirty=$(GITDIRTY)

.PHONY: all
all: install test lint

.PHONY: clean
clean: ## Clean testcache and delete build output
	go clean -testcache
	@rm -rf bin/
	@rm -rf dist/

$(OUTPUT): $(GO_SOURCES)
	@echo "Building $(VERSION)"
	go build -o $(OUTPUT) -ldflags "$(LDFLAGS_VERSION)" ./cmd/$(NAME)

.PHONY: build
build: $(OUTPUT) ## Build the main binary

.PHONY: test
test: ## Run the unit tests
	go test -short ./...

.PHONY: test-export-org
test-export-org: ## Run export org integration tests only
	@rm -rf ./test/export-org-tests
	go test -timeout 15m -v -tags integration ./test/e2e/export_org_test.go

.PHONY: test-export-space
test-export-space: ## Run export space integration tests only
	@rm -rf ./test/export-space-tests
	go test -timeout 15m -v -tags integration ./test/e2e/export_space_test.go

.PHONY: test-import-org
test-import-org: ## Run import org integration tests only
	go test -timeout 15m -v -tags integration ./test/e2e/import_org_test.go

.PHONY: test-import-space
test-import-space: ## Run import space integration tests only
	go test -timeout 15m -v -tags integration ./test/e2e/import_space_test.go

.PHONY: test-e2e
test-e2e: test-export-org test-import-org test-export-space test-import-space ## Run all the integration tests

.PHONY: test-all
test-all: test test-e2e ## Run all the tests including long running ones [default timeout is 15min]

test-bench: ## Run all the benchmark tests
	go test -bench=. -benchmem ./...

.PHONY: install
install: build ## Copy build to GOPATH/bin
	@cp $(OUTPUT) $(GOBIN)
	@echo "[OK] CLI binary installed under $(GOBIN)"

.PHONY: coverage
coverage: ## Run the tests with coverage and race detection
	go test -v --race -coverprofile=c.out -covermode=atomic ./...

.PHONY: report
report: ## Show coverage in an html report
	go tool cover -html=c.out -o coverage.html

.PHONY: generate
generate: ## Generate fakes
	go generate ./...

.PHONY: clean-docs
clean-docs: ## Delete the generated docs
	mkdir -p docs/
	rm -f docs/*.md

.PHONY: docs
docs: clean-docs ## Generate documentation
	go run cmd/generate_docs/main.go

.PHONY: release
release: $(GO_SOURCES) ## Cross-compile binary for various operating systems
	@mkdir -p dist
	GOOS=darwin   GOARCH=amd64 go build -ldflags "$(LDFLAGS_VERSION)" -o $(OUTPUT)     ./cmd/$(NAME) && tar -czf dist/$(NAME)-darwin-amd64.tgz  -C bin . && rm -f $(OUTPUT)
	GOOS=linux    GOARCH=amd64 go build -ldflags "$(LDFLAGS_VERSION)" -o $(OUTPUT)     ./cmd/$(NAME)  && tar -czf dist/$(NAME)-linux-amd64.tgz  -C bin . && rm -f $(OUTPUT)
	GOOS=windows  GOARCH=amd64 go build -ldflags "$(LDFLAGS_VERSION)" -o $(OUTPUT).exe ./cmd/$(NAME)  && zip -rj  dist/$(NAME)-windows-amd64.zip   bin   && rm -f $(OUTPUT).exe

.PHONY: lint-prepare
lint-prepare:
	@echo "Installing latest golangci-lint"
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s latest
	@echo "[OK] golangci-lint installed"

.PHONY: lint
lint: lint-prepare ## Run the golangci linter
	./bin/golangci-lint run

.PHONY: tidy
tidy: ## Remove unused dependencies
	go mod tidy

.PHONY: list
list: ## Print the current module's dependencies.
	go list -m all

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## Print help for each make target
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'