package env_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/crlsmrls/dummybox/cmd/env"
	"github.com/crlsmrls/dummybox/config"
	"github.com/crlsmrls/dummybox/server"
	"github.com/rs/zerolog"
)

func init() {
	// Disable logging for tests
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestEnvEndpoint_WithoutAuth(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_ENV_VAR", "test_value")
	defer os.Unsetenv("TEST_ENV_VAR")

	// Create config without auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "", // No auth required
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/env?format=json", nil)
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

	if response["format"] != "json" {
		t.Errorf("Expected format 'json', got %v", response["format"])
	}

	envVars, ok := response["environment_variables"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected environment_variables to be a map")
	}

	if envVars["TEST_ENV_VAR"] != "test_value" {
		t.Errorf("Expected TEST_ENV_VAR to be 'test_value', got %v", envVars["TEST_ENV_VAR"])
	}
}

func TestEnvEndpoint_WithAuth_NoToken(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "secret-token", // Auth required
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/env?format=json", nil)
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

func TestEnvEndpoint_WithAuth_ValidTokenInQuery(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_ENV_VAR", "test_value")
	defer os.Unsetenv("TEST_ENV_VAR")

	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "secret-token", // Auth required
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	// Create request with valid token in query parameter
	req := httptest.NewRequest(http.MethodGet, "/env?format=text&token=secret-token", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if !strings.Contains(body, "TEST_ENV_VAR=test_value") {
		t.Errorf("Expected body to contain 'TEST_ENV_VAR=test_value', got: %s", body)
	}
}

func TestEnvEndpoint_POST_WithAuth(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_ENV_VAR", "test_value")
	defer os.Unsetenv("TEST_ENV_VAR")

	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "secret-token",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	// Create JSON body
	envParams := env.EnvParams{
		Format: "json",
	}
	jsonBody, _ := json.Marshal(envParams)

	// Create request with valid token in header
	req := httptest.NewRequest(http.MethodPost, "/env", bytes.NewReader(jsonBody))
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

	if response["format"] != envParams.Format {
		t.Errorf("Expected format '%s', got %v", envParams.Format, response["format"])
	}

	envVars, ok := response["environment_variables"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected environment_variables to be a map")
	}

	if envVars["TEST_ENV_VAR"] != "test_value" {
		t.Errorf("Expected TEST_ENV_VAR to be 'test_value', got %v", envVars["TEST_ENV_VAR"])
	}
}

func TestEnvEndpoint_CorrelationID(t *testing.T) {
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	correlationID := "test-correlation-12345"

	req := httptest.NewRequest(http.MethodGet, "/env?format=json", nil)
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
