package policy

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Finding struct {
	RuleID     string
	Severity   Severity
	Confidence float64
	Path       string
}

type FindingDecision struct {
	Path       string
	RuleID     string
	Action     string
	Reason     string
	Severity   Severity
	Confidence float64
}

type Suppression struct {
	Path string
	Rule string
}

type Policy struct {
	BlockOn Severity
	WarnOn  []Severity
	Exclude []Suppression
}

var severityRank = map[Severity]int{
	SeverityLow:      1,
	SeverityMedium:   2,
	SeverityHigh:     3,
	SeverityCritical: 4,
}

func ApplyPolicy(policy Policy, findings []Finding) []FindingDecision {
	out := make([]FindingDecision, 0, len(findings))
	for _, item := range findings {
		dec := FindingDecision{
			Path:       item.Path,
			RuleID:     item.RuleID,
			Severity:   item.Severity,
			Confidence: item.Confidence,
			Action:     "report",
		}
		if isSuppressed(item, policy.Exclude) {
			dec.Action = "suppress"
			dec.Reason = "suppressed by policy"
			out = append(out, dec)
			continue
		}
		if isBlocking(policy.BlockOn, item.Severity) {
			dec.Action = "block"
			dec.Reason = "meets block threshold"
			out = append(out, dec)
			continue
		}
		if isWarn(policy.WarnOn, item.Severity) {
			dec.Action = "warn"
			dec.Reason = "warn threshold match"
		}
		out = append(out, dec)
	}
	return out
}

func isSuppressed(f Finding, suppressions []Suppression) bool {
	for _, suppression := range suppressions {
		if suppression.Rule != "" && suppression.Rule != f.RuleID {
			continue
		}
		if suppression.Path != "" {
			if matchPathGlob(suppression.Path, f.Path) {
				return true
			}
		}
	}
	return false
}

func isBlocking(threshold Severity, level Severity) bool {
	limit, ok := severityRank[threshold]
	if !ok {
		return false
	}
	score, ok := severityRank[level]
	if !ok {
		return false
	}
	return score >= limit
}

func isWarn(warnOn []Severity, level Severity) bool {
	for _, severity := range warnOn {
		if severity == level {
			return true
		}
	}
	return false
}

func ParseSeverity(v string) (Severity, error) {
	s := Severity(v)
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return s, nil
	default:
		return "", fmt.Errorf("invalid severity: %s", v)
	}
}

func matchPathGlob(pattern string, target string) bool {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	target = filepath.ToSlash(strings.TrimSpace(target))
	if pattern == "" {
		return false
	}
	return matchSegments(splitSegments(pattern), splitSegments(target))
}

func splitSegments(v string) []string {
	v = strings.Trim(v, "/")
	if v == "" {
		return nil
	}
	return strings.Split(v, "/")
}

func matchSegments(pattern []string, target []string) bool {
	if len(pattern) == 0 {
		return len(target) == 0
	}
	if pattern[0] == "**" {
		if len(pattern) == 1 {
			return true
		}
		for i := 0; i <= len(target); i++ {
			if matchSegments(pattern[1:], target[i:]) {
				return true
			}
		}
		return false
	}
	if len(target) == 0 {
		return false
	}
	ok, err := filepath.Match(pattern[0], target[0])
	if err != nil || !ok {
		return false
	}
	return matchSegments(pattern[1:], target[1:])
}
