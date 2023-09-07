package cmd

import (
	"encoding/json"
	"net/http"
	"os"
)

// list all environment variables
func InfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(os.Environ())
}
