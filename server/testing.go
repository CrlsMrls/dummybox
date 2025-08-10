package server

import (
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/crlsmrls/dummybox/config"
	"github.com/crlsmrls/dummybox/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// The file provides utilities for integration testing:
// - `server.NewTestServerWithRecorder(cfg, logWriter, registry)`: Creates a server for fast integration tests
// - `server.NewTestServer(cfg, logWriter, registry)`: Creates a full HTTP test server for end-to-end testing
// - `srv.ServeHTTP(responseRecorder, request)`: Direct testing with httptest.ResponseRecorder

// TestServer wraps a Server for testing purposes.
type TestServer struct {
	*Server
	HTTPServer *httptest.Server
}

// NewTestServer creates a new test server with the given configuration.
// This is the recommended way to create servers for integration testing.
func NewTestServer(cfg *config.Config, logWriter io.Writer, reg *prometheus.Registry) *TestServer {
	if reg == nil {
		reg = metrics.InitMetrics()
	}
	
	server := New(cfg, logWriter, reg)
	httpServer := httptest.NewServer(server.router)
	
	return &TestServer{
		Server:     server,
		HTTPServer: httpServer,
	}
}

// NewTestServerWithRecorder creates a test server that uses httptest.ResponseRecorder
// instead of a real HTTP server. This is faster for unit-style integration tests.
func NewTestServerWithRecorder(cfg *config.Config, logWriter io.Writer, reg *prometheus.Registry) *Server {
	if reg == nil {
		reg = metrics.InitMetrics()
	}
	
	return New(cfg, logWriter, reg)
}

// ServeHTTP allows the server to be used directly with httptest.ResponseRecorder.
func (s *Server) ServeHTTP(recorder *httptest.ResponseRecorder, request *http.Request) {
	s.router.ServeHTTP(recorder, request)
}
