.PHONY: build test test-short lint fmt vet vulncheck clean dev setup docker docker-run helm-lint release-dry docs docs-serve

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"
BIN := bin/agentledger

## build: Compile the binary
build:
	go build $(LDFLAGS) -o $(BIN) ./cmd/agentledger

## test: Run all tests with race detection and coverage
test:
	go test -race -cover -count=1 ./...

## test-short: Run fast tests only (skip integration tests)
test-short:
	go test -race -short -count=1 ./...

## test-coverage: Generate HTML coverage report
test-coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in your browser"

## lint: Run golangci-lint
lint:
	~/go/bin/golangci-lint run ./...

## lint-fix: Run golangci-lint with auto-fix
lint-fix:
	~/go/bin/golangci-lint run --fix ./...

## fmt: Format all Go files
fmt:
	gofmt -w .
	~/go/bin/goimports -w -local github.com/WDZ-Dev/agent-ledger .

## vet: Run go vet
vet:
	go vet ./...

## vulncheck: Check dependencies for known vulnerabilities
vulncheck:
	~/go/bin/govulncheck ./...

## tidy: Tidy and verify module dependencies
tidy:
	go mod tidy
	go mod verify

## clean: Remove build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

## dev: Run the proxy in development mode
dev: build
	./$(BIN) serve --config configs/agentledger.example.yaml

## setup: Install dev tools and git hooks
setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/evilmartians/lefthook/v2@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install golang.org/x/tools/cmd/goimports@latest
	~/go/bin/lefthook install

## docker: Build Docker image
docker:
	docker build -t agentledger:dev .

## docker-run: Build and run in Docker
docker-run: docker
	docker run --rm -p 8787:8787 -v agentledger-data:/data agentledger:dev

## helm-lint: Lint the Helm chart
helm-lint:
	helm lint deploy/helm/agentledger

## release-dry: GoReleaser dry run (snapshot)
release-dry:
	goreleaser release --snapshot --clean

## check: Run all checks (what CI runs)
check: fmt vet lint test vulncheck

## docs: Build documentation site
docs:
	pip install -q -r docs/requirements.txt
	mkdocs build --strict

## docs-serve: Serve documentation locally with live reload
docs-serve:
	pip install -q -r docs/requirements.txt
	mkdocs serve

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
