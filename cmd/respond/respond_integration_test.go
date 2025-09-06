package respond_test

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

func TestRespondEndpoint_Integration_WithoutAuth(t *testing.T) {
	// Create config without auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "", // No auth required
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/respond?duration=0&code=200", nil)
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

func TestRespondEndpoint_Integration_WithAuth_NoToken(t *testing.T) {
	// Create config with auth token
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "test-token",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/respond?duration=0&code=200", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	// Should return 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRespondEndpoint_Integration_GET_WithHeaders(t *testing.T) {
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/respond?duration=0&code=200&header_name=X-Custom-Agent&header_value=TestAgent&header_name=X-Request-ID&header_value=12345", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that custom headers are set in HTTP response
	if w.Header().Get("X-Custom-Agent") != "TestAgent" {
		t.Errorf("Expected X-Custom-Agent header 'TestAgent', got %v", w.Header().Get("X-Custom-Agent"))
	}
	if w.Header().Get("X-Request-ID") != "12345" {
		t.Errorf("Expected X-Request-ID header '12345', got %v", w.Header().Get("X-Request-ID"))
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Check that headers are included in response body
	headers, exists := response["headers"]
	if !exists {
		t.Errorf("Expected headers in response")
		return
	}

	headerMap, ok := headers.(map[string]interface{})
	if !ok {
		t.Errorf("Expected headers to be a map")
		return
	}

	if headerMap["X-Custom-Agent"] != "TestAgent" {
		t.Errorf("Expected X-Custom-Agent 'TestAgent', got %v", headerMap["X-Custom-Agent"])
	}
	if headerMap["X-Request-ID"] != "12345" {
		t.Errorf("Expected X-Request-ID '12345', got %v", headerMap["X-Request-ID"])
	}
}

func TestRespondEndpoint_Integration_POST_WithHeaders(t *testing.T) {
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	// Create request body with headers
	requestBody := map[string]interface{}{
		"duration": 0,
		"code":     201,
		"headers": map[string]string{
			"X-User-ID":    "123",
			"X-Session-ID": "abc123",
			"X-Version":    "1.0",
		},
	}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/respond", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Check that custom headers are set in HTTP response
	if w.Header().Get("X-User-ID") != "123" {
		t.Errorf("Expected X-User-ID header '123', got %v", w.Header().Get("X-User-ID"))
	}
	if w.Header().Get("X-Session-ID") != "abc123" {
		t.Errorf("Expected X-Session-ID header 'abc123', got %v", w.Header().Get("X-Session-ID"))
	}
	if w.Header().Get("X-Version") != "1.0" {
		t.Errorf("Expected X-Version header '1.0', got %v", w.Header().Get("X-Version"))
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Check that headers are included in response body
	headers, exists := response["headers"]
	if !exists {
		t.Errorf("Expected headers in response")
		return
	}

	headerMap, ok := headers.(map[string]interface{})
	if !ok {
		t.Errorf("Expected headers to be a map")
		return
	}

	if headerMap["X-User-ID"] != "123" {
		t.Errorf("Expected X-User-ID '123', got %v", headerMap["X-User-ID"])
	}
	if headerMap["X-Session-ID"] != "abc123" {
		t.Errorf("Expected X-Session-ID 'abc123', got %v", headerMap["X-Session-ID"])
	}
	if headerMap["X-Version"] != "1.0" {
		t.Errorf("Expected X-Version '1.0', got %v", headerMap["X-Version"])
	}
}

func TestRespondEndpoint_Integration_DelayFunctionality(t *testing.T) {
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	start := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/respond?duration=1&code=200", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)
	elapsed := time.Since(start)

	// Should have approximately 1 second delay
	if elapsed < 900*time.Millisecond || elapsed > 1100*time.Millisecond {
		t.Errorf("Expected ~1 second delay, got %v", elapsed)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRespondEndpoint_Integration_TextFormat_WithHeaders(t *testing.T) {
	cfg := &config.Config{
		Port:        8080,
		LogLevel:    "info",
		MetricsPath: "/metrics",
		AuthToken:   "",
	}

	srv := server.NewTestServerWithRecorder(cfg, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/respond?duration=0&code=200&format=text&header_name=X-Test-Name&header_value=test&header_name=X-Test-Value&header_value=42", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}

	// Check that custom headers are set
	if w.Header().Get("X-Test-Name") != "test" {
		t.Errorf("Expected X-Test-Name header 'test', got %v", w.Header().Get("X-Test-Name"))
	}
	if w.Header().Get("X-Test-Value") != "42" {
		t.Errorf("Expected X-Test-Value header '42', got %v", w.Header().Get("X-Test-Value"))
	}

	// Check that response contains headers in text format
	body := w.Body.String()
	if !contains(body, "Responded after 0 seconds with status code 200") {
		t.Errorf("Expected response message in body")
	}
	if !contains(body, "Custom Headers:") {
		t.Errorf("Expected headers section in body")
	}
	if !contains(body, "X-Test-Name: test") {
		t.Errorf("Expected X-Test-Name header in body")
	}
	if !contains(body, "X-Test-Value: 42") {
		t.Errorf("Expected X-Test-Value header in body")
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
