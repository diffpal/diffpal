package github

import (
	"fmt"
	"strings"
)

const DefaultReviewChannel = "diffpal"

type ReviewIdentity struct {
	Channel string
}

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

func (identity ReviewIdentity) channel() string {
	if identity.Channel == "" {
		return DefaultReviewChannel
	}
	return identity.Channel
}

func (identity ReviewIdentity) CheckRunName() string {
	return identity.channel() + "-checks"
}

func (identity ReviewIdentity) SummaryMarker() string {
	channel := identity.channel()
	if channel == DefaultReviewChannel {
		return "<!-- diffpal:summary -->"
	}
	return "<!-- diffpal:summary:" + channel + " -->"
}

func (identity ReviewIdentity) SummaryTitle() string {
	channel := identity.channel()
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
