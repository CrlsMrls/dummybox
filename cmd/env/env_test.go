package env

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestEnvHandler_GET_JSON(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	req := httptest.NewRequest(http.MethodGet, "/env?format=json", nil)
	w := httptest.NewRecorder()

	EnvHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["format"] != "json" {
		t.Errorf("Expected format 'json', got %v", response["format"])
	}

	envVars, ok := response["environment_variables"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected environment_variables to be a map")
	}

	if envVars["TEST_VAR"] != "test_value" {
		t.Errorf("Expected TEST_VAR to be 'test_value', got %v", envVars["TEST_VAR"])
	}
}

func TestEnvHandler_GET_Text(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	req := httptest.NewRequest(http.MethodGet, "/env?format=text", nil)
	w := httptest.NewRecorder()

	EnvHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if !strings.Contains(body, "TEST_VAR=test_value") {
		t.Errorf("Expected body to contain 'TEST_VAR=test_value', got: %s", body)
	}

	if !strings.Contains(body, "Environment Variables") {
		t.Errorf("Expected body to contain 'Environment Variables', got: %s", body)
	}
}

func TestEnvHandler_POST_JSON(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	params := EnvParams{
		Format: "json",
	}
	jsonBody, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/env", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	EnvHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["format"] != "json" {
		t.Errorf("Expected format 'json', got %v", response["format"])
	}
}

func TestEnvHandler_POST_Text(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	params := EnvParams{
		Format: "text",
	}
	jsonBody, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/env", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	EnvHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if !strings.Contains(body, "TEST_VAR=test_value") {
		t.Errorf("Expected body to contain 'TEST_VAR=test_value', got: %s", body)
	}
}

func TestEnvHandler_InvalidFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/env?format=invalid", nil)
	w := httptest.NewRecorder()

	EnvHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Should default to JSON for invalid format
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json for invalid format, got %s", w.Header().Get("Content-Type"))
	}
}

func TestEnvHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/env", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	EnvHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
