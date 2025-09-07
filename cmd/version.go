package cmd

import (
	"encoding/json"
	"net/http"
)

// Application information, populated at build time
var (
	Version   = "development"
	BuildDate = "unknown"
	GoVersion = "unknown"
	GitCommit = "unknown"
)

// VersionInfo holds the complete application version information
type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	GitCommit string `json:"git_commit"`
}

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	info := VersionInfo{
		Version:   Version,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		GitCommit: GitCommit,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, "Failed to encode version information", http.StatusInternalServerError)
		return
	}
}
