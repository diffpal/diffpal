package github

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/diffpal/diffpal/internal/findings"
)

type PermanentLinkProvider struct {
	Repo    string
	BaseSHA string
	HeadSHA string
}

func NewPermanentLinkProvider(ctx Context) PermanentLinkProvider {
	return PermanentLinkProvider{
		Repo:    ctx.Repo,
		BaseSHA: ctx.BaseSHA,
		HeadSHA: ctx.HeadSHA,
	}
}

func (p PermanentLinkProvider) Link(finding findings.Finding) (string, bool) {
	link := PermanentLinkForSide(p.Repo, p.BaseSHA, p.HeadSHA, finding)
	return link, link != ""
}

func PermanentLink(repo string, headSHA string, finding findings.Finding) string {
	return permanentLink(repo, headSHA, finding)
}

func PermanentLinkForSide(repo string, baseSHA string, headSHA string, finding findings.Finding) string {
	sha := headSHA
	side, ok := canonicalCommentSide(finding.ChangedSpan.Side)
	if !ok {
		return ""
	}
	if side == findings.SideLeft {
		sha = baseSHA
	}
	return permanentLink(repo, sha, finding)
}

func permanentLink(repo string, sha string, finding findings.Finding) string {
	repo = strings.TrimSpace(repo)
	sha = strings.TrimSpace(sha)
	if repo == "" || sha == "" || finding.StartLine <= 0 {
		return ""
	}
	cleanPath := cleanGitHubPath(finding.Path)
	if cleanPath == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/blob/%s/%s#%s", repo, url.PathEscape(sha), cleanPath, lineFragment(finding.StartLine, finding.EndLine))
}

func cleanGitHubPath(raw string) string {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
	if raw == "" || strings.HasPrefix(raw, "/") {
		return ""
	}
	cleaned := path.Clean(raw)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	parts := strings.Split(cleaned, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func lineFragment(start int, end int) string {
	if end <= 0 || end == start {
		return fmt.Sprintf("L%d", start)
	}
	return fmt.Sprintf("L%d-L%d", start, end)
}
