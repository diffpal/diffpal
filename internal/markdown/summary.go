package markdown

import (
	"fmt"
	"sort"
	"strconv"
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
	sortedFindings := sortFindings(bundle.Findings)
	rows := feedbackRows(bundle, sortedFindings)
	blocking := countBlocking(sortedFindings)

	out := strings.Builder{}
	out.WriteString("# DiffPal Review Summary\n\n")
	fmt.Fprintf(&out, "review_id: %s\n", bundle.ReviewID)
	fmt.Fprintf(&out, "base: %s\nhead: %s\n\n", bundle.BaseSHA, bundle.HeadSHA)
	out.WriteString("## Summary of Changes\n\n")
	fmt.Fprintf(&out, "- Reviewed files: %d\n", len(rows))
	fmt.Fprintf(&out, "- Findings: %d\n", len(sortedFindings))
	fmt.Fprintf(&out, "- Blocking findings: %d\n", blocking)
	if bundle.Language != "" {
		fmt.Fprintf(&out, "- Language: %s\n", bundle.Language)
	}
	if len(bundle.ReviewChecks) > 0 {
		fmt.Fprintf(&out, "- Review checks: %s\n", strings.Join(bundle.ReviewChecks, ", "))
	}
	if len(sortedFindings) == 0 {
		out.WriteString("\nDiffPal found no actionable issues in the reviewed diff.\n\n")
	} else {
		out.WriteString("\nDiffPal found actionable feedback in the reviewed diff.\n\n")
	}

	out.WriteString("## Feedback on Files\n\n")
	if len(rows) == 0 {
		out.WriteString("No reviewable files were recorded.\n")
		return out.String()
	}
	out.WriteString("| File | Status | Notes |\n")
	out.WriteString("| --- | --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(&out, "| `%s` | %s | %s |\n", escapeTable(row.Path), row.Status, escapeTable(row.Notes))
	}
	out.WriteString("\n")

	if len(sortedFindings) == 0 {
		return out.String()
	}

	out.WriteString("## Detailed Comments\n\n")
	byPath := groupByPath(sortedFindings)
	for _, path := range sortedKeys(byPath) {
		fmt.Fprintf(&out, "### %s\n\n", path)
		for _, finding := range byPath[path] {
			fmt.Fprintf(&out, "- **[%s][%s]**", strings.ToLower(finding.Severity), finding.RuleID)
			if finding.StartLine > 0 {
				fmt.Fprintf(&out, " `%s`", lineRange(finding.StartLine, finding.EndLine))
			}
			fmt.Fprintf(&out, ": %s\n", firstNonEmpty(finding.Message, finding.Title))
			if finding.Evidence != "" {
				fmt.Fprintf(&out, "  - Evidence: %s\n", finding.Evidence)
			}
			if finding.Suggestion != "" {
				fmt.Fprintf(&out, "  - Suggestion: %s\n", finding.Suggestion)
			}
			fmt.Fprintf(&out, "  - Confidence: %.2f\n", finding.Confidence)
		}
		out.WriteString("\n")
	}
	return out.String()
}

type feedbackRow struct {
	Path   string
	Status string
	Notes  string
}

func feedbackRows(bundle findings.FindingsBundle, items []findings.Finding) []feedbackRow {
	byPath := groupByPath(items)
	paths := reviewedPaths(bundle, byPath)
	rows := make([]feedbackRow, 0, len(paths))
	for _, path := range paths {
		findingsForFile := byPath[path]
		status := "Passed"
		notes := "No actionable findings."
		if len(findingsForFile) > 0 {
			if countBlocking(findingsForFile) > 0 {
				status = "Blocked"
			} else {
				status = "Needs attention"
			}
			notes = severityNotes(findingsForFile)
		}
		rows = append(rows, feedbackRow{Path: path, Status: status, Notes: notes})
	}
	return rows
}

func reviewedPaths(bundle findings.FindingsBundle, byPath map[string][]findings.Finding) []string {
	seen := map[string]struct{}{}
	paths := make([]string, 0, len(bundle.Files)+len(byPath))
	for _, file := range bundle.Files {
		path := strings.TrimSpace(file.Path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	for path := range byPath {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func sortFindings(items []findings.Finding) []findings.Finding {
	out := append([]findings.Finding(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		left, right := out[i], out[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.StartLine != right.StartLine {
			return left.StartLine < right.StartLine
		}
		if left.EndLine != right.EndLine {
			return left.EndLine < right.EndLine
		}
		leftSeverity, leftKnown := severityOrder[strings.ToLower(left.Severity)]
		rightSeverity, rightKnown := severityOrder[strings.ToLower(right.Severity)]
		if leftKnown && rightKnown && leftSeverity != rightSeverity {
			return leftSeverity < rightSeverity
		}
		if left.RuleID != right.RuleID {
			return left.RuleID < right.RuleID
		}
		return left.Message < right.Message
	})
	return out
}

func severityNotes(items []findings.Finding) string {
	counts := map[string]int{}
	for _, item := range items {
		counts[strings.ToLower(strings.TrimSpace(item.Severity))]++
	}
	parts := make([]string, 0, len(counts))
	for _, severity := range []string{"critical", "high", "medium", "low"} {
		count := counts[severity]
		if count == 0 {
			continue
		}
		parts = append(parts, severity+": "+strconv.Itoa(count))
	}
	if len(parts) == 0 {
		return strconv.Itoa(len(items)) + " finding(s)."
	}
	return strings.Join(parts, ", ")
}

func countBlocking(items []findings.Finding) int {
	count := 0
	for _, item := range items {
		if item.Blocking {
			count++
		}
	}
	return count
}

func lineRange(start, end int) string {
	if end <= 0 || end == start {
		return fmt.Sprintf("L%d", start)
	}
	return fmt.Sprintf("L%d-L%d", start, end)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func escapeTable(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}

func groupByPath(items []findings.Finding) map[string][]findings.Finding {
	out := make(map[string][]findings.Finding, len(items))
	for _, item := range items {
		out[item.Path] = append(out[item.Path], item)
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
