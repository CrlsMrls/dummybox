# DummyBox Command Endpoints Documentation

This document provides detailed information about DummyBox's command endpoints, which are designed for testing various application behaviors and scenarios.

## Overview

Command endpoints simulate different application behaviors to help test monitoring systems, network conditions, and application resilience.

## Authentication

If the server is configured with an `auth-token`, all command endpoints require authentication. The token can be provided in two ways:

1. **Query Parameter**: `?token=your-secret-token`
2. **HTTP Header**: `X-Auth-Token: your-secret-token`

If no authentication token is configured on the server, these endpoints are publicly accessible.

## Correlation ID Support

All endpoints support the `X-Correlation-ID` header for request tracing. If provided, the correlation ID will be included in all log entries related to the request and returned in the response headers.

---

## `/delay` - Response Delay Simulation

### Purpose
The delay endpoint introduces configurable delays in HTTP responses, making it ideal for testing:
- Timeout handling in client applications
- Load balancer behavior with slow backends
- Circuit breaker patterns
- Performance monitoring and alerting
- Network latency simulation

### HTTP Methods
- `GET`: Parameters passed as query parameters
- `POST`: Parameters passed in JSON request body

### Parameters

| Parameter | Type | Required | Default | Valid Range | Description |
|-----------|------|----------|---------|-------------|-------------|
| `duration` | integer | No | `0` | 0-300 | Delay duration in seconds before responding |
| `code` | integer | No | `200` | 100-599 | HTTP status code to return in the response |
| `format` | string | No | `json` | `json`, `text` | Response format type |


### Request Examples

#### GET Request with Query Parameters
```bash
# Basic delay of 2 seconds
curl "http://localhost:8080/delay?duration=2&token=your-token"

# Custom status code with delay
curl "http://localhost:8080/delay?duration=5&code=500&token=your-token"

# Text format response
curl "http://localhost:8080/delay?duration=1&code=201&format=text&token=your-token"
```

#### POST Request with JSON Body and correlation ID
```bash
curl -X POST "http://localhost:8080/delay" \
  -H "Content-Type: application/json" \
  -H "X-Correlation-ID: test-scenario-001"
  -H "X-Auth-Token: your-token" \
  -d '{
    "duration": 3,
    "code": 202,
    "format": "json"
  }'
```

### Response Examples

#### JSON Response (default)
```json
{
  "duration": 2,
  "code": 200,
  "message": "Delayed for 2 seconds with status code 200"
}
```

#### Text Response
```
Delayed for 2 seconds with status code 200
```

---

## `/log` - Log Message Generation

### Purpose
The log endpoint generates structured log messages for testing:
- Log aggregation systems (ELK, Splunk, etc.)
- Monitoring and alerting systems
- Log parsing and analysis tools
- Structured logging pipelines
- Application performance monitoring (APM)

### HTTP Methods
- `GET`: Parameters passed as query parameters
- `POST`: Parameters passed in JSON request body

### Parameters

| Parameter | Type | Required | Default | Valid Values | Description |
|-----------|------|----------|---------|--------------|-------------|
| `level` | string | No | `info` | `info`, `warning`, `error`, `random` | Log level for the generated message |
| `size` | string | No | `short` | `short`, `medium`, `long`, `random` | Size category of the generated message |
| `message` | string | No | (auto-generated) | Any string | Custom message to log (takes precedence over size) |
| `interval` | integer | No | `0` | 0-3600 | Seconds between log entries (0 = log once) |
| `duration` | integer | No | `0` | 0-86400 | Total duration in seconds to generate logs (0 = indefinitely) |
| `correlation` | string | No | `true` | `true`, `false` | Whether to include correlation ID in log entries |

### Parameter Validation
- `level`: Invalid values default to `info`
- `size`: Invalid values default to `short`
- `interval`: Values outside 0-3600 range are reset to 0
- `duration`: Values outside 0-86400 range are reset to 0 (max 24 hours)
- `message`: If provided, takes precedence over auto-generated messages
- `correlation`: Set to "false" to exclude correlation ID from log entries

### Log Output Behavior

#### Output Streams
- **info level**: Logs to `stdout`
- **warning level**: Logs to `stderr`
- **error level**: Logs to `stderr`

#### Message Sizes
- **short**: 8-20 words, simple operational messages
  - Example: "System operational", "Task completed", "Connection established"
- **medium**: 15-50 words, detailed status messages
  - Example: "Database connection pool initialized with 10 connections, ready to serve requests"
- **long**: 100+ words, comprehensive diagnostic messages
  - Example: "System performance analysis completed: CPU usage averaged 35% over the last hour..."

#### Interval Logging
- `interval=0`: Generates one log entry immediately
- `interval>0`: Generates log entries continuously at the specified interval in background
- Background logging runs in separate goroutines and continues until the server stops

### Request Examples

#### GET Request - Single Log Entry
```bash
# Generate one info log with short message
curl "http://localhost:8080/log?level=info&size=short&token=your-token"

# Generate error log with long message
curl "http://localhost:8080/log?level=error&size=long&token=your-token"

# Custom message (URL-encoded)
curl "http://localhost:8080/log?level=warning&message=Custom%20test%20message&token=your-token"
```

#### GET Request - Interval Logging
```bash
# Generate error logs every 30 seconds
curl "http://localhost:8080/log?level=error&size=medium&interval=30&token=your-token"

# Generate info logs every 5 seconds with custom message
curl "http://localhost:8080/log?level=info&message=Heartbeat&interval=5&token=your-token"
```

#### POST Request with JSON Body
```bash
curl -X POST "http://localhost:8080/log" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Token: your-token" \
  -d '{
    "level": "warning",
    "size": "medium",
    "message": "Custom POST test message",
    "interval": 0
  }'
```

#### With Correlation ID
```bash
curl "http://localhost:8080/log?level=info&size=short&token=your-token" \
  -H "X-Correlation-ID: log-test-scenario-001"
```

### Response Examples

#### Successful Response
```json
{
  "level": "warning",
  "size": "medium", 
  "message": "Database connection pool initialized with 10 connections, ready to serve requests",
  "interval": 0,
  "status": "log generation started"
}
```

#### Interval Logging Response
```json
{
  "level": "error",
  "size": "long",
  "message": "System performance analysis completed: CPU usage averaged 35%...",
  "interval": 30,
  "status": "log generation started"
}
```

### Log Entry Format

All generated log entries use structured JSON format with the following fields:

```json
{
  "level": "info",
  "time": "2025-09-19T15:24:39+02:00",
  "message": "System operational"
}
```

When correlation ID is provided:
```json
{
  "level": "warning",
  "correlation_id": "log-test-scenario-001",
  "time": "2025-09-19T15:24:39+02:00", 
  "message": "Database connection pool initialized with 10 connections, ready to serve requests"
}
```

---

## Error Responses

### Authentication Errors
```bash
# Missing token when required
HTTP/1.1 401 Unauthorized
Unauthorized: token required

# Invalid token
HTTP/1.1 401 Unauthorized  
Unauthorized: invalid token
```

### Validation Errors
```bash
# Invalid JSON in POST request
HTTP/1.1 400 Bad Request
Invalid JSON body
```

---

## Testing and Monitoring Use Cases

### Load Testing
```bash
# Test timeout handling with various delays
for delay in 1 2 5 10; do
  curl "http://localhost:8080/delay?duration=$delay&token=your-token"
done
```

### Log Monitoring Testing
```bash
# Generate continuous error logs for alert testing
curl "http://localhost:8080/log?level=error&interval=60&token=your-token"

# Generate burst of warning logs
for i in {1..10}; do
  curl "http://localhost:8080/log?level=warning&message=Burst%20test%20$i&token=your-token"
done
```

### Performance Testing
```bash
# Test various response codes for monitoring
for code in 200 201 400 500 503; do
  curl "http://localhost:8080/delay?code=$code&token=your-token"
done
```
