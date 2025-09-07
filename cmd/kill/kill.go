package kill

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

// TestMode controls whether the handler actually calls os.Exit during tests
var TestMode = false

// KillParams holds parameters for the kill endpoint.
type KillParams struct {
	Delay int `json:"delay"` // delay in seconds before termination
	Code  int `json:"code"`  // exit code (0-255)
}

// KillHandler terminates the application with the specified exit code after the delay.
func KillHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	params := KillParams{
		Delay: 0, // Default: no delay
		Code:  0, // Default: exit code 0
	}

	// Parse parameters based on method
	if r.Method == http.MethodGet {
		if delayStr := r.URL.Query().Get("delay"); delayStr != "" {
			if d, err := strconv.Atoi(delayStr); err == nil {
				params.Delay = d
			}
		}
		if codeStr := r.URL.Query().Get("code"); codeStr != "" {
			if c, err := strconv.Atoi(codeStr); err == nil {
				params.Code = c
			}
		}
	} else if r.Method == http.MethodPost {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&params); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to decode kill parameters from JSON body")
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
	}

	// Validate parameters
	if params.Delay < 0 || params.Delay > 3600 { // Max 1 hour delay
		log.Ctx(ctx).Warn().Int("delay", params.Delay).Msg("invalid delay, defaulting to 0")
		params.Delay = 0
	}
	if params.Code < 0 || params.Code > 255 { // Valid exit code range
		log.Ctx(ctx).Warn().Int("code", params.Code).Msg("invalid exit code, defaulting to 0")
		params.Code = 0
	}

	log.Ctx(ctx).Info().
		Int("delay", params.Delay).
		Int("code", params.Code).
		Msg("kill request received")

	// Return response immediately
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"delay":  params.Delay,
		"code":   params.Code,
		"status": "termination scheduled",
	})

	// Schedule termination in background
	go func() {
		if params.Delay > 0 {
			log.Info().
				Int("delay", params.Delay).
				Int("code", params.Code).
				Msg("waiting before termination")
			time.Sleep(time.Duration(params.Delay) * time.Second)
		}

		log.Info().
			Int("code", params.Code).
			Msg("terminating process")

		// Don't actually exit during tests
		if !TestMode {
			os.Exit(params.Code)
		}
	}()
}
