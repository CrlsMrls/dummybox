package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMemoryEndpoint_BasicHTTP(t *testing.T) {
	// Simple integration test without complex middleware
	req := httptest.NewRequest(http.MethodGet, "/memory?size=20&duration=5", nil)
	w := httptest.NewRecorder()
	
	MemoryHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if int(response["size_mb"].(float64)) != 20 {
		t.Errorf("expected size_mb 20, got %v", response["size_mb"])
	}
}

func TestMemoryEndpoint_POST_JSON(t *testing.T) {
	params := MemoryParams{
		Size:     40,
		Duration: 15,
	}
	jsonBody, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/memory", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	MemoryHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if int(response["size_mb"].(float64)) != 40 {
		t.Errorf("expected size_mb 40, got %v", response["size_mb"])
	}
	if int(response["duration"].(float64)) != 15 {
		t.Errorf("expected duration 15, got %v", response["duration"])
	}
}

func TestMemoryEndpoint_DurationBehavior(t *testing.T) {
	// Test that memory allocation with short duration actually gets cleaned up
	req := httptest.NewRequest(http.MethodGet, "/memory?size=10&duration=1", nil)
	w := httptest.NewRecorder()
	
	MemoryHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	allocKey := response["allocation_key"].(string)

	// Immediately check that memory is allocated
	stats := GetMemoryStats()
	activeAllocs := stats["active_allocations"].(map[string]int)
	if _, exists := activeAllocs[allocKey]; !exists {
		t.Error("expected memory to be allocated immediately after request")
	}

	// Wait for duration + buffer
	time.Sleep(2 * time.Second)

	// Check that memory has been deallocated
	statsAfter := GetMemoryStats()
	activeAllocsAfter := statsAfter["active_allocations"].(map[string]int)
	if _, exists := activeAllocsAfter[allocKey]; exists {
		t.Error("expected memory to be deallocated after duration expires")
	}
}

func TestMemoryEndpoint_ZeroDuration(t *testing.T) {
	// Test that zero duration keeps memory allocated indefinitely
	req := httptest.NewRequest(http.MethodGet, "/memory?size=15&duration=0", nil)
	w := httptest.NewRecorder()
	
	MemoryHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	allocKey := response["allocation_key"].(string)

	// Wait a bit
	time.Sleep(500 * time.Millisecond)

	// Check that memory is still allocated
	stats := GetMemoryStats()
	activeAllocs := stats["active_allocations"].(map[string]int)
	if allocSize, exists := activeAllocs[allocKey]; !exists || allocSize != 15 {
		t.Errorf("expected 15MB allocation to persist with zero duration, got %v", activeAllocs)
	}

	// Clean up manually for test
	deallocateMemory(allocKey)
}

func TestMemoryEndpoint_TextFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/memory?size=55&duration=25&format=text", nil)
	w := httptest.NewRecorder()
	
	MemoryHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("expected Content-Type text/plain, got %s", w.Header().Get("Content-Type"))
	}

	if !strings.Contains(w.Body.String(), "Allocated 55MB") {
		t.Error("expected size information in text response")
	}
	if !strings.Contains(w.Body.String(), "25 seconds") {
		t.Error("expected duration information in text response")
	}
	if !strings.Contains(w.Body.String(), "Current heap size:") {
		t.Error("expected heap size information in text response")
	}
	if !strings.Contains(w.Body.String(), "Allocation key:") {
		t.Error("expected allocation key in text response")
	}

	// Clean up any persisting allocations by extracting key from text response
	body := w.Body.String()
	if keyIndex := strings.Index(body, "Allocation key: "); keyIndex != -1 {
		keyStart := keyIndex + len("Allocation key: ")
		keyEnd := strings.Index(body[keyStart:], "\n")
		if keyEnd != -1 {
			allocKey := body[keyStart : keyStart+keyEnd]
			deallocateMemory(allocKey)
		}
	}
}

func TestMemoryEndpoint_ContextCancel(t *testing.T) {
	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/memory?size=20&duration=300", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Start the handler in a goroutine
	done := make(chan bool)
	go func() {
		MemoryHandler(w, req)
		done <- true
	}()

	// Wait a bit for the allocation to happen
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for the handler to complete
	select {
	case <-done:
		// Handler completed
	case <-time.After(2 * time.Second):
		t.Error("handler did not complete after context cancellation")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
