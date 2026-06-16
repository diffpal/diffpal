package reviewer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	dpconfig "github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/diff"
	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/policy"
	"github.com/diffpal/diffpal/internal/reliability"
	"github.com/diffpal/diffpal/internal/reviewer/promptpack"
	"github.com/normahq/norma/pkg/runtime/agentconfig"
)

type Options struct {
	WorkingDir       string
	Repo             string
	ReviewID         string
	BaseSHA          string
	HeadSHA          string
	MaxFiles         int
	ContextLines     int
	MaxPatchChars    int
	MaxFilesPerChunk int
	BlockOn          string
	Language         string
	ReviewChecks     []string
	Instructions     string
}

type Result struct {
	Bundle          findings.FindingsBundle
	Files           []diff.FileChange
	ChangedFiles    int
	ReviewableFiles int
	ContextFiles    int
	ContextChunks   int
	TestSummary     string
}

type ChunkSpan struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type ChunkFile struct {
	Path         string      `json:"path"`
	Status       string      `json:"status"`
	PreviousPath string      `json:"previous_path,omitempty"`
	Spans        []ChunkSpan `json:"spans"`
}

type ChunkInput struct {
	ReviewID              string      `json:"review_id"`
	Repo                  string      `json:"repo"`
	BaseSHA               string      `json:"base_sha"`
	HeadSHA               string      `json:"head_sha"`
	ChunkIndex            int         `json:"chunk_index"`
	ChunkCount            int         `json:"chunk_count"`
	ReviewTask            string      `json:"review_task"`
	UntrustedInputWarning string      `json:"untrusted_input_warning"`
	UntrustedInputStart   string      `json:"untrusted_input_start"`
	UntrustedInputEnd     string      `json:"untrusted_input_end"`
	Language              string      `json:"language"`
	ReviewChecks          []string    `json:"review_checks"`
	Instructions          string      `json:"instructions,omitempty"`
	TestSummary           string      `json:"test_summary"`
	CommitMessages        []string    `json:"commit_messages,omitempty"`
	Files                 []ChunkFile `json:"files"`
}

type ChunkFinding struct {
	Category       string                   `json:"category"`
	Severity       string                   `json:"severity"`
	Confidence     float64                  `json:"confidence"`
	Path           string                   `json:"path"`
	StartLine      int                      `json:"start_line"`
	EndLine        int                      `json:"end_line"`
	ChangedSpan    findings.LineSpan        `json:"changed_span"`
	SupportingSpan *findings.LineSpan       `json:"supporting_span,omitempty"`
	Title          string                   `json:"title"`
	Message        string                   `json:"message"`
	Evidence       findings.FindingEvidence `json:"evidence"`
	Impact         findings.FindingImpact   `json:"impact"`
	Suggestion     string                   `json:"suggestion,omitempty"`
}

type ChunkOutput struct {
	ChangeSummary []string       `json:"change_summary"`
	Findings      []ChunkFinding `json:"findings"`
}

type RuntimeUsage struct {
	TokenUsage int64
	Inspection *findings.Inspection
}

type RuntimeConfig struct {
	ProviderID   string
	Providers    map[string]dpconfig.ProviderConfig
	MCPServers   map[string]agentconfig.MCPServerConfig
	WorkingDir   string
	Instructions string
	BaseSHA      string
	HeadSHA      string
	ChangedFiles []ChunkFile
}

type Runtime interface {
	ReviewChunk(ctx context.Context, cfg RuntimeConfig, input ChunkInput) (ChunkOutput, RuntimeUsage, error)
}

func Run(ctx context.Context, cfg dpconfig.Config, opts Options) (Result, error) {
	return RunWithRuntime(ctx, cfg, opts, ADKRuntime{})
}

func RunWithRuntime(ctx context.Context, cfg dpconfig.Config, opts Options, runtime Runtime) (Result, error) {
	if runtime == nil {
		runtime = ADKRuntime{}
	}
	if err := cfg.Validate(); err != nil {
		return Result{}, wrapError(KindConfig, err)
	}
	workingDir := strings.TrimSpace(opts.WorkingDir)
	if workingDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return Result{}, wrapError(KindInternal, fmt.Errorf("get working directory: %w", err))
		}
		workingDir = cwd
	}
	result, err := diff.Collect(diff.Options{
		BaseSHA:  opts.BaseSHA,
		HeadSHA:  opts.HeadSHA,
		MaxFiles: opts.MaxFiles,
		WorkDir:  workingDir,
	})
	if err != nil {
		return Result{}, wrapError(KindInternal, err)
	}
	reviewID := strings.TrimSpace(opts.ReviewID)
	if reviewID == "" {
		reviewID = "review"
	}
	repo := strings.TrimSpace(opts.Repo)
	if repo == "" {
		repo = "local"
	}
	blockOn := strings.TrimSpace(opts.BlockOn)
	if blockOn == "" {
		blockOn = cfg.BlockOn()
	}
	if blockOn == "" {
		blockOn = "high"
	}
	language := strings.TrimSpace(opts.Language)
	if language == "" {
		language = cfg.ReviewLanguage()
	}
	language, err = dpconfig.NormalizeReviewLanguage(language)
	if err != nil {
		return Result{}, wrapError(KindConfig, err)
	}
	reviewChecks := opts.ReviewChecks
	if len(reviewChecks) == 0 {
		reviewChecks = cfg.ReviewChecks()
	}
	reviewChecks, err = dpconfig.NormalizeReviewChecks(reviewChecks)
	if err != nil {
		return Result{}, wrapError(KindConfig, err)
	}
	instructions := strings.TrimSpace(opts.Instructions)
	if instructions == "" {
		instructions = cfg.ReviewInstructions()
	}
	allowedCategories := categoriesForReviewChecks(reviewChecks)
	filtered := filterReviewableFiles(result.Files)
	testSummary := testSummaryFromFiles(filtered)
	commitMessages, err := collectCommitMessages(workingDir, result.BaseSHA, result.HeadSHA)
	if err != nil {
		return Result{}, wrapError(KindInternal, err)
	}
	chunks := chunkFileChanges(filtered, opts.MaxFilesPerChunk)
	reviewed := reviewedFiles(filtered)
	bundle := findings.FindingsBundle{
		Version:       findings.VersionV2,
		ReviewID:      reviewID,
		BaseSHA:       result.BaseSHA,
		HeadSHA:       result.HeadSHA,
		Language:      language,
		ReviewChecks:  append([]string(nil), reviewChecks...),
		Prompt:        promptpack.ReviewMetadata(),
		ChangeSummary: findings.SemanticChangeSummary(reviewed),
		Files:         reviewed,
		Findings:      []findings.Finding{},
	}
	if len(chunks) == 0 {
		return Result{
			Bundle:          bundle,
			Files:           append([]diff.FileChange(nil), result.Files...),
			ChangedFiles:    result.ChangedFiles,
			ReviewableFiles: len(filtered),
			ContextFiles:    len(filtered),
			ContextChunks:   0,
			TestSummary:     testSummary,
		}, nil
	}

	runtimeCfg := RuntimeConfig{
		ProviderID:   cfg.ProviderID(),
		Providers:    providersWithEnv(cfg.Providers),
		MCPServers:   cfg.MCPServers,
		WorkingDir:   workingDir,
		Instructions: instructions,
		BaseSHA:      result.BaseSHA,
		HeadSHA:      result.HeadSHA,
		ChangedFiles: chunkFilesFromChanges(filtered),
	}

	collected := make([]findings.Finding, 0)
	summaries := make([]string, 0, len(chunks))
	inspection := mergeInspection(nil, inspectionForProvider(runtimeCfg.ProviderID, runtimeCfg.Providers))
	for i, chunk := range chunks {
		input := chunkInputFromChanges(reviewID, repo, result.BaseSHA, result.HeadSHA, language, reviewChecks, instructions, testSummary, commitMessages, i, len(chunks), chunk)
		var output ChunkOutput
		var usage RuntimeUsage
		err := reliability.RetryWithPolicy(ctx, reliability.Policy{
			Attempts:  3,
			BaseDelay: 750 * time.Millisecond,
			Timeout:   90 * time.Second,
			IsTransient: func(err error) bool {
				var reviewErr *Error
				if errors.As(err, &reviewErr) {
					return reviewErr.Kind == KindTransient
				}
				return reliability.IsTransient(err)
			},
		}, func(runCtx context.Context) error {
			chunkOutput, chunkUsage, err := runtime.ReviewChunk(runCtx, runtimeCfg, input)
			if err != nil {
				return err
			}
			output = chunkOutput
			usage = chunkUsage
			return nil
		})
		if err != nil {
			if isStructuredOutputProviderError(err) {
				continue
			}
			return Result{}, err
		}
		chunkInspection := mergeInspection(inspectionForProvider(runtimeCfg.ProviderID, runtimeCfg.Providers), usage.Inspection)
		if err := validateInspection(chunkInspection); err != nil {
			return Result{}, wrapError(KindInternal, err)
		}
		inspection = mergeInspection(inspection, usage.Inspection)
		summaries = append(summaries, output.ChangeSummary...)
		collected = append(collected, validateChunkFindings(output.Findings, input.Files, cfg.ProviderID(), allowedCategories)...)
	}

	bundle.ChangeSummary = normalizeChangeSummary(summaries)
	if len(bundle.ChangeSummary) == 0 {
		bundle.ChangeSummary = findings.SemanticChangeSummary(bundle.Files)
	}
	bundle.Findings = dedupeAndSortFindings(collected, repo, reviewID, result.HeadSHA)
	bundle.Inspection = inspection
	if err := applyBlockingPolicy(&bundle, blockOn); err != nil {
		return Result{}, wrapError(KindConfig, err)
	}

	return Result{
		Bundle:          bundle,
		Files:           append([]diff.FileChange(nil), result.Files...),
		ChangedFiles:    result.ChangedFiles,
		ReviewableFiles: len(filtered),
		ContextFiles:    len(filtered),
		ContextChunks:   len(chunks),
		TestSummary:     testSummary,
	}, nil
}

func inspectionForProvider(providerID string, providers map[string]dpconfig.ProviderConfig) *findings.Inspection {
	providerCfg, ok := providers[providerID]
	if !ok {
		return nil
	}
	providerType := strings.ToLower(strings.TrimSpace(providerCfg.Type))
	required := providerType == "openai" || providerType == "aistudio"
	return &findings.Inspection{
		ProviderType: providerType,
		Required:     required,
	}
}

func mergeInspection(base *findings.Inspection, next *findings.Inspection) *findings.Inspection {
	if base == nil {
		if next == nil {
			return nil
		}
		copied := *next
		copied.ToolCalls = append([]string(nil), next.ToolCalls...)
		return &copied
	}
	if next == nil {
		return base
	}
	if base.ProviderType == "" {
		base.ProviderType = next.ProviderType
	}
	base.Required = base.Required || next.Required
	base.DiffInspected = base.DiffInspected || next.DiffInspected
	base.ContextInspected = base.ContextInspected || next.ContextInspected
	seen := make(map[string]struct{}, len(base.ToolCalls)+len(next.ToolCalls))
	for _, call := range base.ToolCalls {
		seen[call] = struct{}{}
	}
	for _, call := range next.ToolCalls {
		if _, ok := seen[call]; ok {
			continue
		}
		seen[call] = struct{}{}
		base.ToolCalls = append(base.ToolCalls, call)
	}
	sort.Strings(base.ToolCalls)
	return base
}

func validateInspection(inspection *findings.Inspection) error {
	if inspection == nil || !inspection.Required {
		return nil
	}
	if !inspection.DiffInspected {
		return fmt.Errorf("hosted provider did not inspect the diff with git_diff")
	}
	return nil
}

func filterReviewableFiles(files []diff.FileChange) []diff.FileChange {
	out := make([]diff.FileChange, 0, len(files))
	for _, file := range files {
		if file.Status == diff.ChangeDeleted || file.ToPath == "/dev/null" {
			continue
		}
		out = append(out, file)
	}
	return out
}

func reviewedFiles(files []diff.FileChange) []findings.ReviewedFile {
	out := make([]findings.ReviewedFile, 0, len(files))
	for _, file := range files {
		path := strings.TrimSpace(file.ToPath)
		if path == "" || path == "/dev/null" {
			path = strings.TrimSpace(file.FromPath)
		}
		if path == "" || path == "/dev/null" {
			continue
		}
		out = append(out, findings.ReviewedFile{
			Path:   path,
			Status: string(file.Status),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

func normalizeChangeSummary(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		item = strings.TrimPrefix(item, "- ")
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	const maxSummaryItems = 8
	if len(out) > maxSummaryItems {
		return out[:maxSummaryItems]
	}
	return out
}

func chunkInputFromChanges(reviewID, repo, baseSHA, headSHA, language string, reviewChecks []string, instructions string, testSummary string, commitMessages []string, chunkIndex, chunkCount int, chunk []diff.FileChange) ChunkInput {
	files := chunkFilesFromChanges(chunk)
	return ChunkInput{
		ReviewID:              reviewID,
		Repo:                  repo,
		BaseSHA:               baseSHA,
		HeadSHA:               headSHA,
		ChunkIndex:            chunkIndex,
		ChunkCount:            chunkCount,
		ReviewTask:            promptpack.ReviewTask(reviewChecks),
		UntrustedInputWarning: promptpack.UntrustedInputWarning,
		UntrustedInputStart:   promptpack.UntrustedInputStart,
		UntrustedInputEnd:     promptpack.UntrustedInputEnd,
		Language:              language,
		ReviewChecks:          append([]string(nil), reviewChecks...),
		Instructions:          strings.TrimSpace(instructions),
		TestSummary:           testSummary,
		CommitMessages:        append([]string(nil), commitMessages...),
		Files:                 files,
	}
}

func chunkFilesFromChanges(changes []diff.FileChange) []ChunkFile {
	files := make([]ChunkFile, 0, len(changes))
	for _, file := range changes {
		spans := make([]ChunkSpan, 0, len(file.ChangedLineSpans))
		for _, span := range file.ChangedLineSpans {
			spans = append(spans, ChunkSpan{Start: span.Start, End: span.End})
		}
		item := ChunkFile{
			Path:   file.ToPath,
			Status: string(file.Status),
			Spans:  spans,
		}
		if file.IsRename && file.FromPath != file.ToPath {
			item.PreviousPath = file.FromPath
		}
		files = append(files, item)
	}
	return files
}

func chunkFileChanges(files []diff.FileChange, maxFilesPerChunk int) [][]diff.FileChange {
	if maxFilesPerChunk <= 0 {
		maxFilesPerChunk = 20
	}
	chunks := [][]diff.FileChange{}
	for start := 0; start < len(files); start += maxFilesPerChunk {
		end := start + maxFilesPerChunk
		if end > len(files) {
			end = len(files)
		}
		chunks = append(chunks, append([]diff.FileChange(nil), files[start:end]...))
	}
	return chunks
}

func testSummaryFromFiles(files []diff.FileChange) string {
	testCount := 0
	for _, file := range files {
		if strings.HasSuffix(file.ToPath, "_test.go") {
			testCount++
		}
	}
	if testCount == 0 {
		return "no_tests_in_diff"
	}
	return fmt.Sprintf("%d_test_files_changed", testCount)
}

func collectCommitMessages(workingDir, baseSHA, headSHA string) ([]string, error) {
	if strings.TrimSpace(baseSHA) == "" || strings.TrimSpace(headSHA) == "" {
		return nil, nil
	}
	cmd := exec.Command("git", "log", "--format=%s", "--max-count=20", fmt.Sprintf("%s..%s", baseSHA, headSHA))
	cmd.Dir = workingDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("collect commit messages: %w", err)
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	messages := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		messages = append(messages, promptpack.EscapeUntrusted(line))
	}
	return messages, nil
}

func providersWithEnv(in map[string]dpconfig.ProviderConfig) map[string]dpconfig.ProviderConfig {
	out := make(map[string]dpconfig.ProviderConfig, len(in))
	for key, cfg := range in {
		copied := cfg
		if cfg.OpenAI != nil {
			block := *cfg.OpenAI
			if strings.TrimSpace(block.APIKey) == "" {
				block.APIKey = os.Getenv("OPENAI_API_KEY")
			}
			copied.OpenAI = &block
		}
		if cfg.AIStudio != nil {
			block := *cfg.AIStudio
			if strings.TrimSpace(block.APIKey) == "" {
				block.APIKey = os.Getenv("GEMINI_API_KEY")
			}
			copied.AIStudio = &block
		}
		out[key] = copied
	}
	return out
}

func validateChunkFindings(items []ChunkFinding, files []ChunkFile, providerID string, allowedCategories map[string]struct{}) []findings.Finding {
	if len(items) == 0 {
		return nil
	}
	allowed := make(map[string][]ChunkSpan, len(files))
	for _, file := range files {
		allowed[file.Path] = append([]ChunkSpan(nil), file.Spans...)
	}
	out := make([]findings.Finding, 0, len(items))
	for _, item := range items {
		finding, ok := normalizeChunkFinding(item, allowed, providerID)
		if !ok {
			continue
		}
		if _, ok := allowedCategories[finding.Category]; !ok {
			continue
		}
		out = append(out, finding)
	}
	return out
}

func categoriesForReviewChecks(checks []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, check := range checks {
		switch check {
		case "security":
			out["security"] = struct{}{}
		case "bugs":
			out["correctness"] = struct{}{}
			out["reliability"] = struct{}{}
		case "performance":
			out["performance"] = struct{}{}
		case "best-practices":
			out["maintainability"] = struct{}{}
			out["testing"] = struct{}{}
			out["style"] = struct{}{}
		}
	}
	return out
}

func normalizeChunkFinding(item ChunkFinding, allowed map[string][]ChunkSpan, providerID string) (findings.Finding, bool) {
	category := strings.ToLower(strings.TrimSpace(item.Category))
	severity := strings.ToLower(strings.TrimSpace(item.Severity))
	path := strings.TrimSpace(item.Path)
	title := strings.TrimSpace(item.Title)
	message := strings.TrimSpace(item.Message)
	suggestion := strings.TrimSpace(item.Suggestion)
	path, startLine, endLine := normalizeFindingLocation(path, item.StartLine, item.EndLine, item.ChangedSpan)

	if !allowedCategory(category) || !allowedSeverity(severity) {
		return findings.Finding{}, false
	}
	if path == "" || title == "" || message == "" || strings.TrimSpace(item.Evidence.Anchor) == "" || strings.TrimSpace(item.Evidence.ReasoningBasis) == "" || strings.TrimSpace(item.Evidence.Source) == "" || strings.TrimSpace(item.Impact.Summary) == "" || strings.TrimSpace(item.Impact.Scope) == "" {
		return findings.Finding{}, false
	}
	if item.Confidence < 0 || item.Confidence > 1 {
		return findings.Finding{}, false
	}
	if startLine <= 0 || endLine <= 0 || startLine > endLine {
		return findings.Finding{}, false
	}
	if !allowedRange(path, startLine, endLine, allowed) {
		return findings.Finding{}, false
	}

	return findings.Finding{
		Category:       category,
		Severity:       severity,
		Confidence:     item.Confidence,
		Path:           path,
		StartLine:      startLine,
		EndLine:        endLine,
		ChangedSpan:    findings.LineSpan{Path: path, StartLine: startLine, EndLine: endLine},
		SupportingSpan: item.SupportingSpan,
		Title:          title,
		Message:        message,
		Evidence:       item.Evidence,
		Impact:         item.Impact,
		Suggestion:     suggestion,
		Provider:       providerID,
	}, true
}

func normalizeFindingLocation(path string, startLine, endLine int, span findings.LineSpan) (string, int, int) {
	if strings.TrimSpace(span.Path) != "" {
		path = strings.TrimSpace(span.Path)
	}
	if span.StartLine > 0 {
		startLine = span.StartLine
	}
	if span.EndLine > 0 {
		endLine = span.EndLine
	}
	return path, startLine, endLine
}

func allowedCategory(category string) bool {
	switch category {
	case "security", "correctness", "reliability", "performance", "maintainability", "testing", "style":
		return true
	default:
		return false
	}
}

func allowedSeverity(severity string) bool {
	switch severity {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func allowedRange(path string, startLine, endLine int, allowed map[string][]ChunkSpan) bool {
	spans, ok := allowed[path]
	if !ok {
		return false
	}
	for _, span := range spans {
		if startLine >= span.Start && endLine <= span.End {
			return true
		}
	}
	return false
}

func dedupeAndSortFindings(items []findings.Finding, repo, reviewID, headSHA string) []findings.Finding {
	seen := map[string]struct{}{}
	out := make([]findings.Finding, 0, len(items))
	for _, item := range items {
		item.ReviewID = reviewID
		fp := findings.Fingerprint(repo, headSHA, item)
		if _, ok := seen[fp]; ok {
			continue
		}
		seen[fp] = struct{}{}
		out = append(out, item)
	}
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
		if left.Category != right.Category {
			return left.Category < right.Category
		}
		if left.Message != right.Message {
			return left.Message < right.Message
		}
		return left.EvidenceText() < right.EvidenceText()
	})
	return out
}

func applyBlockingPolicy(bundle *findings.FindingsBundle, blockOn string) error {
	sev, err := policy.ParseSeverity(strings.ToLower(strings.TrimSpace(blockOn)))
	if err != nil {
		return err
	}
	items := make([]policy.Finding, 0, len(bundle.Findings))
	for _, item := range bundle.Findings {
		parsed, err := policy.ParseSeverity(item.Severity)
		if err != nil {
			return err
		}
		items = append(items, policy.Finding{
			Severity:   parsed,
			Confidence: item.Confidence,
			Path:       item.Path,
		})
	}
	decisions := policy.ApplyPolicy(policy.Policy{BlockOn: sev}, items)
	for i := range bundle.Findings {
		bundle.Findings[i].Blocking = decisions[i].Action == "block"
	}
	return nil
}
