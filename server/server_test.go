package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/crlsmrls/dummybox/config"
	"github.com/crlsmrls/dummybox/logger"
	"github.com/crlsmrls/dummybox/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// getLogEntries reads a buffer and returns a slice of JSON log entries.
func getLogEntries(t *testing.T, buf *bytes.Buffer) []map[string]interface{} {
	var entries []map[string]interface{}
	sc := bufio.NewScanner(buf)
	for sc.Scan() {
		var entry map[string]interface{}
		if err := json.Unmarshal(sc.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}
		entries = append(entries, entry)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("Error scanning log buffer: %v", err)
	}
	return entries
}

var reg *prometheus.Registry

func TestMain(m *testing.M) {
	// Initialize metrics once for all tests
	reg = metrics.InitMetrics()
	// Run tests
	os.Exit(m.Run())
}

func TestHealthAndReadyzEndpoints(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := New(cfg, nil, reg)

	testServer := httptest.NewServer(srv.router)
	defer testServer.Close()

	// Test /healthz
	res, err := http.Get(testServer.URL + "/healthz")
	if err != nil {
		t.Fatalf("Failed to send GET request to /healthz: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for /healthz, got %d", http.StatusOK, res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)
	if string(body) != "OK" {
		t.Errorf("Expected body \"OK\" for /healthz, got \"%s\"", string(body))
	}
}

func TestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger.InitLogger("debug", &buf)

	cfg := config.DefaultConfig()
	cfg.LogLevel = "debug" // Override for this test
	srv := New(cfg, &buf, reg)

	testServer := httptest.NewServer(srv.router)
	defer testServer.Close()

	_, err := http.Get(testServer.URL + "/healthz")
	if err != nil {
		t.Fatalf("Failed to send GET request: %v", err)
	}

	entries := getLogEntries(t, &buf)
	if len(entries) == 0 {
		t.Fatal("No log entries found")
	}

	logOutput := entries[0] // Get the first log entry

	// Check for basic log fields
	if _, ok := logOutput["time"]; !ok {
		t.Error("Log output missing time field")
	}
	if logOutput["level"] != "info" {
		t.Errorf("Expected log level 'info', got %v", logOutput["level"])
	}
	if logOutput["message"] != "request" {
		t.Errorf("Expected log message 'request', got %v", logOutput["message"])
	}
	// Check for request-specific fields
	if logOutput["method"] != "GET" {
		t.Errorf("Expected method 'GET', got %v", logOutput["method"])
	}
	if logOutput["url"] != "/healthz" {
		t.Errorf("Expected URL '/healthz', got %v", logOutput["url"])
	}
	if logOutput["status"] != float64(http.StatusOK) {
		t.Errorf("Expected status %d, got %v", http.StatusOK, logOutput["status"])
	}
}

func TestCorrelationIDMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger.InitLogger("debug", &buf)

	cfg := config.DefaultConfig()
	cfg.LogLevel = "debug" // Override for this test
	srv := New(cfg, &buf, reg)

	testServer := httptest.NewServer(srv.router)
	defer testServer.Close()

	// Test with no X-Correlation-ID header (should generate one)
	req, _ := http.NewRequest("GET", testServer.URL+"/healthz", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send GET request: %v", err)
	}
	defer res.Body.Close()

	correlationID := res.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		t.Error("Expected X-Correlation-ID header, got empty")
	}

	// Verify correlation ID in logs
	entries := getLogEntries(t, &buf)
	if len(entries) == 0 {
		t.Fatal("No log entries found")
	}
	logOutput := entries[0]

	if logOutput["correlation_id"] != correlationID {
		t.Errorf("Expected correlation_id in log to be %s, got %v", correlationID, logOutput["correlation_id"])
	}

	// Test with existing X-Correlation-ID header (should propagate it)
	buf.Reset() // Clear buffer for next test
	existingCorrelationID := "my-custom-correlation-id"
	req, _ = http.NewRequest("GET", testServer.URL+"/healthz", nil)
	req.Header.Set("X-Correlation-ID", existingCorrelationID)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send GET request: %v", err)
	}
	defer res.Body.Close()

	if res.Header.Get("X-Correlation-ID") != existingCorrelationID {
		t.Errorf("Expected X-Correlation-ID header to be %s, got %s", existingCorrelationID, res.Header.Get("X-Correlation-ID"))
	}

	// Verify correlation ID in logs
	entries = getLogEntries(t, &buf)
	if len(entries) == 0 {
		t.Fatal("No log entries found")
	}
	logOutput = entries[0]

	if logOutput["correlation_id"] != existingCorrelationID {
		t.Errorf("Expected correlation_id in log to be %s, got %v", existingCorrelationID, logOutput["correlation_id"])
	}
}

func TestGracefulShutdown(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := New(cfg, nil, reg)

	// Start the server in a goroutine
	done := make(chan struct{})
	go func() {
		srv.Start()
		close(done)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Send interrupt signal
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Failed to send signal: %v", err)
	}

	// Wait for the server to shut down
	select {
	case <-done:
		// Server shut down successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not shut down gracefully within 5 seconds")
	}
}

func TestRootEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := New(cfg, nil, reg)

	testServer := httptest.NewServer(srv.router)
	defer testServer.Close()

	res, err := http.Get(testServer.URL + "/")
	if err != nil {
		t.Fatalf("Failed to send GET request to /: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for /, got %d", http.StatusOK, res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)

	if !bytes.Contains(body, []byte("DummyBox")) {
		t.Errorf("Expected body to contain \"DummyBox\", but it didn't")
	}
	if !bytes.Contains(body, []byte("Available Endpoints")) {
		t.Errorf("Expected body to contain \"Available Endpoints\", but it didn't")
	}
}

func TestInfoEndpoint_JSON(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := New(cfg, nil, reg)

	testServer := httptest.NewServer(srv.router)
	defer testServer.Close()

	req, _ := http.NewRequest("GET", testServer.URL+"/info", nil)
	req.Header.Set("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send GET request to /info: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for /info, got %d", http.StatusOK, res.StatusCode)
	}

	var infoData map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&infoData); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if app, ok := infoData["application"].(map[string]interface{}); !ok || app["version"] == "" {
		t.Errorf("Expected application.version in JSON response")
	}
	if proc, ok := infoData["process"].(map[string]interface{}); !ok || proc["pid"] == nil {
		t.Errorf("Expected process.pid in JSON response")
	}
}

func TestInfoEndpoint_HTML(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := New(cfg, nil, reg)

	testServer := httptest.NewServer(srv.router)
	defer testServer.Close()

	req, _ := http.NewRequest("GET", testServer.URL+"/info", nil)
	req.Header.Set("Accept", "text/html")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send GET request to /info: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for /info, got %d", http.StatusOK, res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)

	if !bytes.Contains(body, []byte("DummyBox Information")) {
		t.Errorf("Expected body to contain \"DummyBox Information\", but it didn't")
	}
	if !bytes.Contains(body, []byte("Application")) {
		t.Errorf("Expected body to contain \"Application\", but it didn't")
	}
}

func TestRequestEndpoint_JSON(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := New(cfg, nil, reg)

	testServer := httptest.NewServer(srv.router)
	defer testServer.Close()

	// Test GET request with query parameters
	req, _ := http.NewRequest("GET", testServer.URL+"/request?param1=value1&param2=value2", nil)
	req.Header.Set("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send GET request to /request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for /request, got %d", http.StatusOK, res.StatusCode)
	}

	var reqInfo map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&reqInfo); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if reqInfo["method"] != "GET" {
		t.Errorf("Expected method GET, got %v", reqInfo["method"])
	}
	if qp, ok := reqInfo["query_parameters"].(map[string]interface{}); !ok || qp["param1"].([]interface{})[0] != "value1" {
		t.Errorf("Expected query_parameters to contain param1=value1")
	}

	// Test POST request with body and JWT
	bodyContent := `{"key":"value"}`
	jwtToken := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c`

	req, _ = http.NewRequest("POST", testServer.URL+"/request", strings.NewReader(bodyContent))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send POST request to /request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for /request, got %d", http.StatusOK, res.StatusCode)
	}

	if err := json.NewDecoder(res.Body).Decode(&reqInfo); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if reqInfo["method"] != "POST" {
		t.Errorf("Expected method POST, got %v", reqInfo["method"])
	}
	if reqInfo["body"] != bodyContent {
		t.Errorf("Expected body %s, got %v", bodyContent, reqInfo["body"])
	}
	if jwtInfo, ok := reqInfo["jwt"].(map[string]interface{}); !ok || jwtInfo["payload"].(map[string]interface{})["name"] != "John Doe" {
		t.Errorf("Expected JWT payload name to be John Doe")
	}
}

func TestRequestEndpoint_HTML(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := New(cfg, nil, reg)

	testServer := httptest.NewServer(srv.router)
	defer testServer.Close()

	// Test GET request with query parameters
	req, _ := http.NewRequest("GET", testServer.URL+"/request?param1=value1", nil)
	req.Header.Set("Accept", "text/html")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send GET request to /request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for /request, got %d", http.StatusOK, res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)

	if !bytes.Contains(body, []byte("DummyBox Request Information")) {
		t.Errorf("Expected body to contain \"DummyBox Request Information\", but it didn't")
	}
	if !bytes.Contains(body, []byte("Method: GET")) {
		t.Errorf("Expected body to contain \"Method: GET\", but it didn't")
	}
	if !bytes.Contains(body, []byte("param1")) || !bytes.Contains(body, []byte("value1")) {
		t.Errorf("Expected body to contain query parameters")
	}

	// Test POST request with body and JWT
	bodyContent := `{"key":"value"}`
	jwtToken := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c`

	req, _ = http.NewRequest("POST", testServer.URL+"/request", strings.NewReader(bodyContent))
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send POST request to /request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for /request, got %d", http.StatusOK, res.StatusCode)
	}

	body, _ = io.ReadAll(res.Body)

	if !bytes.Contains(body, []byte("Method: POST")) {
		t.Errorf("Expected body to contain \"Method: POST\", but it didn't")
	}
	if !bytes.Contains(body, []byte("key")) || !bytes.Contains(body, []byte("value")) {
		t.Errorf("Expected body to contain request body, but it didn't")
	}
	if !bytes.Contains(body, []byte("JWT Token")) || !bytes.Contains(body, []byte("John Doe")) {
		t.Errorf("Expected body to contain JWT info, but it didn't")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := New(cfg, nil, reg)

	testServer := httptest.NewServer(srv.router)
	defer testServer.Close()

	res, err := http.Get(testServer.URL + cfg.MetricsPath)
	if err != nil {
		t.Fatalf("Failed to send GET request to %s: %v", cfg.MetricsPath, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for %s, got %d", http.StatusOK, cfg.MetricsPath, res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)
	bodyStr := string(body)

	// Check for some expected metrics
	if !strings.Contains(bodyStr, "http_requests_total") {
		t.Errorf("Expected metrics output to contain http_requests_total")
	}
	if !strings.Contains(bodyStr, "go_goroutines") {
		t.Errorf("Expected metrics output to contain go_goroutines")
	}
}
