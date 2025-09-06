# DummyBox Command Endpoints Documentation

This document provides detailed information about DummyBox's command endpoints, which are designed for testing various application behaviors and scenarios.

Command endpoints simulate different application behaviors to help test monitoring systems, network conditions, and application resilience.


All endpoints support the following features:

- `GET`: Parameters passed as query parameters
- `POST`: Parameters passed in JSON request body

## Authentication

The authentication mechanism is a simple token-based system. The token can be set via the `DUMMYBOX_AUTH_TOKEN` environment variable or `--auth-token` command-line flag.

When `auth-token` is configured, protected endpoints require authentication via token. You can provide the token in two ways:

1. **Query parameter**: `?token=your-secret-token`
2. **HTTP header**: `X-Auth-Token: your-secret-token`

**Note:** If no authentication token is configured on the server, these endpoints are publicly accessible.

## Correlation ID Support

All endpoints support the `X-Correlation-ID` HTTP header for request tracing. If provided, the correlation ID will be included in all log entries related to the request and returned in the response headers.

---

## `/respond` - Configurable HTTP Response Simulation

### Purpose
The respond endpoint introduces configurable delays, status codes, and custom HTTP response headers, making it ideal for testing:
- Timeout handling in client applications
- Load balancer behavior with slow backends
- Circuit breaker patterns
- Performance monitoring and alerting
- Network latency simulation
- Custom HTTP header scenarios
- API mocking with dynamic response headers


### Parameters

| Parameter | Type | Required | Default | Valid Range | Description |
|-----------|------|----------|---------|-------------|-------------|
| `duration` | integer | No | `0` | 0-300 | Delay duration in seconds before responding |
| `code` | integer | No | `200` | 100-599 | HTTP status code to return in the response |
| `format` | string | No | `json` | `json`, `text` | Response format type |
| `headers` | object | No | `{}` | - | Custom HTTP response headers (POST only) |
| `header_name` | string | No | - | - | Custom header name (GET only, paired with header_value) |
| `header_value` | string | No | - | - | Custom header value (GET only, paired with header_name) |


### Request Examples

#### GET Request with Query Parameters
```bash
# Basic response with 2 second delay
curl "http://localhost:8080/respond?duration=2"

# Custom status code with delay
curl "http://localhost:8080/respond?duration=5&code=500"

# Response with custom headers
curl "http://localhost:8080/respond?duration=0&code=200&header_name=X-Custom-Agent&header_value=MyApp&header_name=X-Request-ID&header_value=12345"
```

#### POST Request with JSON Body and correlation ID
```bash
curl -X POST "http://localhost:8080/respond" \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: test-scenario-001" \
  -H "X-Auth-Token: your-token" \
  -d '{
    "duration": 3,
    "code": 202,
    "headers": {
      "X-Custom-Agent": "MyApp",
      "X-Request-ID": "12345",
      "Authorization": "Bearer token123"
    }
  }'
```

### Response Examples

#### JSON Response (default)
```json
{
  "duration": "2",
  "code": "200",
  "message": "Responded after 2 seconds with status code 200"
}
```

#### JSON Response with Headers
```json
{
  "duration": "0",
  "code": "200",
  "message": "Responded after 0 seconds with status code 200",
  "headers": {
    "X-Custom-Agent": "MyApp",
    "X-Request-ID": "12345",
    "Authorization": "Bearer token123"
  }
}
```

#### Text Response
```
Responded after 2 seconds with status code 200
```

#### Text Response with Headers
```
Responded after 0 seconds with status code 200
Custom Headers:
  X-Custom-Agent: MyApp
  X-Request-ID: 12345
  Authorization: Bearer token123
```

**Note:** Custom headers are also set in the actual HTTP response headers, not just included in the response body for visibility.

---

## `/log` - Log Message Generation

### Purpose
The log endpoint generates structured log messages for testing:
- Log aggregation systems (ELK, Splunk, etc.)
- Monitoring and alerting systems
- Log parsing and analysis tools
- Structured logging pipelines
- Application performance monitoring (APM)


### Parameters

| Parameter | Type | Required | Default | Valid Values | Description |
|-----------|------|----------|---------|--------------|-------------|
| `level` | string | No | `info` | `info`, `warning`, `error`, `random` | Log level for the message |
| `size` | string | No | `short` | `short`, `medium`, `long`, `random` | Size category of the randomly generated message |
| `message` | string | No | (self-generated) | Any string | Custom message to log (takes precedence over size) |
| `interval` | integer | No | `0` | 0-3600 | Seconds between log entries (0 = log once) |
| `duration` | integer | No | `0` | 0-86400 | Total duration in seconds to generate logs (0 = indefinitely) |
| `correlation` | string | No | `true` | `true`, `false` | Whether to include correlation ID in log entries |


Interval Logging

- `interval=0`: Generates one log entry immediately
- `interval>0`: Generates log entries continuously at the specified interval in background
- Background logging runs in separate goroutines and continues until the server stops

### Request Examples

#### GET Request - Single Log Entry
```bash
# Generate one info log with short message
curl "http://localhost:8080/log?level=info&size=short"

# Generate error log with long message
curl "http://localhost:8080/log?level=error&size=long"

# Custom message (URL-encoded)
curl "http://localhost:8080/log?level=warning&message=Custom%20test%20message"
```

#### GET Request - Interval Logging
```bash
# Generate error logs every 30 seconds
curl "http://localhost:8080/log?level=error&size=medium&interval=30"

# Generate info logs every 5 seconds with custom message
curl "http://localhost:8080/log?level=info&message=Heartbeat&interval=5"
```

#### POST Request with JSON Body and Correlation ID
```bash
curl -X POST "http://localhost:8080/log" \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: log-test-scenario-001"
  -H "X-Auth-Token: your-token" \
  -d '{
    "level": "warning",
    "size": "medium",
    "message": "Custom POST test message",
    "interval": 0
  }'
```

### Response Examples

Successful Response

```json
{
  "level": "error",
  "size": "long",
  "message": "System performance analysis completed: CPU usage averaged 35%...",
  "interval": "30",
  "status": "log generation started"
}
```


---

## `/cpu` - CPU Load Generation

### Purpose
The CPU endpoint generates configurable CPU load with intensity-based control, making it ideal for testing:
- CPU monitoring and alerting systems
- Horizontal Pod Autoscaler (HPA) scaling based on CPU utilization
- Resource limit enforcement and throttling
- CPU-based container orchestration decisions
- Load balancer behavior with high CPU backends


### Parameters

| Parameter | Type | Required | Default | Valid Values | Description |
|-----------|------|----------|---------|--------------|-------------|
| `intensity` | string | No | `medium` | `light`, `medium`, `heavy`, `extreme` | CPU load intensity level |
| `duration` | integer | No | `60` | 0-3600 | Duration in seconds to generate CPU load (0 = forever) |
| `format` | string | No | `json` | `json`, `text` | Response format type |


Each intensity level uses a different work pattern to generate varying CPU loads:

| Intensity | Work Size | Work Duration | Sleep Duration | Description |
|-----------|-----------|---------------|----------------|-------------|
| `light` | 5000 | 100ms | 400ms | Light CPU stress - minimal system impact |
| `medium` | 15000 | 250ms | 250ms | Medium CPU stress - moderate system load |
| `heavy` | 30000 | 400ms | 100ms | Heavy CPU stress - high system load |
| `extreme` | 50000 | 500ms | 0ms | Extreme CPU stress - maximum system load |

### CPU Load Behavior

- Spawns one worker goroutine per CPU core (`runtime.NumCPU()`)
- Each worker performs CPU-intensive prime number calculations
- Workers run independently with configurable work/sleep cycles
- Automatic cleanup when duration expires or context is cancelled

### Request Examples

#### GET Request - Basic CPU Load
```bash
# Generate medium CPU load for 60 seconds (defaults)
curl "http://localhost:8080/cpu"

# Generate heavy CPU load for 2 minutes
curl "http://localhost:8080/cpu?intensity=heavy&duration=120"

# Generate light CPU load indefinitely
curl "http://localhost:8080/cpu?intensity=light&duration=0"
```

#### POST Request with JSON Body
```bash
# Generate CPU load using JSON payload
curl -X POST "http://localhost:8080/cpu" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: your-token" \
  -d '{
    "intensity": "heavy",
    "duration": 180
  }'
```

#### Advanced Usage with Correlation ID
```bash
curl -X POST "http://localhost:8080/cpu" \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: cpu-stress-test-001" \
  -H "X-Auth-Token: your-token" \
  -d '{
    "intensity": "extreme",
    "duration": 300
  }'
```

### Response Examples

#### JSON Response (Default)
```json
{
  "intensity": "heavy",
  "duration": "120",
  "job_key": "cpu-job-1-20240919-162548",
  "workers": "8",
  "description": "Heavy CPU stress - high system load",
  "config": {
    "work_size": "30000",
    "work_duration": "400ms",
    "sleep_duration": "100ms",
    "description": "Heavy CPU stress - high system load"
  },
  "message": "Generating heavy CPU load for 120 seconds"
}
```

#### Text Response (`format=text`)
```
Generating heavy CPU load for 120 seconds
Job key: cpu-job-1-20240919-162548
Workers: 8
Description: Heavy CPU stress - high system load
```

---

## `/memory` - Memory Utilization Generator

The `/memory` endpoint allows you to simulate memory utilization by allocating specified amounts of memory for a given duration. This is useful for:

- Testing application behavior under memory pressure
- Triggering memory-based alerts and monitoring
- Simulating out-of-memory conditions

### Parameters

| Parameter | Type | Required | Default | Valid Values | Description |
|-----------|------|----------|---------|--------------|-------------|
| `size` | integer | No | `100` | 1-8192 | Memory to allocate in MB (max 8GB) |
| `duration` | integer | No | `60` | 0-3600 | Duration to hold memory in seconds (0 = forever) |


### Memory Allocation Behavior

#### Allocation Strategy
- Memory is allocated in 10MB chunks to avoid large contiguous allocation issues
- Each allocation gets a unique key for tracking: `YYYYMMDD-HHMMSS-{size}`
- Memory is filled with data to prevent compiler optimizations
- Multiple concurrent allocations are supported

#### Memory Statistics
- Real-time heap size monitoring via `runtime.MemStats`
- Active allocation tracking with unique keys
- Automatic garbage collection after deallocation

### Request Examples

#### GET Request - Allocate Memory
```bash
# Allocate 50MB for 30 seconds
curl "http://localhost:8080/memory?size=50&duration=30"

# Allocate 200MB indefinitely
curl "http://localhost:8080/memory?size=200&duration=0"
```

#### POST Request - JSON Body
```bash
# Allocate memory using JSON payload
curl -X POST "http://localhost:8080/memory" \
  -H "Content-Type: application/json" \
  -d '{
    "size": 150,
    "duration": 45
  }'
```

### Response Examples

#### JSON Response (Default)
```json
{
  "size_mb": "100",
  "duration": "60",
  "allocation_key": "20250919-162329-100",
  "current_heap_mb": "105.47",
  "message": "Allocated 100MB of memory for 60 seconds"
}
```

#### Text Response (`format=text`)
```
Allocated 100MB of memory for 60 seconds
Current heap size: 105.47MB
Allocation key: 20250919-162329-100
```

