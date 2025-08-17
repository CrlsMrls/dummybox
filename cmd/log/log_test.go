package log

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func init() {
	// Disable logging for tests
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestLogHandler_GET_DefaultParameters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/log", nil)
	w := httptest.NewRecorder()

	LogHandler(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Check default response values
	if response["level"] != "info" {
		t.Errorf("Expected default level 'info', got %v", response["level"])
	}
	if response["size"] != "short" {
		t.Errorf("Expected default size 'short', got %v", response["size"])
	}
	if response["interval"] != float64(0) {
		t.Errorf("Expected default interval 0, got %v", response["interval"])
	}
	if response["duration"] != float64(0) {
		t.Errorf("Expected default duration 0, got %v", response["duration"])
	}
	if response["correlation"] != "true" {
		t.Errorf("Expected default correlation 'true', got %v", response["correlation"])
	}
	if response["status"] != "log generation started" {
		t.Errorf("Expected status 'log generation started', got %v", response["status"])
	}
}

func TestLogHandler_GET_WithParameters(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected LogParams
	}{
		{
			name: "info level, medium size",
			url:  "/log?level=info&size=medium&interval=0",
			expected: LogParams{
				Level:    "info",
				Size:     "medium",
				Interval: 0,
			},
		},
		{
			name: "warning level, long size",
			url:  "/log?level=warning&size=long&interval=0",
			expected: LogParams{
				Level:    "warning",
				Size:     "long",
				Interval: 0,
			},
		},
		{
			name: "error level, short size",
			url:  "/log?level=error&size=short&interval=0",
			expected: LogParams{
				Level:    "error",
				Size:     "short",
				Interval: 0,
			},
		},
		{
			name: "with custom message",
			url:  "/log?level=info&message=" + url.QueryEscape("Custom test message"),
			expected: LogParams{
				Level:   "info",
				Size:    "short", // default
				Message: "Custom test message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			LogHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to parse JSON response: %v", err)
			}

			if response["level"] != tt.expected.Level {
				t.Errorf("Expected level '%s', got %v", tt.expected.Level, response["level"])
			}
			if response["size"] != tt.expected.Size {
				t.Errorf("Expected size '%s', got %v", tt.expected.Size, response["size"])
			}
			if tt.expected.Message != "" {
				expectedMessage := tt.expected.Message + " (Fake message)"
				if response["message"] != expectedMessage {
					t.Errorf("Expected message '%s', got %v", expectedMessage, response["message"])
				}
			}
		})
	}
}

func TestLogHandler_POST_WithJSONBody(t *testing.T) {
	params := LogParams{
		Level:       "warning",
		Size:        "medium",
		Message:     "Test POST message",
		Interval:    0,
		Duration:    0,
		Correlation: "true",
	}

	body, _ := json.Marshal(params)
	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	LogHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response["level"] != params.Level {
		t.Errorf("Expected level '%s', got %v", params.Level, response["level"])
	}
	if response["size"] != params.Size {
		t.Errorf("Expected size '%s', got %v", params.Size, response["size"])
	}
	// Check that message includes "(Fake message)" suffix
	expectedMessage := params.Message + " (Fake message)"
	if response["message"] != expectedMessage {
		t.Errorf("Expected message '%s', got %v", expectedMessage, response["message"])
	}
}

func TestLogHandler_POST_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/log", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	LogHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid JSON body") {
		t.Errorf("Expected 'Invalid JSON body' in response, got: %s", body)
	}
}

func TestLogHandler_InvalidParameters(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedMsg string
	}{
		{
			name:        "invalid level",
			url:         "/log?level=debug&size=short",
			expectedMsg: "info", // should default to info
		},
		{
			name:        "invalid size",
			url:         "/log?level=info&size=huge",
			expectedMsg: "short", // should default to short  
		},
		{
			name:        "invalid interval",
			url:         "/log?level=info&interval=-5",
			expectedMsg: "0", // should default to 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			LogHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to parse JSON response: %v", err)
			}

			// The handler should have corrected invalid parameters
			// We just verify it doesn't crash and returns 200
		})
	}
}

func TestIsValidLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected bool
	}{
		{"info", true},
		{"warning", true},
		{"error", true},
		{"random", true},
		{"debug", false},
		{"fatal", false},
		{"INFO", true}, // case insensitive
		{"Warning", true},
		{"RANDOM", true},
		{"", false},
	}

	for _, tt := range tests {
		result := isValidLevel(tt.level)
		if result != tt.expected {
			t.Errorf("isValidLevel(%q) = %v, expected %v", tt.level, result, tt.expected)
		}
	}
}

func TestIsValidSize(t *testing.T) {
	tests := []struct {
		size     string
		expected bool
	}{
		{"short", true},
		{"medium", true},
		{"long", true},
		{"random", true},
		{"tiny", false},
		{"huge", false},
		{"SHORT", true}, // case insensitive
		{"Medium", true},
		{"RANDOM", true},
		{"", false},
	}

	for _, tt := range tests {
		result := isValidSize(tt.size)
		if result != tt.expected {
			t.Errorf("isValidSize(%q) = %v, expected %v", tt.size, result, tt.expected)
		}
	}
}

func TestGenerateLogMessage(t *testing.T) {
	tests := []string{"short", "medium", "long", "random"}

	for _, size := range tests {
		message := generateLogMessage(size)
		if message == "" {
			t.Errorf("generateLogMessage(%q) returned empty string", size)
		}

		// All messages should end with "(Fake message)"
		if !strings.HasSuffix(message, "(Fake message)") {
			t.Errorf("Message for size %q should end with '(Fake message)', got: %s", size, message)
		}

		// Test that different sizes produce different length ranges (excluding random)
		switch size {
		case "short":
			if len(message) > 150 {
				t.Errorf("Short message too long: %d chars", len(message))
			}
		case "medium":
			if len(message) < 80 || len(message) > 500 {
				t.Errorf("Medium message unexpected length: %d chars", len(message))
			}
		case "long":
			if len(message) < 300 {
				t.Errorf("Long message too short: %d chars", len(message))
			}
		case "random":
			// Random can be any size, just verify it's not empty and has fake suffix
			if len(message) < 20 {
				t.Errorf("Random message too short: %d chars", len(message))
			}
		}
	}

	// Test invalid size defaults to short
	message := generateLogMessage("invalid")
	if message == "" {
		t.Error("generateLogMessage with invalid size returned empty string")
	}
	if !strings.HasSuffix(message, "(Fake message)") {
		t.Error("Invalid size message should still end with '(Fake message)'")
	}
}

func TestGenerateLogEntry(t *testing.T) {
	// Just test that generateLogEntry doesn't crash with different levels
	tests := []struct {
		level string
		msg   string
	}{
		{"info", "Test info message"},
		{"warning", "Test warning message"},
		{"error", "Test error message"},
		{"invalid", "Test invalid level message"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			// Just verify it doesn't panic
			ctx := context.Background()
			generateLogEntry(ctx, tt.level, tt.msg)
			// If we reach here, the function didn't panic
		})
	}
}

func TestGetActualLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected []string // possible outputs for random
	}{
		{"info", []string{"info"}},
		{"warning", []string{"warning"}},
		{"error", []string{"error"}},
		{"random", []string{"info", "warning", "error"}},
		{"INFO", []string{"INFO"}},
		{"invalid", []string{"invalid"}},
	}

	for _, tt := range tests {
		result := getActualLevel(tt.input)
		
		// Check if result is in expected values
		found := false
		for _, expected := range tt.expected {
			if result == expected {
				found = true
				break
			}
		}
		
		if !found {
			t.Errorf("getActualLevel(%q) = %q, expected one of %v", tt.input, result, tt.expected)
		}
	}
}

func TestGetActualMessage(t *testing.T) {
	tests := []struct {
		customMessage string
		size          string
		expectCustom  bool
	}{
		{"Custom message", "short", true},
		{"", "short", false},
		{"Already fake (Fake message)", "medium", true},
		{"Custom without suffix", "long", true},
	}

	for _, tt := range tests {
		result := getActualMessage(tt.customMessage, tt.size)
		
		if tt.expectCustom {
			if tt.customMessage == "" {
				t.Errorf("Expected custom message but got generated message")
				continue
			}
			
			// Should contain custom message and end with (Fake message)
			if !strings.HasSuffix(result, "(Fake message)") {
				t.Errorf("Custom message should end with '(Fake message)', got: %s", result)
			}
			
			if strings.HasSuffix(tt.customMessage, "(Fake message)") {
				// Already has suffix, should not duplicate
				if result != tt.customMessage {
					t.Errorf("Message with existing suffix should not be modified: got %s, expected %s", result, tt.customMessage)
				}
			} else {
				// Should add suffix
				expected := tt.customMessage + " (Fake message)"
				if result != expected {
					t.Errorf("Expected %s, got %s", expected, result)
				}
			}
		} else {
			// Should be generated message
			if !strings.HasSuffix(result, "(Fake message)") {
				t.Errorf("Generated message should end with '(Fake message)', got: %s", result)
			}
		}
	}
}

func TestLogHandler_NewParameters(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedDuration int
		expectedCorrelation string
	}{
		{
			name:        "with duration parameter",
			url:         "/log?duration=300&correlation=false",
			expectedDuration: 300,
			expectedCorrelation: "false",
		},
		{
			name:        "with random level and size",
			url:         "/log?level=random&size=random",
			expectedDuration: 0,
			expectedCorrelation: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			LogHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to parse JSON response: %v", err)
			}

			if response["duration"] != float64(tt.expectedDuration) {
				t.Errorf("Expected duration %d, got %v", tt.expectedDuration, response["duration"])
			}
			if response["correlation"] != tt.expectedCorrelation {
				t.Errorf("Expected correlation '%s', got %v", tt.expectedCorrelation, response["correlation"])
			}
		})
	}
}

func TestLogHandler_CorrelationID(t *testing.T) {
	correlationID := "test-correlation-123"
	
	req := httptest.NewRequest(http.MethodGet, "/log?level=info", nil)
	req.Header.Set("X-Correlation-ID", correlationID)
	w := httptest.NewRecorder()

	LogHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// The correlation ID should be handled by middleware in the actual server
	// Here we just verify the handler doesn't crash when the header is present
}
