.PHONY: setup build run clean test fmt lint release snapshot

# Install development tools and git hooks
setup:
	@command -v lefthook >/dev/null || (echo "Install lefthook: brew install lefthook" && exit 1)
	lefthook install

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X github.com/salmonumbrella/fastmail-cli/internal/cmd.Version=$(VERSION) -X github.com/salmonumbrella/fastmail-cli/internal/cmd.Commit=$(COMMIT) -X github.com/salmonumbrella/fastmail-cli/internal/cmd.Date=$(DATE)"

# Build the binary
build:
	go build $(LDFLAGS) -o ./bin/fastmail ./cmd/fastmail

# Run the CLI
run: build
	./bin/fastmail $(ARGS)

# Clean build artifacts
clean:
	rm -rf ./bin ./dist

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Lint code
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download
	go mod tidy

# Install locally
install: build
	cp ./bin/fastmail /usr/local/bin/fastmail

# Create a release snapshot (for testing)
snapshot:
	goreleaser release --snapshot --clean

# Create a release (requires GITHUB_TOKEN)
release:
	goreleaser release --clean
