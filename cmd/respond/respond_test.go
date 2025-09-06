package respond

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func init() {
	// Disable logging for tests
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestRespondHandler_Basic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/respond", nil)
	w := httptest.NewRecorder()

	RespondHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	if response["duration"] != "0" {
		t.Errorf("Expected duration '0', got %v", response["duration"])
	}
	if response["code"] != "200" {
		t.Errorf("Expected code '200', got %v", response["code"])
	}
}

func TestRespondHandler_WithDelay(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/respond?duration=1&code=201", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	RespondHandler(w, req)
	elapsed := time.Since(start)

	if elapsed < 900*time.Millisecond || elapsed > 1100*time.Millisecond {
		t.Errorf("Expected ~1 second delay, got %v", elapsed)
	}

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}
