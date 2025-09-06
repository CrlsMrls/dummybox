package request

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog/log"
)

// RequestInfo holds detailed HTTP request information.
type RequestInfo struct {
	Headers         map[string][]string `json:"headers"`
	Method          string              `json:"method"`
	URL             string              `json:"url"`
	QueryParameters map[string][]string `json:"query_parameters"`
	Body            string              `json:"body"`
	JWT             *JWTInfo            `json:"jwt,omitempty"`
}

// JWTInfo holds decoded JWT token information.
type JWTInfo struct {
	Header    map[string]interface{} `json:"header"`
	Payload   map[string]interface{} `json:"payload"`
	Signature string                 `json:"signature"`
}

var funcMap = template.FuncMap{
	"jsonify": func(data interface{}) (template.HTML, error) {
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return "", err
		}
		return template.HTML(b), nil
	},
	"sub": func(a, b int) int {
		return a - b
	},
}

// RequestHandler returns detailed HTTP request information.
func RequestHandler(w http.ResponseWriter, r *http.Request) {
	info := RequestInfo{
		Headers:         r.Header,
		Method:          r.Method,
		URL:             r.URL.String(),
		QueryParameters: r.URL.Query(),
	}

	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to read request body")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	info.Body = string(bodyBytes)

	// Parse JWT from Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
		if err == nil {
			info.JWT = &JWTInfo{
				Header:    token.Header,
				Payload:   token.Claims.(jwt.MapClaims),
				Signature: token.Signature,
			}
		} else {
			log.Ctx(r.Context()).Warn().Err(err).Msg("failed to parse JWT token")
		}
	}

	// Determine response type
	acceptHeader := r.Header.Get("Accept")
	if strings.Contains(acceptHeader, "text/html") {
		renderHTML(w, r, info)
	} else {
		renderJSON(w, r, info)
	}
}

func renderJSON(w http.ResponseWriter, r *http.Request, info RequestInfo) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(info); err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to encode request info to JSON")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func renderHTML(w http.ResponseWriter, r *http.Request, info RequestInfo) {
	// Get the absolute path to the web directory
	_, filename, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(filename)
	webDir := filepath.Join(currentDir, "..", "..", "web") // Adjust path for cmd/request
	indexPath := filepath.Join(webDir, "request.html")

	tmpl, err := template.New("request.html").Funcs(funcMap).ParseFiles(indexPath)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to parse request.html template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, info); err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to execute request.html template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
