package logging

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogging initializes the logging system with the specified level
func InitLogging(levelStr string) {
	// Parse the log level
	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid log level '%s', defaulting to 'error'\n", levelStr)
		level = zerolog.ErrorLevel
	}

	// Configure zerolog
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	})
}

// TODO: Add file logging option based on config?
