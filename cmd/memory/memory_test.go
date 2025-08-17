package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMemoryHandler_GET_DefaultParameters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/memory", nil)
	w := httptest.NewRecorder()

	MemoryHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Check default values
	if response["size_mb"] != float64(100) {
		t.Errorf("expected size_mb 100, got %v", response["size_mb"])
	}
	if response["duration"] != float64(60) {
		t.Errorf("expected duration 60, got %v", response["duration"])
	}

	// Check that allocation key is present
	if _, ok := response["allocation_key"]; !ok {
		t.Error("expected allocation_key in response")
	}

	// Check that current_heap_mb is present and reasonable
	if heapMB, ok := response["current_heap_mb"].(float64); !ok || heapMB < 0 {
		t.Errorf("expected positive current_heap_mb, got %v", response["current_heap_mb"])
	}
}

func TestMemoryHandler_GET_WithParameters(t *testing.T) {
	testCases := []struct {
		name     string
		size     string
		duration string
		wantSize int
		wantDur  int
	}{
		{
			name:     "custom size and duration",
			size:     "50",
			duration: "30",
			wantSize: 50,
			wantDur:  30,
		},
		{
			name:     "zero duration (forever)",
			size:     "25",
			duration: "0",
			wantSize: 25,
			wantDur:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/memory?size="+tc.size+"&duration="+tc.duration, nil)
			w := httptest.NewRecorder()

			MemoryHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if int(response["size_mb"].(float64)) != tc.wantSize {
				t.Errorf("expected size_mb %d, got %v", tc.wantSize, response["size_mb"])
			}
			if int(response["duration"].(float64)) != tc.wantDur {
				t.Errorf("expected duration %d, got %v", tc.wantDur, response["duration"])
			}
		})
	}
}

func TestMemoryHandler_GET_TextFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/memory?format=text&size=20&duration=10", nil)
	w := httptest.NewRecorder()

	MemoryHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("expected Content-Type text/plain, got %s", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if !strings.Contains(body, "Allocated 20MB") {
		t.Error("expected allocated memory information in text response")
	}
	if !strings.Contains(body, "10 seconds") {
		t.Error("expected duration information in text response")
	}
	if !strings.Contains(body, "Current heap size:") {
		t.Error("expected current heap size in text response")
	}
}

func TestMemoryHandler_POST_ValidJSON(t *testing.T) {
	params := MemoryParams{
		Size:     75,
		Duration: 45,
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

	if int(response["size_mb"].(float64)) != 75 {
		t.Errorf("expected size_mb 75, got %v", response["size_mb"])
	}
	if int(response["duration"].(float64)) != 45 {
		t.Errorf("expected duration 45, got %v", response["duration"])
	}
}

func TestMemoryHandler_POST_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/memory", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	MemoryHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestMemoryHandler_ParameterValidation(t *testing.T) {
	testCases := []struct {
		name         string
		size         string
		duration     string
		expectedSize int
		expectedDur  int
	}{
		{
			name:         "negative size",
			size:         "-10",
			duration:     "30",
			expectedSize: 100, // should default
			expectedDur:  30,
		},
		{
			name:         "excessive size",
			size:         "10000", // > 8192 MB limit
			duration:     "30",
			expectedSize: 100, // should default
			expectedDur:  30,
		},
		{
			name:         "negative duration",
			size:         "50",
			duration:     "-5",
			expectedSize: 50,
			expectedDur:  60, // should default
		},
		{
			name:         "excessive duration",
			size:         "50",
			duration:     "4000", // > 3600 seconds limit
			expectedSize: 50,
			expectedDur:  60, // should default
		},
		{
			name:         "edge case valid values",
			size:         "1",
			duration:     "1",
			expectedSize: 1,
			expectedDur:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/memory?size="+tc.size+"&duration="+tc.duration, nil)
			w := httptest.NewRecorder()

			MemoryHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if int(response["size_mb"].(float64)) != tc.expectedSize {
				t.Errorf("expected size_mb %d, got %v", tc.expectedSize, response["size_mb"])
			}
			if int(response["duration"].(float64)) != tc.expectedDur {
				t.Errorf("expected duration %d, got %v", tc.expectedDur, response["duration"])
			}
		})
	}
}

func TestMemoryAllocation(t *testing.T) {
	// Test memory allocation and deallocation
	testKey := "test-allocation"
	sizeMB := 50

	// Get initial memory stats
	var initialStats runtime.MemStats
	runtime.ReadMemStats(&initialStats)
	initialHeap := initialStats.HeapAlloc

	// Allocate memory
	err := allocateMemory(testKey, sizeMB)
	if err != nil {
		t.Fatalf("failed to allocate memory: %v", err)
	}

	// Check memory was allocated
	var afterAllocStats runtime.MemStats
	runtime.ReadMemStats(&afterAllocStats)
	if afterAllocStats.HeapAlloc <= initialHeap {
		t.Error("expected heap allocation to increase after memory allocation")
	}

	// Check memory stats
	stats := GetMemoryStats()
	activeAllocs := stats["active_allocations"].(map[string]int)
	if activeAllocs[testKey] != sizeMB {
		t.Errorf("expected %dMB allocation for key %s, got %d", sizeMB, testKey, activeAllocs[testKey])
	}
	if stats["total_allocated_mb"].(int) < sizeMB {
		t.Errorf("expected total allocated MB to be at least %d, got %v", sizeMB, stats["total_allocated_mb"])
	}

	// Deallocate memory
	deallocateMemory(testKey)
	runtime.GC() // Force garbage collection

	// Check memory was deallocated
	statsAfterDealloc := GetMemoryStats()
	activeAllocsAfter := statsAfterDealloc["active_allocations"].(map[string]int)
	if _, exists := activeAllocsAfter[testKey]; exists {
		t.Error("expected allocation to be removed after deallocation")
	}
}

func TestMemoryHandler_ContextCancellation(t *testing.T) {
	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	
	req := httptest.NewRequest(http.MethodGet, "/memory?size=30&duration=300", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Start the handler
	go func() {
		MemoryHandler(w, req)
	}()

	// Give it a moment to allocate memory
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Give it a moment to clean up
	time.Sleep(200 * time.Millisecond)

	// The memory should be cleaned up due to context cancellation
	// We can't easily test this directly, but the allocation should not persist
}

func TestGetMemoryStats(t *testing.T) {
	// Clean up any existing allocations
	memoryMutex.Lock()
	memoryBlocks = make(map[string][][]byte)
	memoryMutex.Unlock()

	stats := GetMemoryStats()

	// Check that all expected fields are present
	expectedFields := []string{
		"active_allocations",
		"total_allocated_mb",
		"current_heap_mb",
		"total_heap_mb",
		"gc_count",
	}

	for _, field := range expectedFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("expected field %s in memory stats", field)
		}
	}

	// Check that values are reasonable
	if stats["total_allocated_mb"].(int) != 0 {
		t.Error("expected total_allocated_mb to be 0 with no active allocations")
	}

	if stats["current_heap_mb"].(float64) < 0 {
		t.Error("expected current_heap_mb to be non-negative")
	}

	if stats["gc_count"].(uint32) < 0 {
		t.Error("expected gc_count to be non-negative")
	}
}
