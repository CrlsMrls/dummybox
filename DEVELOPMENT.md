# Development

## Quick Start with Make

The project includes a Makefile for consistent development workflows:

- `make help`: List all available commands
- `make dev`: Run the server in development mode
- `make test`: Run all tests
- `make build`: Build the binary locally with version injection
- `make version`: Check the current version


## Running the Server in Development Mode

To start the server in development mode using `Make`
```bash
make dev
```

Or directly with Go
```bash
go run .
```

The server will start on port 8080 by default. You can then access http://localhost:8080/

To stop the server, press `Ctrl+C`.


## Testing

### Running Tests

To run all tests in the project:

```bash
go test ./...
```

To run tests with verbose output:

```bash
go test -v ./...
```

To run tests for a specific package:

```bash
go test ./server
go test ./config
go test ./logger
go test ./cmd/delay
```

To run only unit tests (exclude integration tests):

```bash
go test -short ./...
```

To run only integration tests for a specific command:

```bash
go test ./cmd/delay -run Integration
```

Run tests with coverage

```bash
go test -cover ./...
```

Run tests and generate coverage report
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Structure

The project is tested using separation between unit and integration tests:

1. **Unit Tests**: Test individual handler functions in isolation
   - Located in `cmd/*/` packages
   - Package name matches the implementation (e.g., `package delay`)
   - Fast, focused on handler logic only

2. **Integration Tests**: Test endpoints through the full server stack
   - Located in `cmd/*/` packages but use `package *_test` naming
   - Test authentication, middleware, routing, and handler together
   - Use server test utilities from `server/testing.go`


## Versioning

How it works:

- **Version Storage**: The current version is stored in the `VERSION` file at the project root
- **Environment Variable**: The build process reads the VERSION file: `VERSION=$(cat VERSION)`, Make automatically sets this when running build/publish commands
- **Build-time Injection**: Ko uses `-ldflags -X github.com/crlsmrls/dummybox/cmd.Version={{.Env.VERSION}}` to replace the default "development" value in `cmd.Version` with the actual version from the file
- **Runtime Access**: The version is accessible via the `/version` HTTP endpoint
- **Container Tagging**: Container images are automatically tagged with the version from the `VERSION` file

## Updating the version:

Make automatically increments version

```bash
make bump-patch    # 0.0.1 -> 0.0.2
make bump-minor    # 0.0.1 -> 0.1.0  
make bump-major    # 0.0.1 -> 1.0.0
```

Or manually update the VERSION file (not recommended):

```bash
echo "0.1.0" > VERSION
```
The new version will be used in the next build

## Building and Publishing

### Prerequisites

Install `ko` build tool:

```bash
# Using Make
make install-ko

# Or manually
go install github.com/google/ko@latest
```

### Building

```bash
# Build binary locally with version injection
make build

# Build and publish container image to registry
make publish

# Build and publish container image locally  
make publish-local
```

### Complete Release Workflow

```bash
# Run tests, build, publish, and create git tag
make release
```

### Manual Commands (if not using Make)

```bash
# Build and publish the container image
VERSION=$(cat VERSION) KO_DOCKER_REPO=crlsmrls ko publish -B -t $(cat VERSION) -t latest . 
VERSION=$(cat VERSION) KO_DOCKER_REPO=ko.local ko publish -B -t $(cat VERSION) -t latest . 
```

## Quick Testing Reference

### Common Test Commands

```bash
# Run all tests
make test
go test ./...

# Run tests for delay command (unit + integration)
go test ./cmd/delay -v

# Run only integration tests for delay
go test ./cmd/delay -run Integration -v

# Run server tests
go test ./server -v

# Run tests with coverage
go test -cover ./...

# Run tests and generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test File Patterns

- **Unit Tests**: `*_test.go` in same package as implementation
- **Integration Tests**: `*_integration_test.go` with `package *_test`
- **Test Utilities**: `testing.go` in server package

### Key Testing Principles

1. **Unit tests** for handler logic (fast, isolated)
2. **Integration tests** for full request flow (authentication, middleware, etc.)
3. **Use test utilities** instead of exposing internal server methods
4. **Co-locate tests** with their respective commands for better organization
5. **External package testing** (`package *_test`) for integration tests to avoid circular imports

