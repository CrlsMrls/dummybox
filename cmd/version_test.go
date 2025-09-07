package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionHandler(t *testing.T) {
	// Set test values for version variables
	originalVersion := Version
	originalBuildDate := BuildDate
	originalGoVersion := GoVersion
	originalGitCommit := GitCommit

	// Set test values
	Version = "1.0.0"
	BuildDate = "2025-09-21T10:00:00Z"
	GoVersion = "go1.25.1"
	GitCommit = "abc123"

	// Restore original values after test
	defer func() {
		Version = originalVersion
		BuildDate = originalBuildDate
		GoVersion = originalGoVersion
		GitCommit = originalGitCommit
	}()

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	VersionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}

	var info VersionInfo
	if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if info.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", info.Version)
	}
	if info.BuildDate != "2025-09-21T10:00:00Z" {
		t.Errorf("expected build_date '2025-09-21T10:00:00Z', got '%s'", info.BuildDate)
	}
	if info.GoVersion != "go1.25.1" {
		t.Errorf("expected go_version 'go1.25.1', got '%s'", info.GoVersion)
	}
	if info.GitCommit != "abc123" {
		t.Errorf("expected git_commit 'abc123', got '%s'", info.GitCommit)
	}
}
