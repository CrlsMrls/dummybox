package log_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/crlsmrls/dummybox/cmd/log"
	"github.com/crlsmrls/dummybox/config"
	"github.com/crlsmrls/dummybox/server"
	"github.com/rs/zerolog"
)

func init() {
	// Disable logging for tests
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestLogEndpoint_WithoutAuth(t *testing.T) {
	// Create config without auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "", // No auth required
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/log?level=info&size=short", nil)
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

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response["level"] != "info" {
		t.Errorf("Expected level 'info', got %v", response["level"])
	}
	if response["size"] != "short" {
		t.Errorf("Expected size 'short', got %v", response["size"])
	}
}

func TestLogEndpoint_WithAuth_NoToken(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "secret-token", // Auth required
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/log?level=info&size=short", nil)
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

func TestLogEndpoint_WithAuth_ValidTokenInQuery(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "secret-token", // Auth required
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	// Create request with valid token in query parameter
	req := httptest.NewRequest(http.MethodGet, "/log?level=warning&size=medium&token=secret-token", nil)
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

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response["level"] != "warning" {
		t.Errorf("Expected level 'warning', got %v", response["level"])
	}
	if response["size"] != "medium" {
		t.Errorf("Expected size 'medium', got %v", response["size"])
	}
}

func TestLogEndpoint_POST_WithAuth(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "secret-token",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	// Create JSON body
	logParams := log.LogParams{
		Level:       "error",
		Size:        "long",
		Message:     "POST integration test message",
		Interval:    0,
		Duration:    0,
		Correlation: "true",
	}
	jsonBody, _ := json.Marshal(logParams)

	// Create request with valid token in header
	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", "secret-token")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response["level"] != logParams.Level {
		t.Errorf("Expected level '%s', got %v", logParams.Level, response["level"])
	}
	if response["size"] != logParams.Size {
		t.Errorf("Expected size '%s', got %v", logParams.Size, response["size"])
	}
	// Check that message includes "(Fake message)" suffix
	expectedMessage := logParams.Message + " (Fake message)"
	if response["message"] != expectedMessage {
		t.Errorf("Expected message '%s', got %v", expectedMessage, response["message"])
	}
}

func TestLogEndpoint_CorrelationID(t *testing.T) {
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	correlationID := "test-correlation-12345"

	req := httptest.NewRequest(http.MethodGet, "/log?level=info", nil)
	req.Header.Set("X-Correlation-ID", correlationID)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that the correlation ID is returned in the response header
	responseCorrelationID := w.Header().Get("X-Correlation-ID")
	if responseCorrelationID != correlationID {
		t.Errorf("Expected correlation ID '%s', got '%s'", correlationID, responseCorrelationID)
	}
}
