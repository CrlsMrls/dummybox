package delay_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/crlsmrls/dummybox/config"
	"github.com/crlsmrls/dummybox/server"
	"github.com/rs/zerolog"
)

func init() {
	// Disable logging for tests
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestDelayEndpoint_WithoutAuth(t *testing.T) {
	// Create config without auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "", // No auth required
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/delay?duration=0&code=200", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that response contains correlation ID
	correlationID := w.Header().Get("X-Correlation-ID")
	if correlationID == "" {
		t.Error("Expected X-Correlation-ID header to be set")
	}
}

func TestDelayEndpoint_WithAuth_NoToken(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info", 
		MetricsPath: "/metrics",
		AuthToken:   "secret-token", // Auth required
	}

	// Using test server utilities
	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/delay?duration=0&code=200", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	expectedBody := "Unauthorized: token required\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}
}

func TestDelayEndpoint_WithAuth_InvalidToken(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "secret-token", // Auth required
	}

	// Using test server utilities
	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	// Create request with invalid token
	req := httptest.NewRequest(http.MethodGet, "/delay?duration=0&code=200&token=wrong-token", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	expectedBody := "Unauthorized: invalid token\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}
}

func TestDelayEndpoint_WithAuth_ValidTokenInQuery(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "secret-token", // Auth required
	}

	// Using test server utilities
	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	// Create request with valid token in query parameter
	req := httptest.NewRequest(http.MethodGet, "/delay?duration=0&code=200&token=secret-token", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that response contains correlation ID
	correlationID := w.Header().Get("X-Correlation-ID")
	if correlationID == "" {
		t.Error("Expected X-Correlation-ID header to be set")
	}
}

func TestDelayEndpoint_POST_WithAuth(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "secret-token",
	}

	// Using test server utilities
	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	// Create JSON body
	delayParams := map[string]interface{}{
		"duration": 1,
		"code":     201,
	}
	jsonBody, _ := json.Marshal(delayParams)

	// Create request with valid token in header
	req := httptest.NewRequest(http.MethodPost, "/delay", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", "secret-token")
	w := httptest.NewRecorder()

	start := time.Now()
	srv.ServeHTTP(w, req)
	elapsed := time.Since(start)

	// Should have approximately 1 second delay
	if elapsed < 900*time.Millisecond || elapsed > 1100*time.Millisecond {
		t.Errorf("Expected ~1 second delay, got %v", elapsed)
	}

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Check response values
	if response["duration"] != float64(1) {
		t.Errorf("Expected duration 1, got %v", response["duration"])
	}
	if response["code"] != float64(201) {
		t.Errorf("Expected code 201, got %v", response["code"])
	}
}

func TestDelayEndpoint_CorrelationID_Provided(t *testing.T) {
	// Create config without auth token for simplicity
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "",
	}

	// Using test server utilities
	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	// Create request with correlation ID
	req := httptest.NewRequest(http.MethodGet, "/delay?duration=0&code=200", nil)
	testCorrelationID := "test-correlation-123"
	req.Header.Set("X-Correlation-ID", testCorrelationID)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that the same correlation ID is returned
	correlationID := w.Header().Get("X-Correlation-ID")
	if correlationID != testCorrelationID {
		t.Errorf("Expected correlation ID %q, got %q", testCorrelationID, correlationID)
	}
}


func TestDelayEndpoint_XAuthTokenHeader_Valid(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "my-secret-token",
	}

	// Using test server utilities
	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	// Create request with valid X-Auth-Token header
	req := httptest.NewRequest(http.MethodGet, "/delay?duration=0&code=200", nil)
	req.Header.Set("X-Auth-Token", "my-secret-token")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that response contains correlation ID
	correlationID := w.Header().Get("X-Correlation-ID")
	if correlationID == "" {
		t.Error("Expected X-Correlation-ID header to be set")
	}
}

func TestDelayEndpoint_TokenPrecedence(t *testing.T) {
	// Test that query parameter takes precedence over headers
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "correct-token",
	}

	// Using test server utilities
	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	// Create request with correct token in query and wrong token in header
	req := httptest.NewRequest(http.MethodGet, "/delay?duration=0&code=200&token=correct-token", nil)
	req.Header.Set("X-Auth-Token", "wrong-token")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
