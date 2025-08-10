package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crlsmrls/dummybox/config"
	"github.com/crlsmrls/dummybox/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

// Server holds the HTTP server and its configuration.
type Server struct {
	httpServer *http.Server
	router     *chi.Mux
	config     *config.Config
}

// New creates a new HTTP server.
func New(cfg *config.Config, logWriter io.Writer, reg *prometheus.Registry) *Server {
	r := chi.NewRouter()

	if logWriter == nil {
		logWriter = os.Stdout
	}
	// Create a zerolog logger instance that writes to the provided writer
	logger := zerolog.New(logWriter).With().Timestamp().Caller().Logger()

	// Set up middleware chain
	r.Use(
		// Inject zerolog logger into the request context
		hlog.NewHandler(logger),

		// Collect HTTP metrics
		metrics.HTTPMetricsMiddleware,

		// Log request details
		hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			hlog.FromRequest(r).Info().
				Str("method", r.Method).
				Str("url", r.URL.String()).
				Int("status", status).
				Int("size", size).
				Dur("duration", duration).
				Msg("request")
		}),

		// Add remote IP address to the logger
		hlog.RemoteAddrHandler("ip"),

		// Add user agent to the logger
		hlog.UserAgentHandler("user_agent"),

		// Add request ID to the logger
		middleware.RequestID,

		// Handle X-Correlation-ID header
		CorrelationIDMiddleware,

		// Recover from panics and log them
		middleware.Recoverer,
	)

	// Set up routes
	setupRoutes(r, cfg, reg)

	s := &Server{
		router: r,
		config: cfg,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      r,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
	}

	return s
}

// Start starts the HTTP server and handles graceful shutdown.
func (s *Server) Start() error {
	log.Info().Msgf("Starting server on port %d", s.config.Port)

	// Listen for OS signals for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Start server in a goroutine
	go func() {
		var err error
		if s.config.TLSCertFile != "" && s.config.TLSKeyFile != "" {
			log.Info().Msg("TLS enabled")
			err = s.httpServer.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
		} else {
			log.Info().Msg("TLS disabled")
			err = s.httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed to start")
		}
	}()

	// Wait for OS signal
	<-stop

	log.Info().Msg("Shutting down server...")

	// Create a context with a timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shut down the server gracefully
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server shutdown failed")
	}

	log.Info().Msg("Server gracefully stopped.")
	return nil
}
