package cmd

import (
	"fmt"
	"net/http"
)

// Application information, populated at build time
var (
	Version   = "development"
	BuildDate = "unknown"
	GoVersion = "unknown"
	GitCommit = "unknown"
)

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Version: %s\n", Version)
}
