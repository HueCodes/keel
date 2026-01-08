package optimizer

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// Transform is the interface for AST transformations
type Transform interface {
	// Name returns the transform name
	Name() string

	// Description returns what this transform does
	Description() string

	// Rules returns the rule IDs this transform can fix
	Rules() []string

	// Transform applies the transformation to the AST
	// Returns true if any changes were made
	Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool
}

// Optimizer applies transforms to fix Dockerfile issues
type Optimizer struct {
	transforms []Transform
	dryRun     bool
}

// Option configures an Optimizer
type Option func(*Optimizer)

// New creates a new Optimizer
func New(opts ...Option) *Optimizer {
	o := &Optimizer{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithTransforms adds transforms to the optimizer
func WithTransforms(transforms ...Transform) Option {
	return func(o *Optimizer) {
		o.transforms = append(o.transforms, transforms...)
	}
}

// WithDryRun enables dry-run mode (no actual changes)
func WithDryRun(dryRun bool) Option {
	return func(o *Optimizer) {
		o.dryRun = dryRun
	}
}

// Optimize applies all relevant transforms to fix diagnostics
func (o *Optimizer) Optimize(df *parser.Dockerfile, diags []analyzer.Diagnostic) *Result {
	result := &Result{
		Original:   df,
		Optimized:  df, // Will be modified in place
		ChangesMade: []Change{},
	}

	// Build a map of rule IDs that have diagnostics
	// We apply transforms regardless of Fixable flag - if a transform exists, we apply it
	ruleIDs := make(map[string]bool)
	for _, d := range diags {
		ruleIDs[d.Rule] = true
	}

	// Apply each transform that handles a triggered rule
	for _, transform := range o.transforms {
		// Check if this transform handles any of our diagnostics
		shouldApply := false
		for _, ruleID := range transform.Rules() {
			if ruleIDs[ruleID] {
				shouldApply = true
				break
			}
		}

		if !shouldApply {
			continue
		}

		if o.dryRun {
			result.ChangesMade = append(result.ChangesMade, Change{
				Transform:   transform.Name(),
				Description: transform.Description(),
				Applied:     false,
			})
			continue
		}

		if transform.Transform(df, diags) {
			result.ChangesMade = append(result.ChangesMade, Change{
				Transform:   transform.Name(),
				Description: transform.Description(),
				Applied:     true,
			})
		}
	}

	return result
}

// Result holds the optimization result
type Result struct {
	Original    *parser.Dockerfile
	Optimized   *parser.Dockerfile
	ChangesMade []Change
}

// HasChanges returns true if any changes were made
func (r *Result) HasChanges() bool {
	for _, c := range r.ChangesMade {
		if c.Applied {
			return true
		}
	}
	return false
}

// Change represents a single optimization change
type Change struct {
	Transform   string
	Description string
	Applied     bool
}

// AllTransforms returns all available transforms
func AllTransforms() []Transform {
	return []Transform{
		&MergeRun{},
		&AddCacheCleanup{},
		&AddNoInstallRecommends{},
	}
}

// MergeRun merges consecutive RUN instructions
type MergeRun struct{}

func (t *MergeRun) Name() string        { return "merge-run" }
func (t *MergeRun) Description() string { return "Merge consecutive RUN instructions to reduce layers" }
func (t *MergeRun) Rules() []string     { return []string{"PERF004"} }

func (t *MergeRun) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
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

func canMergeRun(run *parser.RunInstruction) bool {
	if run.Heredoc != nil || run.IsExec || run.Mount != "" {
		return false
	}
	return true
}

func mergeRuns(runs []*parser.RunInstruction) *parser.RunInstruction {
	if len(runs) == 0 {
		return nil
	}
	if len(runs) == 1 {
		return runs[0]
	}

	var commands []string
	for _, run := range runs {
		cmd := strings.TrimSpace(run.Command)
		if cmd != "" {
			commands = append(commands, cmd)
		}
	}

	return &parser.RunInstruction{
		BaseInstruction: parser.BaseInstruction{
			StartPos: runs[0].Pos(),
			EndPos:   runs[len(runs)-1].End(),
		},
		Command: strings.Join(commands, " && "),
	}
}

// AddCacheCleanup adds package manager cache cleanup
type AddCacheCleanup struct{}

func (t *AddCacheCleanup) Name() string        { return "add-cache-cleanup" }
func (t *AddCacheCleanup) Description() string { return "Add package manager cache cleanup" }
func (t *AddCacheCleanup) Rules() []string     { return []string{"PERF003"} }

func (t *AddCacheCleanup) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	changed := false

	cleanups := map[string]string{
		"apt-get install": " && rm -rf /var/lib/apt/lists/*",
		"apt install":     " && rm -rf /var/lib/apt/lists/*",
		"yum install":     " && yum clean all",
		"dnf install":     " && dnf clean all",
	}

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			run, ok := inst.(*parser.RunInstruction)
			if !ok || run.Heredoc != nil || run.IsExec {
				continue
			}

			// Handle apk specially
			if strings.Contains(run.Command, "apk add") && !strings.Contains(run.Command, "--no-cache") {
				run.Command = strings.Replace(run.Command, "apk add", "apk add --no-cache", 1)
				changed = true
			}

			// Handle other package managers
			for pattern, cleanup := range cleanups {
				if strings.Contains(run.Command, pattern) && !hasCleanup(run.Command) {
					run.Command = strings.TrimSpace(run.Command) + cleanup
					changed = true
					break
				}
			}
		}
	}

	return changed
}

func hasCleanup(cmd string) bool {
	cleanupPatterns := []string{
		"rm -rf /var/lib/apt/lists",
		"apt-get clean",
		"yum clean all",
		"dnf clean all",
	}
	for _, p := range cleanupPatterns {
		if strings.Contains(cmd, p) {
			return true
		}
	}
	return false
}

// AddNoInstallRecommends adds --no-install-recommends
type AddNoInstallRecommends struct{}

func (t *AddNoInstallRecommends) Name() string        { return "add-no-install-recommends" }
func (t *AddNoInstallRecommends) Description() string { return "Add --no-install-recommends to apt" }
func (t *AddNoInstallRecommends) Rules() []string     { return []string{"PERF005"} }

func (t *AddNoInstallRecommends) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	changed := false

	for _, stage := range df.Stages {
		for _, inst := range stage.Instructions {
			run, ok := inst.(*parser.RunInstruction)
			if !ok || run.Heredoc != nil || run.IsExec {
				continue
			}

			if strings.Contains(run.Command, "apt-get install") && !strings.Contains(run.Command, "--no-install-recommends") {
				run.Command = strings.Replace(run.Command, "apt-get install", "apt-get install --no-install-recommends", 1)
				changed = true
			}
			if strings.Contains(run.Command, "apt install") && !strings.Contains(run.Command, "--no-install-recommends") {
				run.Command = strings.Replace(run.Command, "apt install", "apt install --no-install-recommends", 1)
				changed = true
			}
		}
	}

	return changed
}
