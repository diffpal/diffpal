package reviewer

import (
	"strings"
	"testing"
)

func TestReviewInstructionDefinesReviewCheckCategories(t *testing.T) {
	instruction := reviewInstruction("")
	required := []string{
		"security covers security",
		"bugs covers correctness and reliability",
		"When security is requested",
		"When bugs is requested",
		"input.instructions",
		"Use severity critical",
	}
	for _, want := range required {
		if !strings.Contains(instruction, want) {
			t.Fatalf("reviewInstruction() missing %q:\n%s", want, instruction)
		}
	}
}

func TestReviewInstructionIncludesCustomInstructions(t *testing.T) {
	instruction := reviewInstruction("Focus on auth boundary changes.")
	if !strings.Contains(instruction, "Repository-local custom instructions:") {
		t.Fatalf("reviewInstruction() missing custom instructions heading:\n%s", instruction)
	}
	if !strings.Contains(instruction, "Focus on auth boundary changes.") {
		t.Fatalf("reviewInstruction() missing custom instruction:\n%s", instruction)
	}
}
