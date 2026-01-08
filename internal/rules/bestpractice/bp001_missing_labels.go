package bestpractice

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// BP001MissingLabels checks for missing important labels
type BP001MissingLabels struct{}

func (r *BP001MissingLabels) ID() string          { return "BP001" }
func (r *BP001MissingLabels) Name() string        { return "missing-labels" }
func (r *BP001MissingLabels) Category() analyzer.Category { return analyzer.CategoryBestPractice }
func (r *BP001MissingLabels) Severity() analyzer.Severity { return analyzer.SeverityInfo }

func (r *BP001MissingLabels) Description() string {
	return "Images should have maintainer, version, and description labels for documentation."
}

var recommendedLabels = []string{
	"maintainer",
	"version",
	"description",
}

// OCI label equivalents
var ociLabelMapping = map[string][]string{
	"maintainer":  {"org.opencontainers.image.authors", "maintainer"},
	"version":     {"org.opencontainers.image.version", "version"},
	"description": {"org.opencontainers.image.description", "description"},
}

func (r *BP001MissingLabels) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	if len(df.Stages) == 0 {
		return diags
	}

	// Only check final stage
	finalStage := df.Stages[len(df.Stages)-1]

	// Collect all labels
	labels := make(map[string]bool)
	for _, inst := range finalStage.Instructions {
		label, ok := inst.(*parser.LabelInstruction)
		if !ok {
			continue
		}
		for _, kv := range label.Labels {
			labels[strings.ToLower(kv.Key)] = true
		}
	}

	// Check for missing labels
	var missing []string
	for _, rec := range recommendedLabels {
		found := false
		for _, variant := range ociLabelMapping[rec] {
			if labels[strings.ToLower(variant)] {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, rec)
		}
	}

	if len(missing) > 0 {
		diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
			WithSeverity(r.Severity()).
			WithMessagef("Missing recommended labels: %s", strings.Join(missing, ", ")).
			WithPos(finalStage.From.Pos()).
			WithHelp("Add LABEL instructions, e.g., LABEL maintainer=\"you@example.com\" version=\"1.0\" description=\"My app\"").
			Build()
		diags = append(diags, diag)
	}

	return diags
}

func init() {
	Register(&BP001MissingLabels{})
}
