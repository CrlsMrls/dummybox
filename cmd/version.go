package cmd

import (
	"encoding/json"
	"net/http"
)

var Version = "development"

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"version": Version})
}
