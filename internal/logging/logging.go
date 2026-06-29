// Package logging configures DiffPal's process-wide zerolog logger.
package logging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const validationJSONField = "validation_json_full"

// Init configures zerolog for the current process and returns a context carrying
// the configured logger.
func Init(ctx context.Context, debug bool) context.Context {
	return initWithWriter(ctx, debug, os.Stderr)
}

func initWithWriter(ctx context.Context, debug bool, out io.Writer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if out == nil {
		out = io.Discard
	}
	level := zerolog.InfoLevel
	if debug {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339

	writer := zerolog.ConsoleWriter{
		Out:        out,
		NoColor:    true,
		TimeFormat: time.RFC3339,
		FieldsExclude: []string{
			validationJSONField,
		},
		FormatExtra: formatProviderResponse,
	}
	logger := zerolog.New(writer).With().Timestamp().Logger()
	log.Logger = logger
	zerolog.DefaultContextLogger = &log.Logger
	return log.Logger.WithContext(ctx)
}

// DebugEnabled reports whether debug logging is enabled globally.
func DebugEnabled() bool {
	return zerolog.GlobalLevel() <= zerolog.DebugLevel
}

func formatProviderResponse(evt map[string]interface{}, buf *bytes.Buffer) error {
	if evt == nil || buf == nil {
		return nil
	}
	level, _ := evt[zerolog.LevelFieldName].(string)
	if level != zerolog.DebugLevel.String() {
		return nil
	}
	raw, ok := evt[validationJSONField].(string)
	if !ok || raw == "" {
		return nil
	}
	if buf.Len() > 0 {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(buf, "provider response:"); err != nil {
		return err
	}
	_, err := fmt.Fprint(buf, raw)
	return err
}
