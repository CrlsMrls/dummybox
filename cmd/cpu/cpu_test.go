package cpu

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockCPULoadGenerator is a test implementation that doesn't actually consume CPU.
type MockCPULoadGenerator struct {
	WorkCalls       []int
	SleepCalls      []time.Duration
	WorkCallCount   int
	SleepCallCount  int
}

func (m *MockCPULoadGenerator) DoWork(workSize int) int {
	m.WorkCalls = append(m.WorkCalls, workSize)
	m.WorkCallCount++
	return workSize / 10 // fake result
}

func (m *MockCPULoadGenerator) Sleep(duration time.Duration) {
	m.SleepCalls = append(m.SleepCalls, duration)
	m.SleepCallCount++
	// Don't actually sleep in tests
}

func setupMockGenerator() *MockCPULoadGenerator {
	mock := &MockCPULoadGenerator{}
	SetCPULoadGenerator(mock)
	return mock
}

func teardownMockGenerator() {
	SetCPULoadGenerator(&ProductionCPULoadGenerator{})
}

func TestValidateIntensity(t *testing.T) {
	tests := []struct {
		input    string
		expected CPUIntensity
		valid    bool
	}{
		{"light", Light, true},
		{"medium", Medium, true},
		{"heavy", Heavy, true},
		{"extreme", Extreme, true},
		{"invalid", "", false},
		{"", "", false},
		{"LIGHT", "", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			intensity, valid := ValidateIntensity(tt.input)
			if valid != tt.valid {
				t.Errorf("ValidateIntensity(%q) valid = %v, want %v", tt.input, valid, tt.valid)
			}
			if valid && intensity != tt.expected {
				t.Errorf("ValidateIntensity(%q) intensity = %v, want %v", tt.input, intensity, tt.expected)
			}
		})
	}
}

func TestGetIntensityConfig(t *testing.T) {
	tests := []struct {
		intensity CPUIntensity
		expectExists bool
	}{
		{Light, true},
		{Medium, true},
		{Heavy, true},
		{Extreme, true},
		{CPUIntensity("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.intensity), func(t *testing.T) {
			config, exists := GetIntensityConfig(tt.intensity)
			if exists != tt.expectExists {
				t.Errorf("GetIntensityConfig(%v) exists = %v, want %v", tt.intensity, exists, tt.expectExists)
			}
			if exists {
				if config.WorkSize <= 0 {
					t.Error("Expected positive work size")
				}
				if config.WorkDuration <= 0 {
					t.Error("Expected positive work duration")
				}
				if config.Description == "" {
					t.Error("Expected non-empty description")
				}
			}
		})
	}
}

func TestCPUHandler_GET_DefaultParameters(t *testing.T) {
	mock := setupMockGenerator()
	defer teardownMockGenerator()

	req := httptest.NewRequest(http.MethodGet, "/cpu", nil)
	w := httptest.NewRecorder()

	CPUHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["intensity"] != "medium" {
		t.Errorf("expected default intensity 'medium', got %v", response["intensity"])
	}

	if int(response["duration"].(float64)) != 60 {
		t.Errorf("expected default duration 60, got %v", response["duration"])
	}

	// Verify job key is present
	if jobKey, ok := response["job_key"].(string); !ok || jobKey == "" {
		t.Error("expected job_key to be present and non-empty")
	}

	// Wait briefly for workers to start and verify mock was called
	time.Sleep(50 * time.Millisecond)
	cleanupAllJobs()

	if mock.WorkCallCount == 0 {
		t.Error("expected CPU work to be called")
	}
}

func TestCPUHandler_GET_WithParameters(t *testing.T) {
	_ = setupMockGenerator()
	defer teardownMockGenerator()

	req := httptest.NewRequest(http.MethodGet, "/cpu?intensity=heavy&duration=30", nil)
	w := httptest.NewRecorder()

	CPUHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["intensity"] != "heavy" {
		t.Errorf("expected intensity 'heavy', got %v", response["intensity"])
	}

	if int(response["duration"].(float64)) != 30 {
		t.Errorf("expected duration 30, got %v", response["duration"])
	}

	// Wait briefly and cleanup
	time.Sleep(50 * time.Millisecond)
	cleanupAllJobs()
}

func TestCPUHandler_POST_JSON(t *testing.T) {
	_ = setupMockGenerator()
	defer teardownMockGenerator()

	requestBody := `{"intensity": "light", "duration": 15}`
	req := httptest.NewRequest(http.MethodPost, "/cpu", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CPUHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["intensity"] != "light" {
		t.Errorf("expected intensity 'light', got %v", response["intensity"])
	}

	if int(response["duration"].(float64)) != 15 {
		t.Errorf("expected duration 15, got %v", response["duration"])
	}

	// Wait briefly and cleanup
	time.Sleep(50 * time.Millisecond)
	cleanupAllJobs()
}

func TestCPUHandler_TextFormat(t *testing.T) {
	_ = setupMockGenerator()
	defer teardownMockGenerator()

	req := httptest.NewRequest(http.MethodGet, "/cpu?intensity=extreme&duration=5&format=text", nil)
	w := httptest.NewRecorder()

	CPUHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("expected Content-Type text/plain, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Generating extreme CPU load") {
		t.Error("expected response to contain CPU load information")
	}
	if !strings.Contains(body, "Job key:") {
		t.Error("expected response to contain job key")
	}

	// Wait briefly and cleanup
	time.Sleep(50 * time.Millisecond)
	cleanupAllJobs()
}

func TestCPUHandler_InvalidIntensity(t *testing.T) {
	_ = setupMockGenerator()
	defer teardownMockGenerator()

	req := httptest.NewRequest(http.MethodGet, "/cpu?intensity=invalid&duration=10", nil)
	w := httptest.NewRecorder()

	CPUHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Should default to medium
	if response["intensity"] != "medium" {
		t.Errorf("expected intensity to default to 'medium', got %v", response["intensity"])
	}

	// Wait briefly and cleanup
	time.Sleep(50 * time.Millisecond)
	cleanupAllJobs()
}

func TestCPUHandler_InvalidJSON(t *testing.T) {
	setupMockGenerator()
	defer teardownMockGenerator()

	req := httptest.NewRequest(http.MethodPost, "/cpu", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CPUHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCPUWorker_IntensityLevels(t *testing.T) {
	_ = setupMockGenerator()
	defer teardownMockGenerator()

	tests := []struct {
		intensity      CPUIntensity
		expectedWork   int
		expectedSleep  time.Duration
	}{
		{Light, 5000, 400 * time.Millisecond},
		{Medium, 15000, 250 * time.Millisecond},
		{Heavy, 30000, 100 * time.Millisecond},
		{Extreme, 50000, 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.intensity), func(t *testing.T) {
			config, _ := GetIntensityConfig(tt.intensity)
			
			if config.WorkSize != tt.expectedWork {
				t.Errorf("intensity %s: expected work size %d, got %d", 
					tt.intensity, tt.expectedWork, config.WorkSize)
			}
			
			if config.SleepDuration != tt.expectedSleep {
				t.Errorf("intensity %s: expected sleep duration %v, got %v", 
					tt.intensity, tt.expectedSleep, config.SleepDuration)
			}
		})
	}
}

func TestGetCPUStats(t *testing.T) {
	// Clear any existing jobs first
	cleanupAllJobs()

	// Test initial stats
	stats := GetCPUStats()
	
	if stats["total_jobs"].(int) != 0 {
		t.Errorf("expected 0 active jobs initially, got %v", stats["total_jobs"])
	}

	if stats["cpu_count"].(int) <= 0 {
		t.Error("expected positive CPU count")
	}

	if stats["default_intensity"].(string) != "medium" {
		t.Errorf("expected default intensity 'medium', got %v", stats["default_intensity"])
	}

	intensityLevels := stats["intensity_levels"].([]string)
	if len(intensityLevels) != 4 {
		t.Errorf("expected 4 intensity levels, got %d", len(intensityLevels))
	}

	activeJobs := stats["active_jobs"].([]string)
	if len(activeJobs) != 0 {
		t.Errorf("expected 0 active jobs, got %d", len(activeJobs))
	}
}

func TestGetAvailableIntensities(t *testing.T) {
	intensities := GetAvailableIntensities()
	
	expectedLevels := []CPUIntensity{Light, Medium, Heavy, Extreme}
	for _, level := range expectedLevels {
		config, exists := intensities[level]
		if !exists {
			t.Errorf("expected intensity level %s to exist", level)
		}
		if config.Description == "" {
			t.Errorf("expected description for intensity level %s", level)
		}
	}
}

// Helper function to clean up all running jobs
func cleanupAllJobs() {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()

	for jobKey, cancel := range cpuJobs {
		cancel()
		delete(cpuJobs, jobKey)
	}
}
