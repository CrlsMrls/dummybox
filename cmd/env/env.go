package env

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

// EnvParams holds parameters for the env endpoint.
type EnvParams struct {
	Format string `json:"format"` // json or text
}

// EnvHandler returns all environment variables.
func EnvHandler(w http.ResponseWriter, r *http.Request) {
	params := EnvParams{
		Format: "json", // Default format
	}

	// Parse parameters based on method
	if r.Method == http.MethodGet {
		if format := r.URL.Query().Get("format"); format != "" {
			params.Format = format
		}
	} else if r.Method == http.MethodPost {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&params); err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("failed to decode env request body")
			http.Error(w, "Bad Request: invalid JSON", http.StatusBadRequest)
			return
		}
	}

	// Validate format parameter
	if params.Format != "json" && params.Format != "text" {
		params.Format = "json" // Default to json for invalid values
	}

	// Get all environment variables
	envVars := os.Environ()

	// Parse into map for JSON response or keep as slice for text
	envMap := make(map[string]string)
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Determine response format
	if params.Format == "text" {
		renderTextResponse(w, r, envVars)
	} else {
		renderJSONResponse(w, r, envMap)
	}
}

func renderJSONResponse(w http.ResponseWriter, r *http.Request, envMap map[string]string) {
	response := map[string]interface{}{
		"format":                "json",
		"count":                 len(envMap),
		"environment_variables": envMap,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to encode env response to JSON")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func renderTextResponse(w http.ResponseWriter, r *http.Request, envVars []string) {
	// Sort environment variables for consistent output
	sort.Strings(envVars)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "Environment Variables (%d total):\n\n", len(envVars))
	for _, env := range envVars {
		fmt.Fprintf(w, "%s\n", env)
	}
}
