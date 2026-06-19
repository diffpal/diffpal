package github

import "testing"

func TestReviewIdentityDefaultsToStableChannel(t *testing.T) {
	t.Parallel()

	identity, err := NewReviewIdentity("")
	if err != nil {
		t.Fatalf("NewReviewIdentity() error = %v", err)
	}
	if identity.ReviewMarker("head-a") != "<!-- diffpal:review:diffpal head_sha:head-a -->" {
		t.Fatalf("ReviewMarker() = %q, want default marker", identity.ReviewMarker("head-a"))
	}
	if identity.SummaryTitle() != "DiffPal Review Summary" {
		t.Fatalf("SummaryTitle() = %q, want default title", identity.SummaryTitle())
	}
}

func TestReviewIdentityIsolatesDevChannel(t *testing.T) {
	t.Parallel()

	identity, err := NewReviewIdentity("DiffPal-Dev")
	if err != nil {
		t.Fatalf("NewReviewIdentity() error = %v", err)
	}
	if identity.Channel != "diffpal-dev" {
		t.Fatalf("Channel = %q, want diffpal-dev", identity.Channel)
	}
	if identity.ReviewMarker("head-a") != "<!-- diffpal:review:diffpal-dev head_sha:head-a -->" {
		t.Fatalf("ReviewMarker() = %q, want dev marker", identity.ReviewMarker("head-a"))
	}
	if identity.SummaryTitle() != "DiffPal Dev Review Summary" {
		t.Fatalf("SummaryTitle() = %q, want dev title", identity.SummaryTitle())
	}
}

func TestReviewIdentityRejectsUnsafeChannels(t *testing.T) {
	t.Parallel()

	for _, channel := range []string{"bad channel", "bad/channel", "-bad", "bad\nchannel"} {
		if _, err := NewReviewIdentity(channel); err == nil {
			t.Fatalf("NewReviewIdentity(%q) error = nil, want error", channel)
		}
	}
}
