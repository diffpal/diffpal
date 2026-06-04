package cmd

import (
	"reflect"
	"testing"
)

func TestDefaultModesMatchProductContract(t *testing.T) {
	t.Parallel()

	cases := []struct {
		platform string
		want     []string
	}{
		{platform: "github", want: []string{"check-run", "comments", "sarif", "summary"}},
		{platform: "gitlab", want: []string{"code-quality", "discussions", "sarif", "summary"}},
		{platform: "azure", want: []string{"threads", "status"}},
	}
	for _, tc := range cases {
		if got := defaultModes(tc.platform); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("defaultModes(%q) = %v, want %v", tc.platform, got, tc.want)
		}
	}
}
