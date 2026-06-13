package sarif

import (
	"encoding/json"
	"sort"

	"github.com/diffpal/diffpal/internal/findings"
)

type Report struct {
	Version string `json:"version"`
	Schema  string `json:"$schema"`
	Runs    []Run  `json:"runs"`
}

type Run struct {
	Tool    Tool     `json:"tool"`
	Results []Result `json:"results"`
}

type Tool struct {
	Driver ToolDriver `json:"driver"`
}

type ToolDriver struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	InformationURI string `json:"informationUri"`
	Rules          []Rule `json:"rules,omitempty"`
}

type Rule struct {
	ID               string         `json:"id"`
	Name             string         `json:"name,omitempty"`
	ShortDescription Message        `json:"shortDescription,omitempty"`
	Properties       RuleProperties `json:"properties,omitempty"`
}

type RuleProperties struct {
	Category string `json:"category,omitempty"`
}

type Result struct {
	Rule                string            `json:"ruleId"`
	Level               string            `json:"level"`
	Message             Message           `json:"message"`
	Locations           []Location        `json:"locations"`
	PartialFingerprints map[string]string `json:"partialFingerprints,omitempty"`
	Properties          ResultProperties  `json:"properties,omitempty"`
}

type ResultProperties struct {
	Category   string  `json:"category,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Blocking   bool    `json:"blocking,omitempty"`
}

type Message struct {
	Text string `json:"text"`
}

type Location struct {
	PhysicalLocation PhysicalLocation `json:"physicalLocation"`
}

type PhysicalLocation struct {
	ArtifactLocation ArtifactLocation `json:"artifactLocation"`
	Region           Region           `json:"region"`
}

type ArtifactLocation struct {
	URI string `json:"uri"`
}

type Region struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
}

func ToReport(bundle findings.FindingsBundle) Report {
	r := Report{
		Version: "2.1.0",
		Schema:  "https://schemastore.azureedge.net/schemas/json/sarif-2.1.0-rtm.5.json",
		Runs: []Run{
			{
				Tool: Tool{
					Driver: ToolDriver{
						Name:           "DiffPal",
						Version:        "0.1.0",
						InformationURI: "https://diffpal.io",
						Rules:          buildRules(bundle.Findings),
					},
				},
			},
		},
	}
	r.Runs[0].Results = make([]Result, 0, len(bundle.Findings))
	for _, f := range bundle.Findings {
		r.Runs[0].Results = append(r.Runs[0].Results, Result{
			Rule:    f.Category,
			Level:   sarifLevel(f.Severity),
			Message: Message{Text: resultMessage(f)},
			Locations: []Location{
				{
					PhysicalLocation: PhysicalLocation{
						ArtifactLocation: ArtifactLocation{URI: f.Path},
						Region:           Region{StartLine: f.StartLine, EndLine: f.EndLine},
					},
				},
			},
			PartialFingerprints: map[string]string{
				"diffpalFingerprint": f.ID,
			},
			Properties: ResultProperties{
				Category:   f.Category,
				Confidence: f.Confidence,
				Blocking:   f.Blocking,
			},
		})
	}
	return r
}

func ToJSON(report Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func sarifLevel(sev string) string {
	switch sev {
	case "critical", "high":
		return "error"
	case "medium":
		return "warning"
	default:
		return "note"
	}
}

func buildRules(findingsList []findings.Finding) []Rule {
	if len(findingsList) == 0 {
		return nil
	}
	byID := make(map[string]Rule, len(findingsList))
	for _, finding := range findingsList {
		if _, ok := byID[finding.Category]; ok {
			continue
		}
		byID[finding.Category] = Rule{
			ID:               finding.Category,
			Name:             finding.Category,
			ShortDescription: Message{Text: "DiffPal " + finding.Category + " finding"},
			Properties: RuleProperties{
				Category: finding.Category,
			},
		}
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	rules := make([]Rule, 0, len(ids))
	for _, id := range ids {
		rules = append(rules, byID[id])
	}
	return rules
}

func resultMessage(f findings.Finding) string {
	if f.Title == "" {
		return f.Message
	}
	if f.Message == "" || f.Message == f.Title {
		return f.Title
	}
	return f.Title + ": " + f.Message
}
