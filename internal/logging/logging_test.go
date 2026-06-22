package logging

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
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
