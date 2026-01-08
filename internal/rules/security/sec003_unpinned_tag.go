package security

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// SEC003UnpinnedTag checks for images using 'latest' or no tag
type SEC003UnpinnedTag struct{}

func (r *SEC003UnpinnedTag) ID() string          { return "SEC003" }
func (r *SEC003UnpinnedTag) Name() string        { return "unpinned-image-tag" }
func (r *SEC003UnpinnedTag) Category() analyzer.Category { return analyzer.CategorySecurity }
func (r *SEC003UnpinnedTag) Severity() analyzer.Severity { return analyzer.SeverityError }

func (r *SEC003UnpinnedTag) Description() string {
	return "Base image uses unpinned tag. Using 'latest' or no tag can lead to unpredictable builds."
}

// Common images that are okay to use with latest or no tag
var trustedImages = map[string]bool{
	"scratch": true,
}

func (r *SEC003UnpinnedTag) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		from := stage.From
		if from == nil {
			continue
		}

		// Skip scratch
		if trustedImages[from.Image] {
			continue
		}

		// Skip if using digest
		if from.Digest != "" {
			continue
		}

		// Skip if image is a variable
		if strings.HasPrefix(from.Image, "$") {
			continue
		}

		// Check for missing or 'latest' tag
		if from.Tag == "" || from.Tag == "latest" {
			var msg string
			if from.Tag == "" {
				msg = "Base image has no tag (implicitly uses 'latest')"
			} else {
				msg = "Base image uses 'latest' tag"
			}

			diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
				WithSeverity(r.Severity()).
				WithMessage(msg).
				WithPos(from.Pos()).
				WithContext(ctx.GetLine(from.Pos().Line)).
				WithHelp("Pin to a specific version for reproducible builds, e.g., " + from.Image + ":22.04 or use a digest").
				Build()
			diags = append(diags, diag)
		}
	}

	return diags
}

func init() {
	Register(&SEC003UnpinnedTag{})
}
