package findings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestV5RuleFindingRoundTrip(t *testing.T) {
	fixture := filepath.Join("..", "..", "schemas", "findings", "testdata", "lintpal-right.json")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "findings.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	bundle, err := ReadBundle(path)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := FormatBundle(bundle, "test-repo")
	if err != nil {
		t.Fatal(err)
	}
	validateV5Schema(t, encoded)
	var original, written map[string]any
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &written); err != nil {
		t.Fatal(err)
	}
	initial := original["findings"].([]any)[0].(map[string]any)
	result := written["findings"].([]any)[0].(map[string]any)
	for _, key := range []string{"id", "decision", "evidence", "changed_span", "work_item_id", "model"} {
		before, _ := json.Marshal(initial[key])
		after, _ := json.Marshal(result[key])
		if string(before) != string(after) {
			t.Fatalf("%s changed: %s => %s", key, before, after)
		}
	}
	for _, key := range []string{"confidence", "impact"} {
		if _, ok := result[key]; ok {
			t.Fatalf("rule finding unexpectedly has %s", key)
		}
	}
}

func validateV5Schema(t *testing.T, raw []byte) {
	t.Helper()
	schemaRaw, err := os.ReadFile(filepath.Join("..", "..", "schemas", "findings", "v5.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document, value any
	if err := json.Unmarshal(schemaRaw, &document); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("v5.schema.json", document); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile("v5.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.Validate(value); err != nil {
		t.Fatalf("v5 schema validation: %v", err)
	}
}
