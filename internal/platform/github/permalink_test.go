package github

import (
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestPermanentLinkBuildsLineRange(t *testing.T) {
	got := PermanentLink("acme/diffpal", "head-a", findings.Finding{
		Path:      "internal/db/query.go",
		StartLine: 12,
		EndLine:   17,
	})
	want := "https://github.com/acme/diffpal/blob/head-a/internal/db/query.go#L12-L17"
	if got != want {
		t.Fatalf("PermanentLink() = %q, want %q", got, want)
	}
}

func TestPermanentLinkProviderUsesBaseForLeftAndHeadForRight(t *testing.T) {
	t.Parallel()

	provider := NewPermanentLinkProvider(Context{Repo: "acme/diffpal", BaseSHA: "base-a", HeadSHA: "head-a"})
	left, ok := provider.Link(findings.Finding{Path: "orders.go", StartLine: 12, ChangedSpan: findings.LineSpan{Side: findings.SideLeft}})
	if !ok || left != "https://github.com/acme/diffpal/blob/base-a/orders.go#L12" {
		t.Fatalf("LEFT link = %q, %v; want base SHA", left, ok)
	}
	right, ok := provider.Link(findings.Finding{Path: "orders.go", StartLine: 14, ChangedSpan: findings.LineSpan{Side: findings.SideRight}})
	if !ok || right != "https://github.com/acme/diffpal/blob/head-a/orders.go#L14" {
		t.Fatalf("RIGHT link = %q, %v; want head SHA", right, ok)
	}
}

func TestPermanentLinkEscapesPathSegments(t *testing.T) {
	got := PermanentLink("acme/diffpal", "head-a", findings.Finding{
		Path:      "src/user form/app+test.js",
		StartLine: 42,
	})
	want := "https://github.com/acme/diffpal/blob/head-a/src/user%20form/app+test.js#L42"
	if got != want {
		t.Fatalf("PermanentLink() = %q, want %q", got, want)
	}
}

func TestPermanentLinkRejectsMissingOrUnsafeInputs(t *testing.T) {
	cases := []findings.Finding{
		{Path: "main.go", StartLine: 0},
		{Path: "", StartLine: 1},
		{Path: "../main.go", StartLine: 1},
		{Path: "/main.go", StartLine: 1},
	}
	for _, tc := range cases {
		if got := PermanentLink("acme/diffpal", "head-a", tc); got != "" {
			t.Fatalf("PermanentLink(%+v) = %q, want empty", tc, got)
		}
	}
	if got := PermanentLink("", "head-a", findings.Finding{Path: "main.go", StartLine: 1}); got != "" {
		t.Fatalf("PermanentLink() with missing repo = %q, want empty", got)
	}
	if got := PermanentLink("acme/diffpal", "", findings.Finding{Path: "main.go", StartLine: 1}); got != "" {
		t.Fatalf("PermanentLink() with missing head = %q, want empty", got)
	}
}
