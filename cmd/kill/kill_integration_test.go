package kill_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/crlsmrls/dummybox/cmd/kill"
	"github.com/crlsmrls/dummybox/config"
	"github.com/crlsmrls/dummybox/server"
	"github.com/rs/zerolog"
)

func init() {
	// Disable logging for tests
	zerolog.SetGlobalLevel(zerolog.Disabled)
	// Enable test mode to prevent actual os.Exit calls
	kill.TestMode = true
}

func TestKillEndpoint_Integration_WithoutAuth(t *testing.T) {
	// Create config without auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "", // No auth required
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/kill?delay=0&code=0", nil)
	req.Header.Set("X-Correlation-ID", "test-correlation-id")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that response contains correlation ID
	correlationID := w.Header().Get("X-Correlation-ID")
	if correlationID != "test-correlation-id" {
		t.Errorf("Expected X-Correlation-ID header to be 'test-correlation-id', got '%s'", correlationID)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "termination scheduled" {
		t.Errorf("Expected status 'termination scheduled', got %v", response["status"])
	}
	if response["delay"] != float64(0) {
		t.Errorf("Expected delay 0, got %v", response["delay"])
	}
	if response["code"] != float64(0) {
		t.Errorf("Expected code 0, got %v", response["code"])
	}
}

func TestKillEndpoint_Integration_WithAuth_NoToken(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "test-token",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/kill", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	// Should return 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestKillEndpoint_Integration_WithAuth_ValidToken(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "test-token",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/kill?delay=5&code=1", nil)
	req.Header.Set("X-Auth-Token", "test-token")
	req.Header.Set("X-Correlation-ID", "test-correlation-id")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["delay"] != float64(5) {
		t.Errorf("Expected delay 5, got %v", response["delay"])
	}
	if response["code"] != float64(1) {
		t.Errorf("Expected code 1, got %v", response["code"])
	}
}

func TestKillEndpoint_Integration_WithAuth_InvalidToken(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "test-token",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/kill", nil)
	req.Header.Set("X-Auth-Token", "wrong-token")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	// Should return 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestKillEndpoint_Integration_POST_JSON(t *testing.T) {
	// Create config without auth for simplicity
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	jsonBody := []byte(`{"delay": 10, "code": 42}`)
	req := httptest.NewRequest(http.MethodPost, "/kill", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-ID", "test-correlation-id")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["delay"] != float64(10) {
		t.Errorf("Expected delay 10, got %v", response["delay"])
	}
	if response["code"] != float64(42) {
		t.Errorf("Expected code 42, got %v", response["code"])
	}
	if response["status"] != "termination scheduled" {
		t.Errorf("Expected status 'termination scheduled', got %v", response["status"])
	}
}

func TestKillEndpoint_Integration_POST_InvalidJSON(t *testing.T) {
	// Create config without auth for simplicity
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/kill", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestKillEndpoint_Integration_ParameterValidation(t *testing.T) {
	// Create config without auth for simplicity
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	tests := []struct {
		name          string
		queryParams   string
		expectedDelay float64
		expectedCode  float64
	}{
		{
			name:          "negative delay should default to 0",
			queryParams:   "delay=-1&code=1",
			expectedDelay: 0,
			expectedCode:  1,
		},
		{
			name:          "excessive delay should default to 0",
			queryParams:   "delay=4000&code=1",
			expectedDelay: 0,
			expectedCode:  1,
		},
		{
			name:          "negative code should default to 0",
			queryParams:   "delay=1&code=-1",
			expectedDelay: 1,
			expectedCode:  0,
		},
		{
			name:          "excessive code should default to 0",
			queryParams:   "delay=1&code=300",
			expectedDelay: 1,
			expectedCode:  0,
		},
		{
			name:          "valid boundary values",
			queryParams:   "delay=3600&code=255",
			expectedDelay: 3600,
			expectedCode:  255,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/kill?"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			srv.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if response["delay"] != tt.expectedDelay {
				t.Errorf("Expected delay %v, got %v", tt.expectedDelay, response["delay"])
			}
			if response["code"] != tt.expectedCode {
				t.Errorf("Expected code %v, got %v", tt.expectedCode, response["code"])
			}
		})
	}
}
