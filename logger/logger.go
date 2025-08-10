package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// InitLogger initializes the global logger
func InitLogger(level string, writer io.Writer) {
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	if writer == nil {
		writer = os.Stdout
	}

	zerolog.SetGlobalLevel(logLevel)
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.CallerFieldName = "source"

	log := zerolog.New(writer).With().Timestamp().Caller().Logger()
	zerolog.DefaultContextLogger = &log
}

// FromContext returns a logger from the context, or the default logger if none is found
func FromContext(ctx context.Context) *zerolog.Logger {
	logger := zerolog.Ctx(ctx)
	// If no logger is found in context, Ctx returns a disabled logger.
	// We'll check if it's disabled and if so, return the default logger.
	if logger.GetLevel() == zerolog.Disabled {
		defLogger := zerolog.DefaultContextLogger
		if defLogger != nil {
			return defLogger
		}
		// As a final fallback, create a new one, though InitLogger should have been called.
		l := zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
		return &l
	}
	return logger
}

// WithCorrelationID returns a new context and a logger with the correlation ID field.
func WithCorrelationID(ctx context.Context, correlationID string) (context.Context, *zerolog.Logger) {
	logger := FromContext(ctx).With().Str("correlation_id", correlationID).Logger()
	return logger.WithContext(ctx), &logger
}
