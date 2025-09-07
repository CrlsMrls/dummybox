package kill

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func init() {
	// Enable test mode to prevent actual os.Exit calls
	TestMode = true
}

func TestKillHandler_GET_DefaultParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/kill", nil)
	w := httptest.NewRecorder()

	KillHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["delay"] != float64(0) {
		t.Errorf("expected delay 0, got %v", response["delay"])
	}
	if response["code"] != float64(0) {
		t.Errorf("expected code 0, got %v", response["code"])
	}
	if response["status"] != "termination scheduled" {
		t.Errorf("expected status 'termination scheduled', got %v", response["status"])
	}
}

func TestKillHandler_GET_WithParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/kill?delay=5&code=1", nil)
	w := httptest.NewRecorder()

	KillHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["delay"] != float64(5) {
		t.Errorf("expected delay 5, got %v", response["delay"])
	}
	if response["code"] != float64(1) {
		t.Errorf("expected code 1, got %v", response["code"])
	}
}

func TestKillHandler_POST_JSON(t *testing.T) {
	params := KillParams{
		Delay: 10,
		Code:  2,
	}
	jsonBody, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/kill", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	KillHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["delay"] != float64(10) {
		t.Errorf("expected delay 10, got %v", response["delay"])
	}
	if response["code"] != float64(2) {
		t.Errorf("expected code 2, got %v", response["code"])
	}
}

func TestKillHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/kill", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	KillHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestKillHandler_ParameterValidation(t *testing.T) {
	tests := []struct {
		name          string
		delay         string
		code          string
		expectedDelay float64
		expectedCode  float64
	}{
		{
			name:          "negative delay",
			delay:         "-1",
			code:          "0",
			expectedDelay: 0, // should default to 0
			expectedCode:  0,
		},
		{
			name:          "excessive delay",
			delay:         "4000",
			code:          "0",
			expectedDelay: 0, // should default to 0
			expectedCode:  0,
		},
		{
			name:          "negative code",
			delay:         "0",
			code:          "-1",
			expectedDelay: 0,
			expectedCode:  0, // should default to 0
		},
		{
			name:          "excessive code",
			delay:         "0",
			code:          "300",
			expectedDelay: 0,
			expectedCode:  0, // should default to 0
		},
		{
			name:          "valid boundary values",
			delay:         "3600",
			code:          "255",
			expectedDelay: 3600,
			expectedCode:  255,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/kill?delay="+tt.delay+"&code="+tt.code, nil)
			w := httptest.NewRecorder()

			KillHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if response["delay"] != tt.expectedDelay {
				t.Errorf("expected delay %v, got %v", tt.expectedDelay, response["delay"])
			}
			if response["code"] != tt.expectedCode {
				t.Errorf("expected code %v, got %v", tt.expectedCode, response["code"])
			}
		})
	}
}
