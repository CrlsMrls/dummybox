package server

import (
	"html/template"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/crlsmrls/dummybox/cmd/cpu"
	"github.com/crlsmrls/dummybox/cmd/info"
	logcmd "github.com/crlsmrls/dummybox/cmd/log"
	"github.com/crlsmrls/dummybox/cmd/memory"
	"github.com/crlsmrls/dummybox/cmd/request"
	"github.com/crlsmrls/dummybox/cmd/respond"
	"github.com/crlsmrls/dummybox/config"
	"github.com/crlsmrls/dummybox/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// setupRoutes configures the application's routes.
func setupRoutes(router *chi.Mux, cfg *config.Config, reg *prometheus.Registry) {
	// Get the absolute path to the web directory
	_, filename, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(filename)
	webDir := filepath.Join(currentDir, "..", "web")
	indexPath := filepath.Join(webDir, "index.html")

	// Root endpoint
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(indexPath)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("failed to parse index.html template")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Placeholder data for the template
		data := struct {
			Version   string
			GoVersion string
			BuildDate string
			GitCommit string
		}{
			Version:   "v0.0.1",
			GoVersion: "go1.21",
			BuildDate: "2025-01-01",
			GitCommit: "abcdef12345",
		}

		w.Header().Set("Content-Type", "text/html")
		if err := tmpl.Execute(w, data); err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("failed to execute index.html template")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Placeholder for other routes from main.go
	router.HandleFunc("/positions", func(w http.ResponseWriter, r *http.Request) {
		log.Ctx(r.Context()).Info().Msg("positions handler called")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("positions"))
	})
	router.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		log.Ctx(r.Context()).Info().Msg("version handler called")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("version"))
	})
	router.Get("/info", info.InfoHandler)
	router.HandleFunc("/request", request.RequestHandler)

	// Command endpoints (protected with token auth)
	router.Route("/respond", func(r chi.Router) {
		r.Use(TokenAuthMiddleware(cfg))
		r.Get("/", respond.RespondHandler)
		r.Post("/", respond.RespondHandler)
	})

	router.Route("/log", func(r chi.Router) {
		r.Use(TokenAuthMiddleware(cfg))
		r.Get("/", logcmd.LogHandler)
		r.Post("/", logcmd.LogHandler)
	})

	router.Route("/memory", func(r chi.Router) {
		r.Use(TokenAuthMiddleware(cfg))
		r.Get("/", memory.MemoryHandler)
		r.Post("/", memory.MemoryHandler)
	})

	router.Route("/cpu", func(r chi.Router) {
		r.Use(TokenAuthMiddleware(cfg))
		r.Get("/", cpu.CPUHandler)
		r.Post("/", cpu.CPUHandler)
	})

	// Metrics endpoint
	router.Handle(cfg.MetricsPath, metrics.MetricsHandler(reg))
}
