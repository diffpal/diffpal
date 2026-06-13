package policy

import "testing"

func TestApplyPolicySupportsRecursivePathSuppressionsAndThresholds(t *testing.T) {
	t.Parallel()

	findings := []Finding{
		{Severity: SeverityMedium, Path: "pkg/a/file.snap"},
		{Severity: SeverityCritical, Path: "vendor/github.com/x/y.go"},
		{Severity: SeverityHigh, Path: "internal/app/main.go"},
		{Severity: SeverityLow, Path: "internal/app/style.go"},
	}

	decisions := ApplyPolicy(Policy{
		BlockOn: SeverityHigh,
		WarnOn:  []Severity{SeverityLow, SeverityMedium},
		Exclude: []Suppression{
			{Path: "**/*.snap"},
			{Path: "vendor/**"},
		},
	}, findings)

	if decisions[0].Action != "suppress" {
		t.Fatalf("snap action = %q, want suppress", decisions[0].Action)
	}
	if decisions[1].Action != "suppress" {
		t.Fatalf("vendor action = %q, want suppress", decisions[1].Action)
	}
	if decisions[2].Action != "block" {
		t.Fatalf("panic action = %q, want block", decisions[2].Action)
	}
	if decisions[3].Action != "warn" {
		t.Fatalf("style action = %q, want warn", decisions[3].Action)
	}
}

func TestParseSeverityRejectsUnknownValue(t *testing.T) {
	t.Parallel()

	if _, err := ParseSeverity("unknown"); err == nil {
		t.Fatal("ParseSeverity() error = nil, want error")
	}
}

func TestMatchPathGlob(t *testing.T) {
	t.Parallel()

	cases := []struct {
		pattern string
		target  string
		want    bool
	}{
		{pattern: "vendor/**", target: "vendor/github.com/x/file.go", want: true},
		{pattern: "**/*.snap", target: "pkg/a/file.snap", want: true},
		{pattern: "internal/*.go", target: "internal/app/main.go", want: false},
		{pattern: "internal/**", target: "internal/app/main.go", want: true},
	}
	for _, tc := range cases {
		if got := matchPathGlob(tc.pattern, tc.target); got != tc.want {
			t.Fatalf("matchPathGlob(%q, %q) = %t, want %t", tc.pattern, tc.target, got, tc.want)
		}
	}
}
