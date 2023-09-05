package main

import (
	"encoding/json"
	"net/http"
	"os"
)

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	version := os.Getenv("VERSION")

	if version == "" {
		version = "VERSION not set"
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"version": version})
}
