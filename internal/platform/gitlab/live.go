package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	gl "gitlab.com/gitlab-org/api/client-go"
)

const summaryDiscussionMarker = "<!-- diffpal:summary -->"

func PublishDiscussions(ctx context.Context, tokenMode, token string, reviewCtx Context, plan DiscussionPlan, client *http.Client) error {
	if len(plan.Actions) == 0 {
		return nil
	}
	gitlabClient, mrIID, err := newClient(tokenMode, token, reviewCtx, client)
	if err != nil {
		return err
	}
	version, err := latestDiffVersion(ctx, gitlabClient, reviewCtx, mrIID)
	if err != nil {
		return err
	}
	positions := discussionPositions(version)
	existing, err := activeDiscussionState(ctx, gitlabClient, reviewCtx, mrIID)
	if err != nil {
		return err
	}
	for _, action := range plan.Actions {
		if strings.TrimSpace(action.Body) == "" {
			continue
		}
		if err := publishFindingDiscussion(ctx, gitlabClient, reviewCtx, mrIID, action, positions, existing); err != nil {
			return err
		}
	}
	return nil
}

func PublishSummaryDiscussion(ctx context.Context, tokenMode, token string, reviewCtx Context, summary string, client *http.Client) error {
	gitlabClient, mrIID, err := newClient(tokenMode, token, reviewCtx, client)
	if err != nil {
		return err
	}
	return publishSummary(ctx, gitlabClient, reviewCtx, mrIID, summary)
}

func publishSummary(ctx context.Context, gitlabClient *gl.Client, reviewCtx Context, mrIID int64, summary string) error {
	body := strings.TrimSpace(summary)
	if body == "" {
		return nil
	}
	body = summaryDiscussionMarker + "\n" + body + "\n"
	discussionID, noteID, err := findSummaryDiscussion(ctx, gitlabClient, reviewCtx, mrIID)
	if err != nil {
		return err
	}
	if discussionID != "" && noteID > 0 {
		_, _, err := gitlabClient.Discussions.UpdateMergeRequestDiscussionNote(reviewCtx.Repo, mrIID, discussionID, noteID, &gl.UpdateMergeRequestDiscussionNoteOptions{
			Body: &body,
		}, gl.WithContext(ctx))
		return err
	}
	_, _, err = gitlabClient.Discussions.CreateMergeRequestDiscussion(reviewCtx.Repo, mrIID, &gl.CreateMergeRequestDiscussionOptions{
		Body: &body,
	}, gl.WithContext(ctx))
	return err
}

func PublishStatus(ctx context.Context, tokenMode, token string, reviewCtx Context, payload StatusPayload, client *http.Client) error {
	gitlabClient, _, err := newClient(tokenMode, token, reviewCtx, client)
	if err != nil {
		return err
	}
	if strings.TrimSpace(reviewCtx.HeadSHA) == "" {
		return fmt.Errorf("missing GitLab head SHA")
	}
	state := gl.BuildStateValue(payload.State)
	opt := &gl.SetCommitStatusOptions{
		State:       state,
		Name:        stringPtr(payload.Name),
		Context:     stringPtr(payload.Context),
		Description: stringPtr(payload.Description),
	}
	if strings.TrimSpace(reviewCtx.SourceBranch) != "" {
		opt.Ref = stringPtr(reviewCtx.SourceBranch)
	}
	if strings.TrimSpace(payload.TargetURL) != "" {
		opt.TargetURL = stringPtr(payload.TargetURL)
	}
	_, _, err = gitlabClient.Commits.SetCommitStatus(reviewCtx.Repo, reviewCtx.HeadSHA, opt, gl.WithContext(ctx))
	return err
}

type existingDiscussion struct {
	DiscussionID string
	NoteID       int64
	FindingID    string
	Resolved     bool
}

func publishFindingDiscussion(ctx context.Context, gitlabClient *gl.Client, reviewCtx Context, mrIID int64, action DiscussionAction, positions map[string]gl.PositionOptions, existing map[string]existingDiscussion) error {
	body := action.Body + "\n" + findingMarker(action.FindingID) + "\n"
	key := discussionKey(action.Path, action.Line, "", action.FindingID)
	if found, ok := existing[action.FindingID]; ok {
		if strings.TrimSpace(found.DiscussionID) == "" || found.NoteID <= 0 {
			return nil
		}
		_, _, err := gitlabClient.Discussions.UpdateMergeRequestDiscussionNote(reviewCtx.Repo, mrIID, found.DiscussionID, found.NoteID, &gl.UpdateMergeRequestDiscussionNoteOptions{
			Body: &body,
		}, gl.WithContext(ctx))
		if err != nil {
			return err
		}
		return setDiscussionResolved(ctx, gitlabClient, reviewCtx, mrIID, found.DiscussionID, action.Resolved)
	}
	position, ok := positions[positionKey(action.Path, action.Line)]
	if !ok {
		return publishFallbackFinding(ctx, gitlabClient, reviewCtx, mrIID, body, action.Resolved)
	}
	discussion, _, err := gitlabClient.Discussions.CreateMergeRequestDiscussion(reviewCtx.Repo, mrIID, &gl.CreateMergeRequestDiscussionOptions{
		Body:     &body,
		Position: &position,
	}, gl.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("create GitLab discussion for %s: %w", key, err)
	}
	if discussion == nil || discussion.ID == "" {
		return nil
	}
	return setDiscussionResolved(ctx, gitlabClient, reviewCtx, mrIID, discussion.ID, action.Resolved)
}

func publishFallbackFinding(ctx context.Context, gitlabClient *gl.Client, reviewCtx Context, mrIID int64, body string, resolved bool) error {
	discussion, _, err := gitlabClient.Discussions.CreateMergeRequestDiscussion(reviewCtx.Repo, mrIID, &gl.CreateMergeRequestDiscussionOptions{
		Body: &body,
	}, gl.WithContext(ctx))
	if err != nil {
		return err
	}
	if discussion == nil || discussion.ID == "" {
		return nil
	}
	return setDiscussionResolved(ctx, gitlabClient, reviewCtx, mrIID, discussion.ID, resolved)
}

func setDiscussionResolved(ctx context.Context, gitlabClient *gl.Client, reviewCtx Context, mrIID int64, discussionID string, resolved bool) error {
	_, _, err := gitlabClient.Discussions.ResolveMergeRequestDiscussion(reviewCtx.Repo, mrIID, discussionID, &gl.ResolveMergeRequestDiscussionOptions{
		Resolved: &resolved,
	}, gl.WithContext(ctx))
	return err
}

func latestDiffVersion(ctx context.Context, gitlabClient *gl.Client, reviewCtx Context, mrIID int64) (*gl.MergeRequestDiffVersion, error) {
	versions, _, err := gitlabClient.MergeRequests.GetMergeRequestDiffVersions(reviewCtx.Repo, mrIID, &gl.GetMergeRequestDiffVersionsOptions{}, gl.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 || versions[0] == nil {
		return nil, fmt.Errorf("GitLab merge request has no diff versions")
	}
	latest := versions[0]
	for _, candidate := range versions[1:] {
		if candidate != nil && candidate.ID > latest.ID {
			latest = candidate
		}
	}
	full, _, err := gitlabClient.MergeRequests.GetSingleMergeRequestDiffVersion(reviewCtx.Repo, mrIID, latest.ID, nil, gl.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	if full == nil {
		return latest, nil
	}
	return full, nil
}

func discussionPositions(version *gl.MergeRequestDiffVersion) map[string]gl.PositionOptions {
	if version == nil {
		return nil
	}
	out := map[string]gl.PositionOptions{}
	for _, item := range version.Diffs {
		if item == nil {
			continue
		}
		for _, line := range parseDiffLines(item.Diff) {
			positionType := "text"
			pos := gl.PositionOptions{
				BaseSHA:      stringPtr(version.BaseCommitSHA),
				StartSHA:     stringPtr(version.StartCommitSHA),
				HeadSHA:      stringPtr(version.HeadCommitSHA),
				NewPath:      stringPtr(item.NewPath),
				OldPath:      stringPtr(item.OldPath),
				PositionType: &positionType,
			}
			if line.NewLine > 0 {
				pos.NewLine = int64Ptr(int64(line.NewLine))
			}
			if line.OldLine > 0 {
				pos.OldLine = int64Ptr(int64(line.OldLine))
			}
			if line.NewLine > 0 {
				out[positionKey(item.NewPath, line.NewLine)] = pos
			}
			if item.RenamedFile && line.NewLine > 0 {
				out[positionKey(item.OldPath, line.NewLine)] = pos
			}
		}
	}
	return out
}

type diffLine struct {
	OldLine int
	NewLine int
}

var hunkHeaderRE = regexp.MustCompile(`^@@ -([0-9]+)(?:,[0-9]+)? \+([0-9]+)(?:,[0-9]+)? @@`)

func parseDiffLines(raw string) []diffLine {
	var out []diffLine
	oldLine, newLine := 0, 0
	for _, line := range strings.Split(raw, "\n") {
		if line == "" {
			continue
		}
		if matches := hunkHeaderRE.FindStringSubmatch(line); len(matches) == 3 {
			oldLine, _ = strconv.Atoi(matches[1])
			newLine, _ = strconv.Atoi(matches[2])
			continue
		}
		if oldLine == 0 && newLine == 0 {
			continue
		}
		switch {
		case strings.HasPrefix(line, `\`):
			continue
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			out = append(out, diffLine{NewLine: newLine})
			newLine++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			out = append(out, diffLine{OldLine: oldLine})
			oldLine++
		default:
			out = append(out, diffLine{OldLine: oldLine, NewLine: newLine})
			oldLine++
			newLine++
		}
	}
	return out
}

func activeDiscussionState(ctx context.Context, gitlabClient *gl.Client, reviewCtx Context, mrIID int64) (map[string]existingDiscussion, error) {
	out := map[string]existingDiscussion{}
	opt := &gl.ListMergeRequestDiscussionsOptions{ListOptions: gl.ListOptions{PerPage: 100}}
	for {
		discussions, resp, err := gitlabClient.Discussions.ListMergeRequestDiscussions(reviewCtx.Repo, mrIID, opt, gl.WithContext(ctx))
		if err != nil {
			return nil, err
		}
		for _, discussion := range discussions {
			if discussion == nil {
				continue
			}
			for _, note := range discussion.Notes {
				if note == nil {
					continue
				}
				findingID := findingIDFromMarker(note.Body)
				if findingID == "" {
					continue
				}
				out[findingID] = existingDiscussion{
					DiscussionID: discussion.ID,
					NoteID:       note.ID,
					FindingID:    findingID,
					Resolved:     note.Resolved,
				}
			}
		}
		if resp == nil || resp.NextPage == 0 {
			return out, nil
		}
		opt.Page = resp.NextPage
	}
}

func findSummaryDiscussion(ctx context.Context, gitlabClient *gl.Client, reviewCtx Context, mrIID int64) (string, int64, error) {
	opt := &gl.ListMergeRequestDiscussionsOptions{ListOptions: gl.ListOptions{PerPage: 100}}
	for {
		discussions, resp, err := gitlabClient.Discussions.ListMergeRequestDiscussions(reviewCtx.Repo, mrIID, opt, gl.WithContext(ctx))
		if err != nil {
			return "", 0, err
		}
		for _, discussion := range discussions {
			if discussion == nil {
				continue
			}
			for _, note := range discussion.Notes {
				if note != nil && strings.Contains(note.Body, summaryDiscussionMarker) {
					return discussion.ID, note.ID, nil
				}
			}
		}
		if resp == nil || resp.NextPage == 0 {
			return "", 0, nil
		}
		opt.Page = resp.NextPage
	}
}

func findingMarker(findingID string) string {
	id := strings.NewReplacer("--", "-", "\n", " ", "\r", " ").Replace(strings.TrimSpace(findingID))
	if id == "" {
		return ""
	}
	return "<!-- diffpal:finding id:" + id + " -->"
}

func findingIDFromMarker(body string) string {
	const prefix = "<!-- diffpal:finding "
	idx := strings.Index(body, prefix)
	if idx < 0 {
		return ""
	}
	end := strings.Index(body[idx:], "-->")
	if end < 0 {
		return ""
	}
	marker := strings.TrimSuffix(strings.TrimPrefix(body[idx:idx+end+3], "<!--"), "-->")
	for _, part := range strings.Fields(marker) {
		key, value, ok := strings.Cut(part, ":")
		if ok && key == "id" {
			return value
		}
	}
	return ""
}

func positionKey(path string, line int) string {
	return strings.TrimSpace(path) + ":" + strconv.Itoa(line)
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}

func newClient(tokenMode, token string, reviewCtx Context, client *http.Client) (*gl.Client, int64, error) {
	if strings.TrimSpace(reviewCtx.Repo) == "" {
		return nil, 0, fmt.Errorf("missing GitLab repository/project")
	}
	if strings.TrimSpace(reviewCtx.MergeRequestIID) == "" {
		return nil, 0, fmt.Errorf("missing GitLab merge request iid")
	}
	mrIID, err := strconv.ParseInt(reviewCtx.MergeRequestIID, 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid GitLab merge request iid %q: %w", reviewCtx.MergeRequestIID, err)
	}
	options := []gl.ClientOptionFunc{
		gl.WithBaseURL(gitLabAPIBaseURL(reviewCtx)),
	}
	if client != nil {
		options = append(options, gl.WithHTTPClient(client))
	}
	switch tokenMode {
	case "ci_job_token":
		gitlabClient, err := gl.NewJobClient(token, options...)
		return gitlabClient, mrIID, err
	default:
		gitlabClient, err := gl.NewClient(token, options...)
		return gitlabClient, mrIID, err
	}
}

func gitLabAPIBaseURL(ctx Context) string {
	if override := strings.TrimSpace(os.Getenv("DIFFPAL_GITLAB_API_URL")); override != "" {
		return override
	}
	if strings.TrimSpace(ctx.WebURL) != "" {
		parsed, err := url.Parse(ctx.WebURL)
		if err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return parsed.Scheme + "://" + parsed.Host + "/api/v4"
		}
	}
	return "https://gitlab.com/api/v4"
}
