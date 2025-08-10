package delay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

// DelayParams holds parameters for the delay endpoint.
type DelayParams struct {
	Duration int `json:"duration"` // in seconds
	Code     int `json:"code"`
}

// DelayHandler introduces a configurable delay and returns a specified status code.
func DelayHandler(w http.ResponseWriter, r *http.Request) {
	params := DelayParams{
		Duration: 0,   // Default duration
		Code:     200, // Default status code
	}

	// Parse parameters based on method
	if r.Method == http.MethodGet {
		durationStr := r.URL.Query().Get("duration")
		if durationStr != "" {
			d, err := strconv.Atoi(durationStr)
			if err == nil {
				params.Duration = d
			}
		}
		codeStr := r.URL.Query().Get("code")
		if codeStr != "" {
			c, err := strconv.Atoi(codeStr)
			if err == nil {
				params.Code = c
			}
		}
	} else if r.Method == http.MethodPost {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&params); err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("failed to decode delay parameters from JSON body")
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
	}

	// Validate parameters
	if params.Duration < 0 || params.Duration > 300 { // Max 5 minutes delay
		log.Ctx(r.Context()).Warn().Int("duration", params.Duration).Msg("invalid duration, defaulting to 0")
		params.Duration = 0
	}
	if params.Code < 100 || params.Code > 599 {
		log.Ctx(r.Context()).Warn().Int("code", params.Code).Msg("invalid status code, defaulting to 200")
		params.Code = 200
	}

	log.Ctx(r.Context()).Info().Int("duration", params.Duration).Int("code", params.Code).Msg("delaying response")

	// Introduce delay
	if params.Duration > 0 {
		time.Sleep(time.Duration(params.Duration) * time.Second)
	}

	// Determine response format
	format := r.URL.Query().Get("format")
	if format == "text" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(params.Code)
		fmt.Fprintf(w, "Delayed for %d seconds with status code %d\n", params.Duration, params.Code)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(params.Code)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"duration": params.Duration,
			"code":     params.Code,
			"message":  fmt.Sprintf("Delayed for %d seconds with status code %d", params.Duration, params.Code),
		})
	}
}
