package bestpractice

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// BP002AddVsCopy checks for ADD usage when COPY would suffice
type BP002AddVsCopy struct{}

func (r *BP002AddVsCopy) ID() string          { return "BP002" }
func (r *BP002AddVsCopy) Name() string        { return "add-vs-copy" }
func (r *BP002AddVsCopy) Category() analyzer.Category { return analyzer.CategoryBestPractice }
func (r *BP002AddVsCopy) Severity() analyzer.Severity { return analyzer.SeverityWarning }

func (r *BP002AddVsCopy) Description() string {
	return "COPY is preferred over ADD for copying local files. ADD has extra features that can be confusing."
}

func (r *BP002AddVsCopy) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			add, ok := inst.(*parser.AddInstruction)
			if !ok {
				continue
			}

			// ADD is acceptable for:
			// 1. Remote URLs (though we warn about this in SEC007)
			// 2. Tar file auto-extraction

			needsAdd := false
			for _, src := range add.Sources {
				// Check for URL
				if isURL(src) {
					needsAdd = true
					break
				}
				// Check for tar file (ADD auto-extracts)
				if isTarFile(src) {
					needsAdd = true
					break
				}
			}

			if !needsAdd {
				diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
					WithSeverity(r.Severity()).
					WithMessage("ADD is used where COPY would suffice").
					WithPos(add.Pos()).
					WithContext(ctx.GetLine(add.Pos().Line)).
					WithHelp("Use COPY for simple file copies. ADD should only be used for URLs or tar extraction.").
					WithFix("COPY").
					Build()
				diags = append(diags, diag)
			}
		}
	}

	return diags
}

func isURL(s string) bool {
	return len(s) > 7 && (s[:7] == "http://" || s[:8] == "https://" || s[:6] == "ftp://")
}

func isTarFile(s string) bool {
	suffixes := []string{".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tar.xz", ".txz"}
	for _, suffix := range suffixes {
		if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}

func init() {
	Register(&BP002AddVsCopy{})
}
