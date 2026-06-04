package provider

import "testing"

func TestSuggestProviderPoolReturnsSortedKeys(t *testing.T) {
	t.Parallel()

	keys, warnings := SuggestProviderPool([]DetectedProvider{
		{Key: "copilot-acp"},
		{Key: "openai-fast"},
	})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if len(keys) != 2 || keys[0] != "copilot-acp" || keys[1] != "openai-fast" {
		t.Fatalf("keys = %v, want sorted provider keys", keys)
	}
}

func TestSuggestProviderPoolWarnsWhenNoDetectedProviders(t *testing.T) {
	t.Parallel()

	keys, warnings := SuggestProviderPool(nil)
	if len(keys) != 0 {
		t.Fatalf("keys = %v, want none", keys)
	}
	if len(warnings) == 0 {
		t.Fatal("warnings = nil, want advisory warning")
	}
}
