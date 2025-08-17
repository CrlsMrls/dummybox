# DummyBox

DummyBox is a lightweight HTTP server designed for testing and validating containerized environments like Kubernetes clusters. It provides various endpoints to simulate different application behaviors, making it ideal for testing monitoring systems, cluster configurations, and network connectivity.

DummyBox serves as a "dummy" application that can:
- **Mock HTTP responses** with custom status codes and delays
- **Expose system information** including environment variables and container details
- **Generate Prometheus metrics** for monitoring system validation
- **Generate logs** with configurable log levels and structured JSON format


Perfect for testing:
- 🔍 **Monitoring systems** (logs, metrics, alerts)
- 🎛️ **Cluster configurations** (networking, RBAC, autoscaling based on workload)
- 📊 **Observability stack** (Prometheus, Grafana, alerting)


## Quick Start

### Running with Podman/Docker

Run with default settings (replace with `docker` if preferred):

```bash
podman run -p 8080:8080 crlsmrls/dummybox:latest
```

Run with custom configuration
```bash
podman run -p 8080:8080 \
  -e DUMMYBOX_LOG_LEVEL=debug \
  -e DUMMYBOX_AUTH_TOKEN=mysecret \
  crlsmrls/dummybox:latest
```

Alternatively, it can be run locally after building the binary:

```bash
go build -o dummybox .
./dummybox --port 8080 --log-level debug --auth-token mysecret
```

## Configuration

DummyBox can be configured through environment variables, command-line flags, or a JSON configuration file.

All configuration options can be set using environment variables with the `DUMMYBOX_` prefix:

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `DUMMYBOX_PORT` | `8080` | HTTP server listening port |
| `DUMMYBOX_LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warning`, `error`) |
| `DUMMYBOX_METRICS_PATH` | `/metrics` | Prometheus metrics endpoint path |
| `DUMMYBOX_TLS_CERT_FILE` | `` | Path to TLS certificate file (enables HTTPS) |
| `DUMMYBOX_TLS_KEY_FILE` | `` | Path to TLS private key file |
| `DUMMYBOX_AUTH_TOKEN` | `` | Authentication token for protected endpoints |
| `DUMMYBOX_CONFIG_FILE` | `` | Path to JSON configuration file |


The configuration file is a JSON object with the same fields as the environment variables, but without the `DUMMYBOX_` prefix and in camelCase. Create a JSON configuration file:

```json
{
  "port": 8080,
  "log-level": "info",
  "metrics-path": "/metrics",
  "tls-cert-file": "/path/to/cert.pem",
  "tls-key-file": "/path/to/key.pem",
  "auth-token": "your-secret-token"
}
```

The same variables can be set with command-line flags, lowercase and hyphenated - e.g. `DUMMYBOX_LOG_LEVEL` becomes `--log-level`.

Run with help flag to see all options:

```bash
dummybox --help
```

## Monitoring and Observability

DummyBox exposes metrics for Prometheus scraping at the configured metrics path (default `/metrics`).

It also uses structured JSON logging with configurable log levels. 

### Health Checks

- `/healthz`: Always returns 200 OK (liveness probe)
- `/readyz`: Returns 200 OK when application is ready (readiness probe)

## Command Endpoints

DummyBox provides several command endpoints for testing different scenarios. These endpoints require authentication if `auth-token` is configured.

> 📖 **For detailed documentation with examples and use cases, see [COMMANDS.md](COMMANDS.md)**

### `/delay` - Response Delay Simulation

The delay endpoint allows you to introduce configurable delays in responses, useful for testing timeout handling, latency scenarios, and load balancing behavior.

**Purpose**: Simulate network latency, slow backends, or timeout scenarios

**HTTP Methods**: `GET`, `POST`

**Authentication**: Required if `auth-token` is configured

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `duration` | integer | No | `0` | Delay duration in seconds (0-300) |
| `code` | integer | No | `200` | HTTP status code to return (100-599) |
| `format` | string | No | `json` | Response format: `json` or `text` |

**Usage Examples**:

```bash
# GET request with 2-second delay and 201 status code
curl "http://localhost:8080/delay?duration=2&code=201&token=your-token"

# POST request with JSON body
curl -X POST "http://localhost:8080/delay" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: your-token" \
  -d '{"duration": 5, "code": 500}'

# Text format response
curl "http://localhost:8080/delay?duration=1&format=text&token=your-token"
```

**Response Example**:
```json
{
  "duration": 2,
  "code": 201,
  "message": "Delayed for 2 seconds with status code 201"
}
```

### `/log` - Log Generation

The log endpoint generates log messages with configurable levels and content, useful for testing log aggregation systems, monitoring alerts, and structured logging pipelines.

**Purpose**: Generate test logs for validating logging systems, alerts, and log processing pipelines

**HTTP Methods**: `GET`, `POST`

**Authentication**: Required if `auth-token` is configured

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `level` | string | No | `info` | Log level: `info`, `warning`, or `error` |
| `size` | string | No | `short` | Message size: `short`, `medium`, or `long` |
| `message` | string | No | (generated) | Custom message to log (URL-encoded for GET) |
| `interval` | integer | No | `0` | Seconds between logs (0 = log once, max 3600) |

**Usage Examples**:

```bash
# Generate a single info log with short message
curl "http://localhost:8080/log?level=info&size=short&token=your-token"

# Generate error logs every 30 seconds
curl "http://localhost:8080/log?level=error&interval=30&token=your-token"

# POST request with custom message
curl -X POST "http://localhost:8080/log" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: your-token" \
  -d '{"level": "warning", "size": "medium", "message": "Custom test message", "interval": 0}'

# Generate logs with correlation ID
curl "http://localhost:8080/log?level=info&token=your-token" \
  -H "X-Correlation-ID: test-scenario-123"
```

**Response Example**:
```json
{
  "level": "warning",
  "size": "medium",
  "message": "Database connection pool initialized with 10 connections, ready to serve requests",
  "interval": 0,
  "status": "log generation started"
}
```

**Log Output Behavior**:
- `info` level: Logs to stdout
- `warning` and `error` levels: Log to stderr
- All logs are in structured JSON format
- Background interval logging runs in separate goroutines
- Logs include correlation ID if provided via `X-Correlation-ID` header

**Message Sizes**:
- `short`: Simple operational messages (8-20 words)
- `medium`: Detailed status messages (15-50 words)  
- `long`: Comprehensive diagnostic messages (100+ words)

## Security

DummyBox is meant to be used in controlled testing environments. However, it includes basic security features to prevent unauthorized access to certain endpoints.

The authentication mechanism is a simple token-based system. The token can be set via the `DUMMYBOX_AUTH_TOKEN` environment variable or `--auth-token` command-line flag.

When `auth-token` is configured, protected endpoints require authentication via token. You can provide the token in two ways:

1. **Query parameter**: `?token=your-secret-token`
2. **HTTP header**: `X-Auth-Token: your-secret-token`

Protected endpoints include all command endpoints (`/delay`, `/log`, etc.).

**DummyBox** - Making container testing simple and reliable! 🚀
