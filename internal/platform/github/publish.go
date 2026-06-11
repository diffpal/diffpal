package github

import (
	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/markdown"
)

const maxAnnotationsPerBatch = 50

type Annotation struct {
	Path            string `json:"path"`
	StartLine       int    `json:"start_line"`
	EndLine         int    `json:"end_line"`
	Message         string `json:"message"`
	AnnotationLevel string `json:"annotation_level"`
}

type AnnotationBatch struct {
	Index       int          `json:"index"`
	Annotations []Annotation `json:"annotations"`
}

type CheckRunPayload struct {
	Name              string            `json:"name"`
	Status            string            `json:"status"`
	Conclusion        string            `json:"conclusion"`
	HeadSHA           string            `json:"head_sha"`
	Summary           string            `json:"summary"`
	Count             int               `json:"count"`
	Annotations       []Annotation      `json:"annotations"`
	AnnotationBatches []AnnotationBatch `json:"annotation_batches,omitempty"`
}

func BuildCheckRunPayload(ctx Context, bundle findings.FindingsBundle, statusSummary string) CheckRunPayload {
	levelBySeverity := map[string]string{
		"critical": "error",
		"high":     "error",
		"medium":   "warning",
		"low":      "note",
	}
	annotations := make([]Annotation, 0, len(bundle.Findings))
	blocking := 0
	for _, finding := range bundle.Findings {
		level, ok := levelBySeverity[finding.Severity]
		if !ok {
			level = "warning"
		}
		if finding.Blocking {
			blocking++
		}
		annotations = append(annotations, Annotation{
			Path:            finding.Path,
			StartLine:       finding.StartLine,
			EndLine:         finding.EndLine,
			Message:         annotationMessage(finding),
			AnnotationLevel: level,
		})
	}
	conclusion := "success"
	if blocking > 0 {
		conclusion = "failure"
	}
	batches := chunkAnnotations(annotations, maxAnnotationsPerBatch)
	primary := []Annotation{}
	if len(batches) > 0 {
		primary = batches[0].Annotations
	}
	return CheckRunPayload{
		Name:              "diffpal-checks",
		Status:            "completed",
		Conclusion:        conclusion,
		HeadSHA:           ctx.HeadSHA,
		Summary:           statusSummary,
		Count:             len(bundle.Findings),
		Annotations:       primary,
		AnnotationBatches: batches,
	}
}

func CheckRunSummary(bundle findings.FindingsBundle) string {
	return markdown.RenderSummary(bundle)
}

func annotationMessage(finding findings.Finding) string {
	if finding.Message != "" {
		return finding.Message
	}
	return finding.Title
}

func chunkAnnotations(items []Annotation, size int) []AnnotationBatch {
	if len(items) == 0 {
		return nil
	}
	if size <= 0 {
		size = len(items)
	}
	batches := make([]AnnotationBatch, 0, (len(items)+size-1)/size)
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, AnnotationBatch{
			Index:       len(batches),
			Annotations: items[start:end],
		})
	}
	return batches
}
