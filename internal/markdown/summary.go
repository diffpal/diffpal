package markdown

import (
	"fmt"
	"sort"
	"strings"

	"github.com/diffpal/diffpal/internal/findings"
)

var severityOrder = map[string]int{
	"critical": 0,
	"high":     1,
	"medium":   2,
	"low":      3,
}

func RenderSummary(bundle findings.FindingsBundle) string {
	bySeverity := map[string][]findings.Finding{}
	for _, f := range bundle.Findings {
		bySeverity[f.Severity] = append(bySeverity[f.Severity], f)
	}

	severities := make([]string, 0, len(bySeverity))
	for severity := range bySeverity {
		severities = append(severities, severity)
	}
	sort.Slice(severities, func(i, j int) bool {
		left, lok := severityOrder[severities[i]]
		right, rok := severityOrder[severities[j]]
		switch {
		case lok && rok:
			if left == right {
				return severities[i] < severities[j]
			}
			return left < right
		case lok:
			return true
		case rok:
			return false
		default:
			return severities[i] < severities[j]
		}
	})

	out := strings.Builder{}
	out.WriteString("# DiffPal Findings Summary\n\n")
	fmt.Fprintf(&out, "review_id: %s\n", bundle.ReviewID)
	fmt.Fprintf(&out, "base: %s\nhead: %s\n\n", bundle.BaseSHA, bundle.HeadSHA)
	if len(severities) == 0 {
		out.WriteString("No findings.\n")
		return out.String()
	}

	for _, severity := range severities {
		findingsAtSeverity := bySeverity[severity]
		files := groupByPath(findingsAtSeverity)
		fileNames := sortedKeys(files)
		fmt.Fprintf(&out, "## %s (%d)\n", strings.ToUpper(severity), len(findingsAtSeverity))
		for _, fileName := range fileNames {
			fmt.Fprintf(&out, "### %s\n", fileName)
			rules := groupByRule(files[fileName])
			ruleIDs := sortedKeys(rules)
			for _, ruleID := range ruleIDs {
				fmt.Fprintf(&out, "- `%s` (%d)\n", ruleID, len(rules[ruleID]))
				for _, finding := range rules[ruleID] {
					fmt.Fprintf(&out, "  - [%d-%d] %s\n", finding.StartLine, finding.EndLine, finding.Message)
				}
			}
		}
		out.WriteString("\n")
	}
	return out.String()
}

func groupByPath(items []findings.Finding) map[string][]findings.Finding {
	out := make(map[string][]findings.Finding, len(items))
	for _, item := range items {
		out[item.Path] = append(out[item.Path], item)
	}
	return out
}

func groupByRule(items []findings.Finding) map[string][]findings.Finding {
	out := make(map[string][]findings.Finding, len(items))
	for _, item := range items {
		out[item.RuleID] = append(out[item.RuleID], item)
	}
	return out
}

func sortedKeys[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
