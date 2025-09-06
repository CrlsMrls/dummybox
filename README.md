# DummyBox

DummyBox is a lightweight HTTP server designed for testing and validating containerized environments like Kubernetes clusters. It provides various endpoints to simulate different application behaviors, making it ideal for testing monitoring systems, cluster configurations, and network connectivity. 

Commands that affect the system state (like memory allocation) can be protected with a simple token-based authentication mechanism.

A User Interface is also provided to interact with the endpoints through a web browser, although the main focus is on HTTP API usage.

DummyBox serves as a "dummy" application that can:
- **Mock HTTP responses** with custom status codes, delays, and properties
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

DummyBox provides several command endpoints for testing different scenarios. These endpoints require authentication if `auth-token` is configured. In those cases, provide the token via the `token` query parameter or `X-Auth-Token` HTTP header.

> 📖 **For detailed documentation with examples and use cases, see [COMMANDS.md](COMMANDS.md)**

### `/respond` - Configurable HTTP Response Simulation

Introduces configurable delays, status codes, and custom HTTP response headers for testing various scenarios including timeout handling, latency, and custom header scenarios.

**Parameters**: 
- `duration` (0-300s)
- `code` (HTTP status)
- `format` (json/text)
- `header_name`/`header_value` (custom headers for GET)
- `headers` (custom headers object for POST)

### `/log` - Log Generation

Generates structured log messages for testing log aggregation systems and monitoring alerts.

**Parameters**: 
- `level` (info/warning/error)
- `size` (short/medium/long)
- `message` (custom)
- `interval` (0-3600s)
- `duration` (0-86400s)
- `correlation` (true/false)


### `/cpu` - CPU Load Generation

Simulates CPU load with configurable intensity levels for testing CPU monitoring and autoscaling scenarios.

**Parameters**: 
- `intensity` (light/medium/heavy/extreme)
- `duration` (0-3600s, 0=forever)
- `format` (json/text)

### `/memory` - Memory Utilization Generator

Allocates memory to simulate memory pressure for testing OOM conditions and resource limits.

**Parameters**:
- `size` (1-8192 MB)
- `duration` (0-3600s, 0=forever)
- `format` (json/text)


**DummyBox** - Making container testing simple! 🚀
