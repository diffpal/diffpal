package github

import (
	"fmt"
	"strings"
)

// DefaultReviewChannel is the stable GitHub publishing channel used by DiffPal.
const DefaultReviewChannel = "diffpal"

// ReviewIdentity identifies one GitHub publishing channel.
type ReviewIdentity struct {
	// Channel is the normalized publishing channel name.
	Channel string
}

// NewReviewIdentity returns a normalized GitHub review identity.
func NewReviewIdentity(channel string) (ReviewIdentity, error) {
	normalized := normalizeReviewChannel(channel)
	if normalized == "" {
		normalized = DefaultReviewChannel
	}
	if !validReviewChannel(normalized) {
		return ReviewIdentity{}, fmt.Errorf("invalid review channel %q", channel)
	}
	return ReviewIdentity{Channel: normalized}, nil
}

func (id ReviewIdentity) channel() string {
	if id.Channel == "" {
		return DefaultReviewChannel
	}
	return id.Channel
}

// CheckRunName returns the GitHub check run name for the review channel.
func (id ReviewIdentity) CheckRunName() string {
	return id.channel() + "-checks"
}

// ReviewMarker returns the hidden marker used to reconcile one PR review per
// publishing channel and head commit.
func (id ReviewIdentity) ReviewMarker(headSHA string) string {
	channel := id.channel()
	cleanHead := strings.TrimSpace(headSHA)
	if cleanHead == "" {
		return "<!-- diffpal:review:" + channel + " -->"
	}
	return "<!-- diffpal:review:" + channel + " head_sha:" + cleanHead + " -->"
}

// SummaryTitle returns the Markdown title for the review channel summary.
func (id ReviewIdentity) SummaryTitle() string {
	channel := id.channel()
	if channel == DefaultReviewChannel {
		return "DiffPal Review Summary"
	}
	label := strings.TrimPrefix(channel, DefaultReviewChannel+"-")
	parts := strings.FieldsFunc(label, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		clean = append(clean, strings.ToUpper(part[:1])+part[1:])
	}
	if len(clean) == 0 {
		return "DiffPal Review Summary"
	}
	return "DiffPal " + strings.Join(clean, " ") + " Review Summary"
}

func normalizeReviewChannel(channel string) string {
	return strings.ToLower(strings.TrimSpace(channel))
}

func validReviewChannel(channel string) bool {
	if channel == "" || len(channel) > 64 {
		return false
	}
	for i, r := range channel {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case i > 0 && (r == '-' || r == '_' || r == '.'):
		default:
			return false
		}
	}
	return true
}
