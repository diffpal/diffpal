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
	HeadSHA string
}

func NewPermanentLinkProvider(ctx Context) PermanentLinkProvider {
	return PermanentLinkProvider{
		Repo:    ctx.Repo,
		HeadSHA: ctx.HeadSHA,
	}
}

func (p PermanentLinkProvider) Link(finding findings.Finding) (string, bool) {
	link := PermanentLink(p.Repo, p.HeadSHA, finding)
	return link, link != ""
}

func PermanentLink(repo string, headSHA string, finding findings.Finding) string {
	repo = strings.TrimSpace(repo)
	headSHA = strings.TrimSpace(headSHA)
	if repo == "" || headSHA == "" || finding.StartLine <= 0 {
		return ""
	}
	cleanPath := cleanGitHubPath(finding.Path)
	if cleanPath == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/blob/%s/%s#%s", repo, url.PathEscape(headSHA), cleanPath, lineFragment(finding.StartLine, finding.EndLine))
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
