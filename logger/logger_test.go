package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
)

func TestInitLogger(t *testing.T) {
	tests := []struct {
		levelStr string
		expected zerolog.Level
	}{
		{"debug", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"invalid", zerolog.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.levelStr, func(t *testing.T) {
			InitLogger(tt.levelStr, nil)
			if zerolog.GlobalLevel() != tt.expected {
				t.Errorf("Expected global level %v, got %v", tt.expected, zerolog.GlobalLevel())
			}
		})
	}
}

func TestLogger_OutputFormat(t *testing.T) {
	var buf bytes.Buffer
	InitLogger("info", &buf)

	logger := FromContext(context.Background())
	logger.Info().Msg("test message")

	var logOutput map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logOutput); err != nil {
		t.Fatalf("Failed to unmarshal log output: %v", err)
	}

	requiredFields := []string{"level", "message", "source", "time"}
	for _, field := range requiredFields {
		if _, ok := logOutput[field]; !ok {
			t.Errorf("Log output missing required field: %s", field)
		}
	}

	if logOutput["level"] != "info" {
		t.Errorf("Expected level 'info', got '%v'", logOutput["level"])
	}
	if logOutput["message"] != "test message" {
		t.Errorf("Expected message 'test message', got '%v'", logOutput["message"])
	}
}

func TestLogger_WithCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	InitLogger("info", &buf)

	ctx := context.Background()
	correlationID := "test-id-123"
	ctx, logger := WithCorrelationID(ctx, correlationID)

	logger.Warn().Msg("a correlated message")

	var logOutput map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logOutput); err != nil {
		t.Fatalf("Failed to unmarshal log output: %v", err)
	}

	if id, ok := logOutput["correlation_id"]; !ok {
		t.Error("Log output missing correlation_id field")
	} else if id != correlationID {
		t.Errorf("Expected correlation_id '%s', got '%v'", correlationID, id)
	}

	if logOutput["level"] != "warn" {
		t.Errorf("Expected level 'warn', got '%v'", logOutput["level"])
	}
}
