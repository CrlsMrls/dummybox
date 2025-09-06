package respond

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

// RespondParams holds parameters for the respond endpoint.
type RespondParams struct {
	Duration int               `json:"duration"` // in seconds
	Code     int               `json:"code"`
	Headers  map[string]string `json:"headers"` // custom HTTP response headers
}

// RespondHandler introduces a configurable delay, returns a specified status code,
// and includes custom properties in the response.
func RespondHandler(w http.ResponseWriter, r *http.Request) {
	params := RespondParams{
		Duration: 0,                       // Default duration
		Code:     200,                     // Default status code
		Headers:  make(map[string]string), // Default empty headers
	}

	// Parse parameters based on method
	if r.Method == http.MethodGet {
		values := r.URL.Query()

		params.Duration = parseDuration(values)
		params.Code = parseCode(values)
		params.Headers = parseHeaders(values)
	} else if r.Method == http.MethodPost {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&params); err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("failed to decode respond parameters from JSON body")
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		// Initialize headers if nil
		if params.Headers == nil {
			params.Headers = make(map[string]string)
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

	log.Ctx(r.Context()).Info().
		Int("duration", params.Duration).
		Int("code", params.Code).
		Int("headers_count", len(params.Headers)).
		Msg("responding with custom parameters")

	// Introduce delay
	if params.Duration > 0 {
		time.Sleep(time.Duration(params.Duration) * time.Second)
	}

	// Add custom headers to response
	for headerName, headerValue := range params.Headers {
		w.Header().Set(headerName, headerValue)
	}

	// Determine response format
	format := r.URL.Query().Get("format")
	if format == "text" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(params.Code)

		responseText := fmt.Sprintf("Responded after %d seconds with status code %d\n", params.Duration, params.Code)
		if len(params.Headers) > 0 {
			responseText += "Custom Headers:\n"
			for key, value := range params.Headers {
				responseText += fmt.Sprintf("  %s: %s\n", key, value)
			}
		}
		fmt.Fprint(w, responseText)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(params.Code)

		response := map[string]interface{}{
			"duration": fmt.Sprintf("%d", params.Duration),
			"code":     fmt.Sprintf("%d", params.Code),
			"message":  fmt.Sprintf("Responded after %d seconds with status code %d", params.Duration, params.Code),
		}

		// Add custom headers info to the response body for visibility
		if len(params.Headers) > 0 {
			response["headers"] = params.Headers
		}

		json.NewEncoder(w).Encode(response)
	}
}

// parseDuration extracts and validates the duration parameter from query values
func parseDuration(values url.Values) int {
	durationStr := values.Get("duration")
	if durationStr != "" {
		if d, err := strconv.Atoi(durationStr); err == nil {
			return d
		}
	}
	return 0 // default value
}

// parseCode extracts and validates the status code parameter from query values
func parseCode(values url.Values) int {
	codeStr := values.Get("code")
	if codeStr != "" {
		if c, err := strconv.Atoi(codeStr); err == nil {
			return c
		}
	}
	return 200 // default value
}

// parseHeaders extracts custom headers from query parameters using repeated parameter names
// Expected format: header_name=HeaderName&header_value=HeaderValue&header_name=AnotherHeader&header_value=AnotherValue
func parseHeaders(values url.Values) map[string]string {
	headers := make(map[string]string)

	headerNames := values["header_name"]
	headerValues := values["header_value"]

	// Pair up names and values
	for i := range headerNames {
		if i < len(headerValues) {
			headers[headerNames[i]] = headerValues[i]
		}
	}

	return headers
}
