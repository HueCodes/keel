package transforms

import (
	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// FixInstructionCaseTransform fixes instruction casing
// Note: This transform works on source lines, not AST
// It's a placeholder - actual fix happens in the rewriter
type FixInstructionCaseTransform struct{}

func (t *FixInstructionCaseTransform) Name() string {
	return "fix-instruction-case"
}

func (t *FixInstructionCaseTransform) Description() string {
	return "Convert instructions to uppercase"
}

func (t *FixInstructionCaseTransform) Rules() []string {
	return []string{"STY001"}
}

func (t *FixInstructionCaseTransform) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	// This transform is handled specially in the rewriter
	// since it needs to operate on source text, not AST
	// Return true to indicate we want to fix these
	for _, d := range diags {
		if d.Rule == "STY001" {
			return true
		}
	}
	return false
}
