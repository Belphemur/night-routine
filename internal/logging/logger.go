package logging

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

// Initialize sets up the global logger with the specified configuration
func Initialize(isDevelopment bool) {
	// Set global time field format
	zerolog.TimeFieldFormat = time.RFC3339
	// Set stack trace marshaler
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// Configure output writer based on environment
	var output io.Writer = os.Stdout
	if isDevelopment {
		// Use pretty console writer for development
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		}
	}

	// Set global logger
	log.Logger = zerolog.New(output).
		With().
		Timestamp().
		Caller(). // Add caller information
		Logger()

	// Set default log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if isDevelopment {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// GetLogger returns a logger with the component field set
func GetLogger(component string) zerolog.Logger {
	return log.With().Str("component", component).Logger()
}

// SetLogLevel sets the global log level
func SetLogLevel(level string) {
	switch level {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel) // Default to InfoLevel if invalid
	}
}
