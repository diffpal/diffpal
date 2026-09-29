package findings

import "encoding/json"

// Rule findings omit code-review-only fields while keeping the same v5
// finding shape and preserving DiffPal's numeric confidence internally.
func (f Finding) MarshalJSON() ([]byte, error) {
	type plain Finding
	if f.Evidence.Kind == "rule" {
		return json.Marshal(struct {
			plain
			Confidence *float64       `json:"confidence,omitempty"`
			Impact     *FindingImpact `json:"impact,omitempty"`
		}{plain: plain(f)})
	}
	return json.Marshal(plain(f))
}
