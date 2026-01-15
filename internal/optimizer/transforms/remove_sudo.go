package transforms

import (
	"regexp"
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// RemoveSudoTransform removes sudo from RUN commands
type RemoveSudoTransform struct{}

func (t *RemoveSudoTransform) Name() string {
	return "remove-sudo"
}

func (t *RemoveSudoTransform) Description() string {
	return "Remove sudo from RUN commands (unnecessary in Docker)"
}

func (t *RemoveSudoTransform) Rules() []string {
	return []string{"SEC005"}
}

// Regex patterns for sudo removal
// These match sudo with various common flags
var sudoPatterns = []*regexp.Regexp{
	// sudo with common flags that don't change user
	regexp.MustCompile(`\bsudo\s+(?:-[EHnPS]\s+)*`),
	// sudo -E (preserve environment)
	regexp.MustCompile(`\bsudo\s+-E\s+`),
	// sudo -n (non-interactive)
	regexp.MustCompile(`\bsudo\s+-n\s+`),
	// sudo alone
	regexp.MustCompile(`\bsudo\s+`),
}

// sudoUserPattern matches sudo -u which changes user and should NOT be auto-fixed
var sudoUserPattern = regexp.MustCompile(`\bsudo\s+(-\w+\s+)*-u\s+`)

func (t *RemoveSudoTransform) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	changed := false

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			run, ok := inst.(*parser.RunInstruction)
			if !ok {
				continue
			}

			// Handle shell form
			if !run.IsExec && run.Heredoc == nil {
				newCmd := removeSudo(run.Command, &changed)
				if newCmd != run.Command {
					run.Command = newCmd
				}
			}

			// Handle heredoc content
			if run.Heredoc != nil {
				newContent := removeSudo(run.Heredoc.Content, &changed)
				if newContent != run.Heredoc.Content {
					run.Heredoc.Content = newContent
				}
			}
		}
	}

	return changed
}

func removeSudo(cmd string, changed *bool) string {
	// Skip if using sudo -u (changing user) - this needs USER instruction instead
	if sudoUserPattern.MatchString(cmd) {
		return cmd
	}

	original := cmd

	// Apply patterns in order from most specific to least specific
	for _, pattern := range sudoPatterns {
		if pattern.MatchString(cmd) {
			cmd = pattern.ReplaceAllString(cmd, "")
		}
	}

	// Clean up any double spaces that may have been introduced
	cmd = strings.Join(strings.Fields(cmd), " ")

	if cmd != original {
		*changed = true
	}

	return cmd
}
