package memory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// MemoryParams holds parameters for the memory endpoint.
type MemoryParams struct {
	Size     int `json:"size"`     // in MB
	Duration int `json:"duration"` // in seconds, 0 means forever
}

var (
	memoryBlocks = make(map[string][][]byte)
	memoryMutex  sync.RWMutex
)

// MemoryHandler generates memory utilization based on specified parameters.
func MemoryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context() // Only use request context for logging in this function
	
	params := MemoryParams{
		Size:     100, // Default 100MB
		Duration: 60,  // Default 60 seconds
	}

	// Parse parameters based on method
	if r.Method == http.MethodGet {
		sizeStr := r.URL.Query().Get("size")
		if sizeStr != "" {
			s, err := strconv.Atoi(sizeStr)
			if err == nil {
				params.Size = s
			}
		}
		durationStr := r.URL.Query().Get("duration")
		if durationStr != "" {
			d, err := strconv.Atoi(durationStr)
			if err == nil {
				params.Duration = d
			}
		}
	} else if r.Method == http.MethodPost {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&params); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to decode memory parameters from JSON body")
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
	}

	// Validate parameters
	if params.Size < 1 || params.Size > 8192 { // Max 8GB
		log.Ctx(ctx).Warn().Int("size", params.Size).Msg("invalid memory size, defaulting to 100MB")
		params.Size = 100
	}
	if params.Duration < 0 || params.Duration > 3600 { // Max 1 hour
		log.Ctx(ctx).Warn().Int("duration", params.Duration).Msg("invalid duration, defaulting to 60 seconds")
		params.Duration = 60
	}

	log.Ctx(ctx).Info().Int("size_mb", params.Size).Int("duration", params.Duration).Msg("allocating memory")

	// Generate unique key for this allocation
	allocKey := fmt.Sprintf("%s-%d", time.Now().Format("20060102-150405"), params.Size)

	// Allocate memory
	err := allocateMemory(allocKey, params.Size)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to allocate memory")
		http.Error(w, "Failed to allocate memory", http.StatusInternalServerError)
		return
	}

	// If duration is 0, keep memory allocated indefinitely
	if params.Duration > 0 {
		go func() {
			time.Sleep(time.Duration(params.Duration) * time.Second)
			deallocateMemory(allocKey)
			log.Info().Str("alloc_key", allocKey).Msg("memory deallocated after timeout")
		}()
	}

	// Get current memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Determine response format
	format := r.URL.Query().Get("format")
	if format == "text" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Allocated %dMB of memory for %d seconds\nCurrent heap size: %.2fMB\nAllocation key: %s\n", 
			params.Size, params.Duration, float64(memStats.HeapAlloc)/1024/1024, allocKey)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"size_mb":          fmt.Sprintf("%d", params.Size),
			"duration":         fmt.Sprintf("%d", params.Duration),
			"allocation_key":   allocKey,
			"current_heap_mb":  fmt.Sprintf("%.2f", float64(memStats.HeapAlloc)/1024/1024),
			"message":          fmt.Sprintf("Allocated %dMB of memory for %d seconds", params.Size, params.Duration),
		})
	}
}

// allocateMemory allocates the specified amount of memory in MB.
func allocateMemory(key string, sizeMB int) error {
	memoryMutex.Lock()
	defer memoryMutex.Unlock()

	// Allocate memory in chunks to avoid single large allocation issues
	chunkSize := 10 * 1024 * 1024 // 10MB chunks
	totalBytes := sizeMB * 1024 * 1024
	numChunks := totalBytes / chunkSize
	remainder := totalBytes % chunkSize

	blocks := make([][]byte, 0, numChunks+1)

	// Allocate full chunks
	for i := 0; i < numChunks; i++ {
		block := make([]byte, chunkSize)
		// Fill with random data to prevent optimization
		for j := range block {
			block[j] = byte(i + j)
		}
		blocks = append(blocks, block)
	}

	// Allocate remainder if any
	if remainder > 0 {
		block := make([]byte, remainder)
		for j := range block {
			block[j] = byte(numChunks + j)
		}
		blocks = append(blocks, block)
	}

	memoryBlocks[key] = blocks
	return nil
}

// deallocateMemory deallocates memory associated with the given key.
func deallocateMemory(key string) {
	memoryMutex.Lock()
	defer memoryMutex.Unlock()

	if blocks, exists := memoryBlocks[key]; exists {
		// Clear references to help GC
		for i := range blocks {
			blocks[i] = nil
		}
		delete(memoryBlocks, key)
		runtime.GC() // Force garbage collection
	}
}

// GetMemoryStats returns current memory allocation statistics.
func GetMemoryStats() map[string]interface{} {
	memoryMutex.RLock()
	defer memoryMutex.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	activeAllocations := make(map[string]int)
	totalAllocatedMB := 0
	
	for key, blocks := range memoryBlocks {
		sizeMB := 0
		for _, block := range blocks {
			sizeMB += len(block)
		}
		sizeMB = sizeMB / 1024 / 1024
		activeAllocations[key] = sizeMB
		totalAllocatedMB += sizeMB
	}

	return map[string]interface{}{
		"active_allocations":     activeAllocations,
		"total_allocated_mb":     totalAllocatedMB,
		"current_heap_mb":        float64(memStats.HeapAlloc) / 1024 / 1024,
		"total_heap_mb":          float64(memStats.HeapSys) / 1024 / 1024,
		"gc_count":               memStats.NumGC,
	}
}
