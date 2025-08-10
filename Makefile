# DummyBox Makefile
.PHONY: help build run test test-verbose clean version bump-patch bump-minor bump-major publish publish-local dev install-deps

# Default target
help: ## Show this help message
	@echo "DummyBox - Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Variables
VERSION := $(shell cat VERSION)
GO_VERSION := $(shell go version | cut -d ' ' -f 3)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Development
dev: ## Run the server in development mode
	go run .

run: dev ## Alias for dev

# Testing
test: ## Run all tests
	go test ./...

test-verbose: ## Run all tests with verbose output
	go test -v ./...

test-coverage: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Building
build: ## Build the binary locally
	go build -ldflags="-X github.com/crlsmrls/dummybox/cmd.Version=$(VERSION) \
		-X github.com/crlsmrls/dummybox/cmd.BuildDate=$(BUILD_DATE) \
		-X github.com/crlsmrls/dummybox/cmd.GoVersion=$(GO_VERSION) \
		-X github.com/crlsmrls/dummybox/cmd.GitCommit=$(GIT_COMMIT)" \
		-o bin/dummybox .

# Version management
version: ## Show current version
	@echo "Current version: $(VERSION)"
	@echo "Go version: $(GO_VERSION)"
	@echo "Build date: $(BUILD_DATE)"
	@echo "Git commit: $(GIT_COMMIT)"

bump-patch: ## Increment patch version (x.y.Z)
	@current=$$(cat VERSION | cut -d'-' -f1); \
	patch=$$(echo $$current | cut -d'.' -f3); \
	new_patch=$$((patch + 1)); \
	new_version=$$(echo $$current | sed "s/\.[0-9]*$$/.$${new_patch}/"); \
	echo "$$new_version" > VERSION; \
	echo "Version bumped to: $$new_version"

bump-minor: ## Increment minor version (x.Y.z)
	@current=$$(cat VERSION | cut -d'-' -f1); \
	minor=$$(echo $$current | cut -d'.' -f2); \
	new_minor=$$((minor + 1)); \
	new_version=$$(echo $$current | sed "s/\.[0-9]*\.[0-9]*$$/.$${new_minor}.0/"); \
	echo "$$new_version" > VERSION; \
	echo "Version bumped to: $$new_version"

bump-major: ## Increment major version (X.y.z)
	@current=$$(cat VERSION | cut -d'-' -f1); \
	major=$$(echo $$current | cut -d'.' -f1); \
	new_major=$$((major + 1)); \
	new_version="$${new_major}.0.0"; \
	echo "$$new_version" > VERSION; \
	echo "Version bumped to: $$new_version"

# Publishing with Ko
publish: ## Build and publish container image to registry
	@echo "Publishing version $(VERSION) to registry..."
	VERSION=$(VERSION) KO_DOCKER_REPO=crlsmrls ko publish -B -t $(VERSION) -t latest .

publish-local: ## Build and publish container image locally
	@echo "Publishing version $(VERSION) locally..."
	VERSION=$(VERSION) KO_DOCKER_REPO=ko.local ko publish -B -t $(VERSION) -t latest .

# Dependencies
install-deps: ## Install project dependencies
	go mod download
	go mod tidy

install-ko: ## Install ko build tool
	go install github.com/google/ko@latest

# Cleanup
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

# Linting and formatting
fmt: ## Format Go code
	go fmt ./...

lint: ## Run golangci-lint (requires golangci-lint to be installed)
	golangci-lint run

# Git operations
tag: ## Create and push git tag for current version
	@echo "Creating git tag v$(VERSION)..."
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)

# Complete release workflow
release: test build publish tag ## Complete release: test, build, publish, and tag
	@echo "Release $(VERSION) completed successfully!"
