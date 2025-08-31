package cpu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CPUIntensity represents the intensity level for CPU load generation.
type CPUIntensity string

const (
	Light   CPUIntensity = "light"   // Minimal CPU stress
	Medium  CPUIntensity = "medium"  // Moderate CPU stress
	Heavy   CPUIntensity = "heavy"   // High CPU stress
	Extreme CPUIntensity = "extreme" // Maximum CPU stress
)

// CPUParams holds parameters for the CPU endpoint.
type CPUParams struct {
	Intensity string `json:"intensity"` // light, medium, heavy, extreme
	Duration  int    `json:"duration"`  // in seconds, 0 means forever
}

// IntensityConfig defines the work characteristics for each intensity level.
type IntensityConfig struct {
	WorkSize      int           `json:"work_size"`      // complexity of CPU work
	WorkDuration  time.Duration `json:"work_duration"`  // how long to work continuously
	SleepDuration time.Duration `json:"sleep_duration"` // how long to sleep between work
	Description   string        `json:"description"`    // human-readable description
}

// CPULoadGenerator defines the interface for CPU load generation.
type CPULoadGenerator interface {
	DoWork(workSize int) int
	Sleep(duration time.Duration)
}

// ProductionCPULoadGenerator is the real implementation for production.
type ProductionCPULoadGenerator struct{}

func (p *ProductionCPULoadGenerator) DoWork(workSize int) int {
	return calculatePrimes(workSize)
}

func (p *ProductionCPULoadGenerator) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

var (
	cpuJobs         = make(map[string]context.CancelFunc)
	cpuMutex        sync.RWMutex
	jobCounter      int64
	loadGenerator   CPULoadGenerator = &ProductionCPULoadGenerator{}
	intensityLevels = map[CPUIntensity]IntensityConfig{
		Light: {
			WorkSize:      5000,
			WorkDuration:  100 * time.Millisecond,
			SleepDuration: 400 * time.Millisecond,
			Description:   "Light CPU stress - minimal system impact",
		},
		Medium: {
			WorkSize:      15000,
			WorkDuration:  250 * time.Millisecond,
			SleepDuration: 250 * time.Millisecond,
			Description:   "Medium CPU stress - moderate system load",
		},
		Heavy: {
			WorkSize:      30000,
			WorkDuration:  400 * time.Millisecond,
			SleepDuration: 100 * time.Millisecond,
			Description:   "Heavy CPU stress - high system load",
		},
		Extreme: {
			WorkSize:      50000,
			WorkDuration:  500 * time.Millisecond,
			SleepDuration: 0, // no sleep - continuous work
			Description:   "Extreme CPU stress - maximum system load",
		},
	}
)

// SetCPULoadGenerator allows dependency injection for testing.
func SetCPULoadGenerator(generator CPULoadGenerator) {
	loadGenerator = generator
}

// GetIntensityConfig returns the configuration for a given intensity level.
func GetIntensityConfig(intensity CPUIntensity) (IntensityConfig, bool) {
	config, exists := intensityLevels[intensity]
	return config, exists
}

// ValidateIntensity checks if the intensity string is valid and returns the CPUIntensity.
func ValidateIntensity(intensityStr string) (CPUIntensity, bool) {
	intensity := CPUIntensity(intensityStr)
	_, exists := intensityLevels[intensity]
	return intensity, exists
}

// CPUHandler generates CPU utilization based on specified parameters.
func CPUHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	params := CPUParams{
		Intensity: string(Medium), // Default medium intensity
		Duration:  60,             // Default 60 seconds
	}

	// Parse parameters based on method
	if r.Method == http.MethodGet {
		intensityStr := r.URL.Query().Get("intensity")
		if intensityStr != "" {
			params.Intensity = intensityStr
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
			log.Ctx(ctx).Error().Err(err).Msg("failed to decode CPU parameters from JSON body")
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
	}

	// Validate intensity
	intensity, valid := ValidateIntensity(params.Intensity)
	if !valid {
		log.Ctx(ctx).Warn().Str("intensity", params.Intensity).Msg("invalid CPU intensity, defaulting to medium")
		intensity = Medium
		params.Intensity = string(Medium)
	}

	// Validate duration
	if params.Duration < 0 || params.Duration > 3600 { // Max 1 hour
		log.Ctx(ctx).Warn().Int("duration", params.Duration).Msg("invalid duration, defaulting to 60 seconds")
		params.Duration = 60
	}

	config, _ := GetIntensityConfig(intensity)
	log.Ctx(ctx).Info().
		Str("intensity", params.Intensity).
		Int("duration", params.Duration).
		Str("description", config.Description).
		Msg("generating CPU load")

	// Generate unique key for this CPU load job
	cpuMutex.Lock()
	jobCounter++
	jobKey := fmt.Sprintf("cpu-job-%d-%s", jobCounter, time.Now().Format("20060102-150405"))
	cpuMutex.Unlock()

	// Start CPU load generation
	jobCtx, jobCancel := context.WithCancel(context.Background())
	cpuMutex.Lock()
	cpuJobs[jobKey] = jobCancel
	cpuMutex.Unlock()

	err := generateCPULoad(jobCtx, jobKey, intensity)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to start CPU load generation")
		http.Error(w, "Failed to generate CPU load", http.StatusInternalServerError)
		return
	}

	// If duration is 0, keep CPU load running indefinitely
	if params.Duration > 0 {
		go func() {
			time.Sleep(time.Duration(params.Duration) * time.Second)
			stopCPULoad(jobKey)
			log.Info().Str("job_key", jobKey).Msg("CPU load stopped after timeout")
		}()
	}

	// Determine response format
	format := r.URL.Query().Get("format")
	if format == "text" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Generating %s CPU load for %d seconds\nJob key: %s\nWorkers: %d\nDescription: %s\n",
			params.Intensity, params.Duration, jobKey, runtime.NumCPU(), config.Description)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"intensity":   params.Intensity,
			"duration":    params.Duration,
			"job_key":     jobKey,
			"workers":     runtime.NumCPU(),
			"description": config.Description,
			"config":      config,
			"message":     fmt.Sprintf("Generating %s CPU load for %d seconds", params.Intensity, params.Duration),
		})
	}
}

// generateCPULoad starts CPU load generation with the specified intensity.
func generateCPULoad(ctx context.Context, jobKey string, intensity CPUIntensity) error {
	config, exists := GetIntensityConfig(intensity)
	if !exists {
		return fmt.Errorf("unknown intensity level: %s", intensity)
	}

	numWorkers := runtime.NumCPU()
	
	log.Info().
		Str("job_key", jobKey).
		Int("workers", numWorkers).
		Str("intensity", string(intensity)).
		Int("work_size", config.WorkSize).
		Dur("work_duration", config.WorkDuration).
		Dur("sleep_duration", config.SleepDuration).
		Msg("starting CPU load workers")

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		go cpuWorker(ctx, jobKey, i, config, loadGenerator)
	}

	return nil
}

// cpuWorker runs the CPU load generation loop for a single worker.
func cpuWorker(ctx context.Context, jobKey string, workerID int, config IntensityConfig, generator CPULoadGenerator) {
	defer func() {
		log.Debug().
			Str("job_key", jobKey).
			Int("worker_id", workerID).
			Msg("CPU worker stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Perform CPU-intensive work for the configured duration
			start := time.Now()
			for time.Since(start) < config.WorkDuration {
				_ = generator.DoWork(config.WorkSize)
				// Continue doing work until the work duration is reached
			}
			
			// Sleep between work cycles (if configured)
			if config.SleepDuration > 0 {
				generator.Sleep(config.SleepDuration)
			}
		}
	}
}

// calculatePrimes performs CPU-intensive work by calculating prime numbers up to n.
func calculatePrimes(n int) int {
	count := 0
	for i := 2; i <= n; i++ {
		isPrime := true
		for j := 2; j*j <= i; j++ {
			if i%j == 0 {
				isPrime = false
				break
			}
		}
		if isPrime {
			count++
		}
	}
	return count
}

// stopCPULoad stops the CPU load generation for the given job key.
func stopCPULoad(jobKey string) {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()

	if cancel, exists := cpuJobs[jobKey]; exists {
		cancel()
		delete(cpuJobs, jobKey)
		log.Info().Str("job_key", jobKey).Msg("CPU load job stopped and cleaned up")
	}
}

// GetCPUStats returns current CPU load job statistics.
func GetCPUStats() map[string]interface{} {
	cpuMutex.RLock()
	defer cpuMutex.RUnlock()

	activeJobs := make([]string, 0, len(cpuJobs))
	for jobKey := range cpuJobs {
		activeJobs = append(activeJobs, jobKey)
	}

	return map[string]interface{}{
		"active_jobs":       activeJobs,
		"total_jobs":        len(activeJobs),
		"cpu_count":         runtime.NumCPU(),
		"goroutines":        runtime.NumGoroutine(),
		"intensity_levels":  []string{"light", "medium", "heavy", "extreme"},
		"default_intensity": "medium",
	}
}

// GetAvailableIntensities returns all available intensity levels with their configurations.
func GetAvailableIntensities() map[CPUIntensity]IntensityConfig {
	return intensityLevels
}
