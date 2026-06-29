package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestInitDefaultLevelIsInfo(t *testing.T) {
	prevLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(prevLevel)

	ctx := Init(context.Background(), false)
	if ctx == nil {
		t.Fatal("Init returned nil context")
	}
	if got := zerolog.GlobalLevel(); got != zerolog.InfoLevel {
		t.Fatalf("zerolog global level = %s, want %s", got, zerolog.InfoLevel)
	}
	if DebugEnabled() {
		t.Fatal("DebugEnabled() = true, want false")
	}
}

func TestInitDebugLevel(t *testing.T) {
	prevLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(prevLevel)

	ctx := Init(context.Background(), true)
	if ctx == nil {
		t.Fatal("Init returned nil context")
	}
	if got := zerolog.GlobalLevel(); got != zerolog.DebugLevel {
		t.Fatalf("zerolog global level = %s, want %s", got, zerolog.DebugLevel)
	}
	if !DebugEnabled() {
		t.Fatal("DebugEnabled() = false, want true")
	}
}

func TestInitWritesTextLogsByDefault(t *testing.T) {
	prevLevel := zerolog.GlobalLevel()
	prevLogger := log.Logger
	defer func() {
		zerolog.SetGlobalLevel(prevLevel)
		log.Logger = prevLogger
	}()

	var out bytes.Buffer
	ctx := initWithWriter(context.Background(), false, &out)
	log.Ctx(ctx).Info().Str("component", "review").Msg("started")

	text := out.String()
	if strings.Contains(text, `{"level"`) {
		t.Fatalf("log output is JSON, want text:\n%s", text)
	}
	for _, needle := range []string{"INF", "started", "component=review"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("log output missing %q:\n%s", needle, text)
		}
	}
}

func TestDebugLogsRenderProviderResponseBlock(t *testing.T) {
	prevLevel := zerolog.GlobalLevel()
	prevLogger := log.Logger
	defer func() {
		zerolog.SetGlobalLevel(prevLevel)
		log.Logger = prevLogger
	}()

	var out bytes.Buffer
	ctx := initWithWriter(context.Background(), true, &out)
	response := "not json\nsecond line"
	log.Ctx(ctx).Debug().Str(validationJSONField, response).Msg("output schema validation failed")

	text := out.String()
	for _, needle := range []string{
		"DBG",
		"output schema validation failed",
		"provider response:\nnot json\nsecond line",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("debug output missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, validationJSONField+"=") {
		t.Fatalf("debug output should render provider response as a block, not key/value:\n%s", text)
	}
}

func TestProviderResponseBlockIsDebugOnly(t *testing.T) {
	prevLevel := zerolog.GlobalLevel()
	prevLogger := log.Logger
	defer func() {
		zerolog.SetGlobalLevel(prevLevel)
		log.Logger = prevLogger
	}()

	var out bytes.Buffer
	ctx := initWithWriter(context.Background(), false, &out)
	log.Ctx(ctx).Info().Str(validationJSONField, "not json").Msg("info event")

	text := out.String()
	if strings.Contains(text, "provider response:") || strings.Contains(text, "not json") {
		t.Fatalf("info output should not include provider response:\n%s", text)
	}
}
