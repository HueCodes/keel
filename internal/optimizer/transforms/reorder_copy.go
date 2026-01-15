package transforms

import (
	"strings"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

// ReorderCopyTransform moves broad COPY instructions after dependency install RUNs
// for better Docker cache utilization
type ReorderCopyTransform struct {
	// DryRun if true, don't actually modify - just check if changes would be made
	DryRun bool
}

func (t *ReorderCopyTransform) Name() string {
	return "reorder-copy"
}

func (t *ReorderCopyTransform) Description() string {
	return "Reorder COPY to come after RUN install commands for better caching"
}

func (t *ReorderCopyTransform) Rules() []string {
	return []string{"PERF001"}
}

func (t *ReorderCopyTransform) Transform(df *parser.Dockerfile, diags []analyzer.Diagnostic) bool {
	changed := false

	for _, stage := range df.Stages {
		if t.reorderStage(stage) {
			changed = true
		}
	}

	return changed
}

// reorderStage reorders instructions within a single stage
func (t *ReorderCopyTransform) reorderStage(stage *parser.Stage) bool {
	// Find the pattern: broad COPY before dependency install
	// We want: dependency file COPY -> RUN install -> broad COPY

	analysis := analyzeStage(stage.Instructions)
	if analysis == nil {
		return false
	}

	// Only reorder if broad COPY comes before the dependency install
	if analysis.broadCopyIdx >= analysis.depInstallIdx {
		return false
	}

	if t.DryRun {
		return true // Would make changes
	}

	// Perform the reorder
	instructions := stage.Instructions
	broadCopy := instructions[analysis.broadCopyIdx]

	// Remove broad COPY from its current position
	newInstructions := make([]parser.Instruction, 0, len(instructions))
	for i, inst := range instructions {
		if i != analysis.broadCopyIdx {
			newInstructions = append(newInstructions, inst)
		}
	}

	// Find where to insert (after the dependency install, accounting for removal)
	insertIdx := analysis.depInstallIdx
	if analysis.broadCopyIdx < analysis.depInstallIdx {
		insertIdx-- // Adjust for removal
	}

	// Insert after the dependency install
	if insertIdx >= len(newInstructions) {
		newInstructions = append(newInstructions, broadCopy)
	} else {
		// Insert after depInstallIdx
		insertIdx++ // Insert after, not at
		newInstructions = append(newInstructions[:insertIdx], append([]parser.Instruction{broadCopy}, newInstructions[insertIdx:]...)...)
	}

	stage.Instructions = newInstructions
	return true
}

// stageAnalysis holds the analysis of a stage's instruction order
type stageAnalysis struct {
	broadCopyIdx  int // Index of broad COPY instruction
	depInstallIdx int // Index of dependency install RUN
}

// analyzeStage analyzes a stage to find reordering opportunities
func analyzeStage(instructions []parser.Instruction) *stageAnalysis {
	broadCopyIdx := -1
	depInstallIdx := -1

	for i, inst := range instructions {
		switch v := inst.(type) {
		case *parser.CopyInstruction:
			// Only consider first broad COPY
			if broadCopyIdx == -1 && isBroadCopyInstruction(v) {
				broadCopyIdx = i
			}
		case *parser.AddInstruction:
			// ADD with broad sources also counts
			if broadCopyIdx == -1 && isBroadAddInstruction(v) {
				broadCopyIdx = i
			}
		case *parser.RunInstruction:
			// Look for dependency install commands
			if isDependencyInstallCommand(v.Command) {
				depInstallIdx = i
				break // Stop at first dependency install
			}
		}
	}

	// Need both a broad copy AND a dependency install to reorder
	if broadCopyIdx == -1 || depInstallIdx == -1 {
		return nil
	}

	return &stageAnalysis{
		broadCopyIdx:  broadCopyIdx,
		depInstallIdx: depInstallIdx,
	}
}

// isBroadCopyInstruction checks if a COPY copies broadly (like COPY . .)
func isBroadCopyInstruction(copy *parser.CopyInstruction) bool {
	// Skip COPY --from (multi-stage copies)
	if copy.From != "" {
		return false
	}

	for _, src := range copy.Sources {
		if isBroadSource(src) {
			return true
		}
	}
	return false
}

// isBroadAddInstruction checks if an ADD instruction is broad
func isBroadAddInstruction(add *parser.AddInstruction) bool {
	for _, src := range add.Sources {
		if isBroadSource(src) {
			return true
		}
	}
	return false
}

// isBroadSource checks if a source is a broad pattern
func isBroadSource(src string) bool {
	// Patterns that copy "everything"
	broadPatterns := []string{
		".",
		"./",
		"*",
		"./*",
	}
	for _, p := range broadPatterns {
		if src == p {
			return true
		}
	}
	return false
}

// isDependencyInstallCommand checks if a command is a dependency installation
func isDependencyInstallCommand(cmd string) bool {
	// Common dependency installation patterns
	installPatterns := []string{
		// Node.js
		"npm install", "npm ci", "yarn install", "yarn add", "pnpm install",
		// Python
		"pip install", "pip3 install", "pipenv install", "poetry install",
		// Go
		"go mod download", "go get",
		// Ruby
		"bundle install", "gem install",
		// PHP
		"composer install",
		// Rust
		"cargo fetch", "cargo build",
		// System packages (less common as first install, but included)
		"apt-get install", "apt install", "apk add", "yum install", "dnf install",
	}

	cmdLower := strings.ToLower(cmd)
	for _, pattern := range installPatterns {
		if strings.Contains(cmdLower, pattern) {
			return true
		}
	}
	return false
}
