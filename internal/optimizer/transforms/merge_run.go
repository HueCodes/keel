package transforms

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// MergeRunTransform merges consecutive RUN instructions
type MergeRunTransform struct{}

func (t *MergeRunTransform) Name() string {
	return "merge-run"
}

func (t *MergeRunTransform) Description() string {
	return "Merge consecutive RUN instructions to reduce layers"
}

func (t *MergeRunTransform) Rules() []string {
	return []string{"PERF004"}
}

func (t *MergeRunTransform) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	changed := false

	for _, stage := range df.Stages {
		stage.Instructions = mergeConsecutiveRuns(stage.Instructions, &changed)
	}

	return changed
}

func mergeConsecutiveRuns(instructions []parser.Instruction, changed *bool) []parser.Instruction {
	if len(instructions) < 2 {
		return instructions
	}

	var result []parser.Instruction
	var runGroup []*parser.RunInstruction

	flushRunGroup := func() {
		if len(runGroup) == 0 {
			return
		}
		if len(runGroup) == 1 {
			result = append(result, runGroup[0])
		} else {
			// Merge multiple RUN instructions
			merged := mergeRuns(runGroup)
			result = append(result, merged)
			*changed = true
		}
		runGroup = nil
	}

	for _, inst := range instructions {
		run, isRun := inst.(*parser.RunInstruction)
		if isRun && canMergeRun(run) {
			runGroup = append(runGroup, run)
		} else {
			flushRunGroup()
			result = append(result, inst)
		}
	}
	flushRunGroup()

	return result
}

// canMergeRun returns true if this RUN can be merged with others
func canMergeRun(run *parser.RunInstruction) bool {
	// Don't merge heredocs
	if run.Heredoc != nil {
		return false
	}
	// Don't merge exec form
	if run.IsExec {
		return false
	}
	// Don't merge if has special mounts
	if run.Mount != "" {
		return false
	}
	return true
}

// mergeRuns combines multiple RUN instructions into one
func mergeRuns(runs []*parser.RunInstruction) *parser.RunInstruction {
	if len(runs) == 0 {
		return nil
	}
	if len(runs) == 1 {
		return runs[0]
	}

	// Collect all commands
	var commands []string
	for _, run := range runs {
		cmd := strings.TrimSpace(run.Command)
		if cmd != "" {
			commands = append(commands, cmd)
		}
	}

	// Join with && and proper formatting
	merged := &parser.RunInstruction{
		BaseInstruction: parser.BaseInstruction{
			StartPos: runs[0].Pos(),
			EndPos:   runs[len(runs)-1].End(),
		},
		Command: strings.Join(commands, " \\\n    && "),
	}

	return merged
}
