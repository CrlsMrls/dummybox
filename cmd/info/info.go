package info

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/crlsmrls/dummybox/cmd"
	"github.com/crlsmrls/dummybox/metrics"
	"github.com/rs/zerolog/log"
)

// Info holds all the application and system information.
type Info struct {
	Application struct {
		Version   string `json:"version"`
		BuildDate string `json:"build_date"`
		GoVersion string `json:"go_version"`
		GitCommit string `json:"git_commit"`
	} `json:"application"`
	Process struct {
		Pid       int       `json:"pid"`
		StartTime time.Time `json:"start_time"`
		Uptime    string    `json:"uptime"`
		OS        string    `json:"os"`
		Arch      string    `json:"arch"`
	} `json:"process"`
	User struct {
		UID string `json:"uid"`
		GID string `json:"gid"`
	} `json:"user"`
	ClusterPosition struct {
		ContainerID      string `json:"container_id"`
		ImageName        string `json:"image_name"`
		ImageTag         string `json:"image_tag"`
		NodeName         string `json:"node_name"`
		PodName          string `json:"pod_name"`
		Namespace        string `json:"namespace"`
		ResourceLimits   string `json:"resource_limits"`
		ResourceRequests string `json:"resource_requests"`
	} `json:"cluster_position"`
	Metrics struct {
		Summary     string                 `json:"summary"`
		Details     map[string]interface{} `json:"details"`
		LastUpdated time.Time              `json:"last_updated"`
	} `json:"metrics"`
}

var startTime = time.Now()

// InfoHandler returns application and system information.
func InfoHandler(w http.ResponseWriter, r *http.Request) {
	info := Info{}

	// Application Info
	info.Application.Version = cmd.Version
	info.Application.BuildDate = cmd.BuildDate
	info.Application.GoVersion = runtime.Version()
	info.Application.GitCommit = cmd.GitCommit

	// Process Info
	info.Process.Pid = os.Getpid()
	info.Process.StartTime = startTime
	info.Process.Uptime = formatUptime(time.Since(startTime))
	info.Process.OS = runtime.GOOS
	info.Process.Arch = runtime.GOARCH

	// User Info
	currentUser, err := user.Current()
	if err == nil {
		info.User.UID = currentUser.Uid
		info.User.GID = currentUser.Gid
	} else {
		log.Ctx(r.Context()).Warn().Err(err).Msg("failed to get current user info")
		info.User.UID = "not available"
		info.User.GID = "not available"
	}

	// Cluster Position Info (from environment variables)
	info.ClusterPosition.ContainerID = getEnvOrDefault("HOSTNAME", "not available")
	info.ClusterPosition.ImageName = getEnvOrDefault("DUMMYBOX_IMAGE_NAME", "not available")
	info.ClusterPosition.ImageTag = getEnvOrDefault("DUMMYBOX_IMAGE_TAG", "not available")
	info.ClusterPosition.NodeName = getEnvOrDefault("NODE_NAME", "not available")
	info.ClusterPosition.PodName = getEnvOrDefault("POD_NAME", "not available")
	info.ClusterPosition.Namespace = getEnvOrDefault("NAMESPACE", "not available")
	info.ClusterPosition.ResourceLimits = getEnvOrDefault("DUMMYBOX_RESOURCE_LIMITS", "not available")
	info.ClusterPosition.ResourceRequests = getEnvOrDefault("DUMMYBOX_RESOURCE_REQUESTS", "not available")

	// Metrics Summary
	metricsData := metrics.GetMetricsInfo()
	info.Metrics.Details = metricsData
	info.Metrics.Summary = generateMetricsSummary(metricsData)
	info.Metrics.LastUpdated = time.Now()

	// Determine response type
	acceptHeader := r.Header.Get("Accept")
	if strings.Contains(acceptHeader, "text/html") {
		renderHTML(w, r, info)
	} else {
		renderJSON(w, r, info)
	}
}

func renderJSON(w http.ResponseWriter, r *http.Request, info Info) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(info); err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to encode info to JSON")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func renderHTML(w http.ResponseWriter, r *http.Request, info Info) {
	// Get the absolute path to the web directory
	_, filename, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(filename)
	webDir := filepath.Join(currentDir, "..", "..", "web") // Adjust path for cmd/info
	indexPath := filepath.Join(webDir, "info.html")

	tmpl, err := template.ParseFiles(indexPath)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to parse info.html template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, info); err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to execute info.html template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

// generateMetricsSummary creates a human-readable summary from metrics data
func generateMetricsSummary(metricsData map[string]interface{}) string {
	if status, exists := metricsData["status"]; exists {
		return fmt.Sprintf("Metrics status: %v", status)
	}

	var summaryParts []string

	// HTTP metrics summary
	if httpData, exists := metricsData["http"].(map[string]interface{}); exists {
		if totalReqs, exists := httpData["total_requests"].(float64); exists {
			summaryParts = append(summaryParts, fmt.Sprintf("HTTP requests: %.0f", totalReqs))
		}
	}

	// Runtime metrics summary
	if runtimeData, exists := metricsData["runtime"].(map[string]interface{}); exists {
		if goroutines, exists := runtimeData["goroutines"].(int); exists {
			summaryParts = append(summaryParts, fmt.Sprintf("Goroutines: %d", goroutines))
		}

		if allocBytes, exists := runtimeData["allocated_bytes"].(int64); exists {
			summaryParts = append(summaryParts, fmt.Sprintf("Memory allocated: %.2f MB", float64(allocBytes)/1024/1024))
		}

		if resMemBytes, exists := runtimeData["resident_memory_bytes"].(int64); exists {
			summaryParts = append(summaryParts, fmt.Sprintf("Resident memory: %.2f MB", float64(resMemBytes)/1024/1024))
		}
	}

	// Total metrics count
	if totalMetrics, exists := metricsData["total_metrics_collected"].(int); exists {
		summaryParts = append(summaryParts, fmt.Sprintf("Total metric families: %d", totalMetrics))
	}

	if len(summaryParts) == 0 {
		return "No metrics data available"
	}

	return strings.Join(summaryParts, " | ")
}

// formatUptime converts a duration to a human-readable uptime string
func formatUptime(duration time.Duration) string {
	totalSeconds := int(duration.Seconds())

	if totalSeconds < 60 {
		return fmt.Sprintf("%ds", totalSeconds)
	}

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}

	return fmt.Sprintf("%dm %ds", minutes, seconds)
}
