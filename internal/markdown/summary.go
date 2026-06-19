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

type SummaryOptions struct {
	Title           string
	FeedbackProfile string
	PublishSurfaces []string
	ShowMetadata    bool
	HideOverview    bool
	HideDetails     bool
	Snippets        SnippetProvider
	Links           FindingLinkProvider
}

func RenderSummary(bundle findings.FindingsBundle) string {
	return RenderSummaryWithOptions(bundle, SummaryOptions{})
}

func RenderSummaryWithOptions(bundle findings.FindingsBundle, opts SummaryOptions) string {
	sortedFindings := sortFindings(bundle.Findings)
	blocking := countBlocking(sortedFindings)

	out := strings.Builder{}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "DiffPal Review Summary"
	}
	fmt.Fprintf(&out, "# %s\n\n", title)
	if !opts.HideOverview {
		writeChangeOverview(&out, bundle)
	}

	out.WriteString("## Review Result\n\n")
	if len(sortedFindings) == 0 {
		out.WriteString("DiffPal found no actionable issues in the reviewed diff.\n\n")
	} else {
		fmt.Fprintf(&out, "DiffPal found %d actionable finding(s)", len(sortedFindings))
		if blocking > 0 {
			fmt.Fprintf(&out, ", including %d blocking finding(s)", blocking)
		}
		out.WriteString(".\n\n")
	}

	if opts.ShowMetadata {
		writeMetadata(&out, bundle, sortedFindings, blocking, opts)
	}

	if len(sortedFindings) == 0 {
		return out.String()
	}
	if opts.HideDetails {
		return out.String()
	}

	out.WriteString("## Detailed Comments\n\n")
	byPath := groupByPath(sortedFindings)
	for _, path := range sortedKeys(byPath) {
		fmt.Fprintf(&out, "### %s\n\n", path)
		for _, finding := range byPath[path] {
			out.WriteString(renderSummaryFindingDetail(finding, summaryFindingDetailOptions{
				Snippet: snippetForFinding(opts.Snippets, finding),
				Link:    linkForFinding(opts.Links, finding),
			}))
		}
		out.WriteString("\n")
	}
	return out.String()
}

func writeChangeOverview(out *strings.Builder, bundle findings.FindingsBundle) {
	out.WriteString("## Summary of Changes\n\n")
	items := changeSummaryItems(bundle)
	if len(items) == 0 {
		out.WriteString("DiffPal could not generate a semantic change overview from the reviewed diff.\n\n")
		return
	}
	for _, item := range items {
		fmt.Fprintf(out, "- %s\n", item)
	}
	out.WriteString("\n")
}

func writeMetadata(out *strings.Builder, bundle findings.FindingsBundle, sortedFindings []findings.Finding, blocking int, opts SummaryOptions) {
	out.WriteString("## Review Metadata\n\n")
	fmt.Fprintf(out, "- Review ID: %s\n", bundle.ReviewID)
	fmt.Fprintf(out, "- Base: %s\n", bundle.BaseSHA)
	fmt.Fprintf(out, "- Head: %s\n", bundle.HeadSHA)
	fmt.Fprintf(out, "- Reviewed files: %d\n", reviewedFileCount(bundle, sortedFindings))
	fmt.Fprintf(out, "- Findings: %d\n", len(sortedFindings))
	fmt.Fprintf(out, "- Blocking findings: %d\n", blocking)
	if strings.TrimSpace(opts.FeedbackProfile) != "" {
		fmt.Fprintf(out, "- Feedback profile: %s\n", strings.TrimSpace(opts.FeedbackProfile))
	}
	if len(opts.PublishSurfaces) > 0 {
		fmt.Fprintf(out, "- Publish surfaces: %s\n", strings.Join(opts.PublishSurfaces, ", "))
	}
	if bundle.Language != "" {
		fmt.Fprintf(out, "- Language: %s\n", bundle.Language)
	}
	out.WriteString("\n")
}

func changeSummaryItems(bundle findings.FindingsBundle) []string {
	out := make([]string, 0, len(bundle.ChangeSummary))
	for _, item := range bundle.ChangeSummary {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	if len(out) > 0 {
		return out
	}
	return nil
}

func reviewedFileCount(bundle findings.FindingsBundle, items []findings.Finding) int {
	seen := map[string]struct{}{}
	for _, file := range bundle.Files {
		path := strings.TrimSpace(file.Path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
	}
	for _, finding := range items {
		path := strings.TrimSpace(finding.Path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
	}
	return len(seen)
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
		if left.Category != right.Category {
			return left.Category < right.Category
		}
		return left.Message < right.Message
	})
	return out
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

type FindingDetailOptions struct {
	ListItem bool
	Snippet  CodeSnippet
	Link     string
}

type summaryFindingDetailOptions struct {
	Snippet CodeSnippet
	Link    string
}

func renderSummaryFindingDetail(finding findings.Finding, opts summaryFindingDetailOptions) string {
	out := strings.Builder{}
	fmt.Fprintf(&out, "#### %s", findingHeading(finding, false))
	if finding.StartLine > 0 {
		fmt.Fprintf(&out, " - %s", lineRange(finding.StartLine, finding.EndLine))
	}
	out.WriteString("\n\n")
	fmt.Fprintf(&out, "%s\n\n", firstNonEmpty(finding.Message, finding.Title))
	if impact := finding.ImpactText(); impact != "" {
		fmt.Fprintf(&out, "**Impact:** %s\n\n", impact)
	}
	if finding.Suggestion != "" {
		fmt.Fprintf(&out, "**Fix:** %s\n\n", finding.Suggestion)
	}
	if evidence := finding.EvidenceText(); evidence != "" {
		fmt.Fprintf(&out, "**Evidence:** %s\n\n", evidence)
	}
	if link := strings.TrimSpace(opts.Link); link != "" {
		fmt.Fprintf(&out, "**Source:** [View changed lines](%s)\n\n", link)
	}
	if opts.Snippet.Code != "" {
		writeCodeFence(&out, opts.Snippet, "")
		out.WriteString("\n")
	}
	return out.String()
}

func RenderFindingDetail(finding findings.Finding, opts FindingDetailOptions) string {
	out := strings.Builder{}
	prefix := ""
	detailPrefix := "- "
	if opts.ListItem {
		prefix = "- "
		detailPrefix = "  - "
	}
	link := strings.TrimSpace(opts.Link)
	hasLink := link != ""
	fmt.Fprintf(&out, "%s**%s**", prefix, findingHeading(finding, hasLink))
	if finding.StartLine > 0 && !hasLink {
		fmt.Fprintf(&out, " `%s`", lineRange(finding.StartLine, finding.EndLine))
	}
	fmt.Fprintf(&out, ": %s\n", firstNonEmpty(finding.Message, finding.Title))
	if evidence := finding.EvidenceText(); evidence != "" {
		fmt.Fprintf(&out, "%s**Evidence**: %s\n", detailPrefix, evidence)
	}
	if impact := finding.ImpactText(); impact != "" {
		fmt.Fprintf(&out, "%s**Impact**: %s\n", detailPrefix, impact)
	}
	if hasLink {
		fmt.Fprintf(&out, "%s**Source**:\n%s%s\n", detailPrefix, detailPrefix, link)
	}
	if opts.Snippet.Code != "" {
		out.WriteString("\n")
		indent := ""
		if opts.ListItem {
			indent = "  "
		}
		writeCodeFence(&out, opts.Snippet, indent)
		out.WriteString("\n")
	}
	if finding.Suggestion != "" {
		fmt.Fprintf(&out, "%s**Suggestion**: %s\n", detailPrefix, finding.Suggestion)
	}
	return out.String()
}

func findingHeading(finding findings.Finding, linked bool) string {
	severity := strings.ToLower(strings.TrimSpace(finding.Severity))
	category := strings.ToLower(strings.TrimSpace(finding.Category))
	return titleWord(severity) + " " + category
}

func titleWord(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func snippetForFinding(provider SnippetProvider, finding findings.Finding) CodeSnippet {
	if provider == nil {
		return CodeSnippet{}
	}
	snippet, ok := provider.Snippet(finding)
	if !ok {
		return CodeSnippet{}
	}
	return snippet
}

func linkForFinding(provider FindingLinkProvider, finding findings.Finding) string {
	if provider == nil {
		return ""
	}
	link, ok := provider.Link(finding)
	if !ok {
		return ""
	}
	return link
}

func writeCodeFence(out *strings.Builder, snippet CodeSnippet, indent string) {
	fence := codeFence(snippet.Code)
	fmt.Fprintf(out, "%s%s%s\n", indent, fence, snippet.Language)
	writeIndentedCode(out, snippet.Code, indent)
	fmt.Fprintf(out, "%s%s\n", indent, fence)
}

func writeIndentedCode(out *strings.Builder, code string, indent string) {
	for _, line := range strings.SplitAfter(code, "\n") {
		if line == "" {
			continue
		}
		out.WriteString(indent)
		out.WriteString(line)
	}
	if !strings.HasSuffix(code, "\n") {
		out.WriteString("\n")
	}
}

func codeFence(code string) string {
	longest := 0
	current := 0
	for _, r := range code {
		if r == '`' {
			current++
			if current > longest {
				longest = current
			}
			continue
		}
		current = 0
	}
	if longest < 3 {
		return "```"
	}
	return strings.Repeat("`", longest+1)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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
