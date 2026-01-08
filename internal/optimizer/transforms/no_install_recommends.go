package transforms

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// AddNoInstallRecommendsTransform adds --no-install-recommends to apt-get install
type AddNoInstallRecommendsTransform struct{}

func (t *AddNoInstallRecommendsTransform) Name() string {
	return "add-no-install-recommends"
}

func (t *AddNoInstallRecommendsTransform) Description() string {
	return "Add --no-install-recommends to apt-get install to reduce image size"
}

func (t *AddNoInstallRecommendsTransform) Rules() []string {
	return []string{"PERF005"}
}

func (t *AddNoInstallRecommendsTransform) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	changed := false

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			run, ok := inst.(*parser.RunInstruction)
			if !ok {
				continue
			}

			if run.Heredoc != nil || run.IsExec {
				continue
			}

			newCmd := addNoInstallRecommends(run.Command, &changed)
			if newCmd != run.Command {
				run.Command = newCmd
			}
		}
	}

	return changed
}

func addNoInstallRecommends(cmd string, changed *bool) string {
	// Handle apt-get install
	if strings.Contains(cmd, "apt-get install") && !strings.Contains(cmd, "--no-install-recommends") {
		cmd = strings.Replace(cmd, "apt-get install", "apt-get install --no-install-recommends", 1)
		*changed = true
	}

	// Handle apt install
	if strings.Contains(cmd, "apt install") && !strings.Contains(cmd, "--no-install-recommends") {
		cmd = strings.Replace(cmd, "apt install", "apt install --no-install-recommends", 1)
		*changed = true
	}

	return cmd
}
