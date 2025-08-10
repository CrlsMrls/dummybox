package delay

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func init() {
	// Disable logging for tests
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestDelayHandler_GET_DefaultParameters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/delay", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	DelayHandler(w, req)
	elapsed := time.Since(start)

	// Should have no delay
	if elapsed > 100*time.Millisecond {
		t.Errorf("Expected minimal delay, got %v", elapsed)
	}

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type (default is JSON)
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Check response values
	if response["duration"] != float64(0) {
		t.Errorf("Expected duration 0, got %v", response["duration"])
	}
	if response["code"] != float64(200) {
		t.Errorf("Expected code 200, got %v", response["code"])
	}
}

func TestDelayHandler_GET_WithParameters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/delay?duration=1&code=201", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	DelayHandler(w, req)
	elapsed := time.Since(start)

	// Should have approximately 1 second delay
	if elapsed < 900*time.Millisecond || elapsed > 1100*time.Millisecond {
		t.Errorf("Expected ~1 second delay, got %v", elapsed)
	}

	// Check status code
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

func TestDelayHandler_GET_TextFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/delay?duration=0&code=202&format=text", nil)
	w := httptest.NewRecorder()

	DelayHandler(w, req)

	// Check status code
	if w.Code != 202 {
		t.Errorf("Expected status 202, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}

	// Check response body
	expectedBody := "Delayed for 0 seconds with status code 202\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}
}

func TestDelayHandler_POST_ValidJSON(t *testing.T) {
	params := DelayParams{
		Duration: 1,
		Code:     203,
	}
	jsonBody, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/delay", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	start := time.Now()
	DelayHandler(w, req)
	elapsed := time.Since(start)

	// Should have approximately 1 second delay
	if elapsed < 900*time.Millisecond || elapsed > 1100*time.Millisecond {
		t.Errorf("Expected ~1 second delay, got %v", elapsed)
	}

	// Check status code
	if w.Code != 203 {
		t.Errorf("Expected status 203, got %d", w.Code)
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
	if response["code"] != float64(203) {
		t.Errorf("Expected code 203, got %v", response["code"])
	}
}

func TestDelayHandler_POST_InvalidJSON(t *testing.T) {
	invalidJSON := `{"duration": "invalid", "code": 200}`

	req := httptest.NewRequest(http.MethodPost, "/delay", strings.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	DelayHandler(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	expectedBody := "Invalid JSON body\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}
}

func TestDelayHandler_POST_WithTextFormat(t *testing.T) {
	params := DelayParams{
		Duration: 0,
		Code:     204,
	}
	jsonBody, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/delay?format=text", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	DelayHandler(w, req)

	// Check status code
	if w.Code != 204 {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}

	// Check response body
	expectedBody := "Delayed for 0 seconds with status code 204\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}
}

func TestDelayHandler_ParameterValidation(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectedDuration int
		expectedCode     int
	}{
		{
			name:             "negative duration",
			url:              "/delay?duration=-1&code=200",
			expectedDuration: 0,
			expectedCode:     200,
		},
		{
			name:             "excessive duration",
			url:              "/delay?duration=500&code=200",
			expectedDuration: 0,
			expectedCode:     200,
		},
		{
			name:             "invalid status code low",
			url:              "/delay?duration=0&code=99",
			expectedDuration: 0,
			expectedCode:     200,
		},
		{
			name:             "invalid status code high",
			url:              "/delay?duration=0&code=600",
			expectedDuration: 0,
			expectedCode:     200,
		},
		{
			name:             "valid edge case duration",
			url:              "/delay?duration=2&code=500",
			expectedDuration: 2,
			expectedCode:     500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			start := time.Now()
			DelayHandler(w, req)
			elapsed := time.Since(start)

			// Check the actual delay with tolerance
			expectedDelay := time.Duration(tt.expectedDuration) * time.Second
			tolerance := 200 * time.Millisecond
			if elapsed < expectedDelay-tolerance || elapsed > expectedDelay+tolerance {
				t.Errorf("Expected delay ~%v, got %v", expectedDelay, elapsed)
			}

			// Check status code
			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, w.Code)
			}

			// Parse JSON response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to parse JSON response: %v", err)
			}

			// Check response values
			if response["duration"] != float64(tt.expectedDuration) {
				t.Errorf("Expected duration %d, got %v", tt.expectedDuration, response["duration"])
			}
			if response["code"] != float64(tt.expectedCode) {
				t.Errorf("Expected code %d, got %v", tt.expectedCode, response["code"])
			}
		})
	}
}

func TestDelayHandler_UnsupportedMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/delay", nil)
	w := httptest.NewRecorder()

	DelayHandler(w, req)

	// Should use default parameters for unsupported methods
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Should use default values
	if response["duration"] != float64(0) {
		t.Errorf("Expected duration 0, got %v", response["duration"])
	}
	if response["code"] != float64(200) {
		t.Errorf("Expected code 200, got %v", response["code"])
	}
}
