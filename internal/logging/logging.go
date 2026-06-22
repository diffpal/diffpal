// Package logging configures DiffPal's process-wide zerolog logger.
package logging

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init configures zerolog for the current process and returns a context carrying
// the configured logger.
func Init(ctx context.Context, debug bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	level := zerolog.InfoLevel
	if debug {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	log.Logger = logger
	zerolog.DefaultContextLogger = &log.Logger
	return log.Logger.WithContext(ctx)
}

// DebugEnabled reports whether debug logging is enabled globally.
func DebugEnabled() bool {
	return zerolog.GlobalLevel() <= zerolog.DebugLevel
}
