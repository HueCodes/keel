package style

import (
	"strings"
	"unicode"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/lexer"
	"github.com/HueCodes/keel/internal/parser"
)

// STY001InstructionCase checks for inconsistent instruction casing
type STY001InstructionCase struct{}

func (r *STY001InstructionCase) ID() string          { return "STY001" }
func (r *STY001InstructionCase) Name() string        { return "instruction-case" }
func (r *STY001InstructionCase) Category() analyzer.Category { return analyzer.CategoryStyle }
func (r *STY001InstructionCase) Severity() analyzer.Severity { return analyzer.SeverityHint }

func (r *STY001InstructionCase) Description() string {
	return "Dockerfile instructions should be uppercase for consistency."
}

func (r *STY001InstructionCase) Check(df *parser.Dockerfile, ctx *analyzer.RuleContext) []analyzer.Diagnostic {
	var diags []analyzer.Diagnostic

	// Check each line for instruction casing
	for lineNum, line := range ctx.SourceLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Extract the first word (instruction)
		parts := strings.Fields(trimmed)
		if len(parts) == 0 {
			continue
		}

		instruction := parts[0]
		if !isDockerInstruction(strings.ToUpper(instruction)) {
			continue
		}

		// Check if not uppercase
		if instruction != strings.ToUpper(instruction) {
			diag := analyzer.NewDiagnostic(r.ID(), r.Category()).
				WithSeverity(r.Severity()).
				WithMessagef("Instruction '%s' should be uppercase: '%s'", instruction, strings.ToUpper(instruction)).
				WithPos(lexer.Position{Line: lineNum + 1, Column: 1}).
				WithContext(line).
				WithHelp("Use uppercase for Dockerfile instructions: " + strings.ToUpper(instruction)).
				WithFix(strings.ToUpper(instruction)).
				Build()
			diags = append(diags, diag)
		}
	}

	return diags
}

func isDockerInstruction(s string) bool {
	instructions := map[string]bool{
		"FROM": true, "RUN": true, "CMD": true, "LABEL": true,
		"MAINTAINER": true, "EXPOSE": true, "ENV": true, "ADD": true,
		"COPY": true, "ENTRYPOINT": true, "VOLUME": true, "USER": true,
		"WORKDIR": true, "ARG": true, "ONBUILD": true, "STOPSIGNAL": true,
		"HEALTHCHECK": true, "SHELL": true,
	}
	return instructions[s]
}

type Position = lexer.Position

// Check if string has any lowercase letters
func hasLowercase(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) {
			return true
		}
	}
	return false
}

func init() {
	Register(&STY001InstructionCase{})
}
