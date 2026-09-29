package schema_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestSharedFindingsSchema(t *testing.T) {
	raw, err := os.ReadFile("v5.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	const canonicalSHA256 = "0f650d402985add03eaa61bf5212420026826b8a934911fef7d1d9ba126a7a5e"
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != canonicalSHA256 {
		t.Fatalf("schema digest = %s, want canonical %s", got, canonicalSHA256)
	}
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
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
	for _, tc := range []struct {
		name  string
		valid bool
	}{
		{"lintpal-right.json", true},
		{"diffpal-left.json", true},
		{"invalid-rule-missing-decision.json", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", tc.name))
			if err != nil {
				t.Fatal(err)
			}
			var value any
			if err := json.Unmarshal(raw, &value); err != nil {
				t.Fatal(err)
			}
			err = compiled.Validate(value)
			if (err == nil) != tc.valid {
				t.Fatalf("valid = %t, want %t: %v", err == nil, tc.valid, err)
			}
		})
	}
}
