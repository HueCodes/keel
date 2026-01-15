package transforms

import (
	"testing"

	"github.com/HueCodes/keel/internal/parser"
)

func TestReorderCopyTransform_Name(t *testing.T) {
	tr := &ReorderCopyTransform{}
	if tr.Name() != "reorder-copy" {
		t.Errorf("expected name 'reorder-copy', got %s", tr.Name())
	}
}

func TestReorderCopyTransform_Rules(t *testing.T) {
	tr := &ReorderCopyTransform{}
	rules := tr.Rules()
	if len(rules) != 1 || rules[0] != "PERF001" {
		t.Errorf("expected rules ['PERF001'], got %v", rules)
	}
}

func TestReorderCopyTransform_BroadCopyBeforeInstall(t *testing.T) {
	// BAD: COPY . . before npm install
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.CopyInstruction{
						Sources:     []string{"."},
						Destination: "/app",
					},
					&parser.RunInstruction{
						Command: "npm install",
					},
					&parser.RunInstruction{
						Command: "npm run build",
					},
				},
			},
		},
	}

	tr := &ReorderCopyTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	// Verify reordering: RUN npm install should come before COPY .
	instructions := df.Stages[0].Instructions
	if len(instructions) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(instructions))
	}

	// First should now be the npm install
	run1, ok := instructions[0].(*parser.RunInstruction)
	if !ok {
		t.Fatal("first instruction should be RunInstruction")
	}
	if run1.Command != "npm install" {
		t.Errorf("expected 'npm install', got '%s'", run1.Command)
	}

	// Second should be the broad COPY
	copy, ok := instructions[1].(*parser.CopyInstruction)
	if !ok {
		t.Fatal("second instruction should be CopyInstruction")
	}
	if len(copy.Sources) != 1 || copy.Sources[0] != "." {
		t.Errorf("expected sources ['.'], got %v", copy.Sources)
	}
}

func TestReorderCopyTransform_AlreadyOptimal(t *testing.T) {
	// GOOD: npm install before COPY .
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.CopyInstruction{
						Sources:     []string{"package.json"},
						Destination: "/app/",
					},
					&parser.RunInstruction{
						Command: "npm install",
					},
					&parser.CopyInstruction{
						Sources:     []string{"."},
						Destination: "/app",
					},
				},
			},
		},
	}

	tr := &ReorderCopyTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected no changes for optimal order")
	}
}

func TestReorderCopyTransform_NoDependencyInstall(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.CopyInstruction{
						Sources:     []string{"."},
						Destination: "/app",
					},
					&parser.RunInstruction{
						Command: "chmod +x /app/run.sh",
					},
				},
			},
		},
	}

	tr := &ReorderCopyTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected no changes without dependency install")
	}
}

func TestReorderCopyTransform_NoBroadCopy(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.CopyInstruction{
						Sources:     []string{"package.json"},
						Destination: "/app/",
					},
					&parser.RunInstruction{
						Command: "npm install",
					},
				},
			},
		},
	}

	tr := &ReorderCopyTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected no changes without broad COPY")
	}
}

func TestReorderCopyTransform_CopyFromStage(t *testing.T) {
	// COPY --from should not be reordered
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.CopyInstruction{
						Sources:     []string{"."},
						Destination: "/app",
						From:        "builder",
					},
					&parser.RunInstruction{
						Command: "npm install",
					},
				},
			},
		},
	}

	tr := &ReorderCopyTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected no changes for COPY --from")
	}
}

func TestReorderCopyTransform_MultipleStages(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.CopyInstruction{
						Sources:     []string{"."},
						Destination: "/app",
					},
					&parser.RunInstruction{
						Command: "npm install",
					},
				},
			},
			{
				Instructions: []parser.Instruction{
					&parser.CopyInstruction{
						Sources:     []string{"."},
						Destination: "/app",
					},
					&parser.RunInstruction{
						Command: "pip install -r requirements.txt",
					},
				},
			},
		},
	}

	tr := &ReorderCopyTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	// Both stages should be reordered
	for i, stage := range df.Stages {
		_, ok := stage.Instructions[0].(*parser.RunInstruction)
		if !ok {
			t.Errorf("stage %d: first instruction should be RunInstruction", i)
		}
	}
}

func TestReorderCopyTransform_DryRun(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.CopyInstruction{
						Sources:     []string{"."},
						Destination: "/app",
					},
					&parser.RunInstruction{
						Command: "npm install",
					},
				},
			},
		},
	}

	tr := &ReorderCopyTransform{DryRun: true}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected dry-run to report would-change")
	}

	// But instructions should NOT be reordered
	_, ok := df.Stages[0].Instructions[0].(*parser.CopyInstruction)
	if !ok {
		t.Error("dry-run should not modify instructions")
	}
}

func TestReorderCopyTransform_VariousBroadPatterns(t *testing.T) {
	patterns := []string{".", "./", "*", "./*"}

	for _, pattern := range patterns {
		t.Run(pattern, func(t *testing.T) {
			df := &parser.Dockerfile{
				Stages: []*parser.Stage{
					{
						Instructions: []parser.Instruction{
							&parser.CopyInstruction{
								Sources:     []string{pattern},
								Destination: "/app",
							},
							&parser.RunInstruction{
								Command: "npm install",
							},
						},
					},
				},
			}

			tr := &ReorderCopyTransform{}
			changed := tr.Transform(df, nil)

			if !changed {
				t.Errorf("expected pattern '%s' to be detected as broad", pattern)
			}
		})
	}
}

func TestReorderCopyTransform_VariousInstallCommands(t *testing.T) {
	installCommands := []string{
		"npm install",
		"npm ci",
		"yarn install",
		"pip install -r requirements.txt",
		"pip3 install flask",
		"go mod download",
		"bundle install",
		"composer install",
		"cargo fetch",
	}

	for _, cmd := range installCommands {
		t.Run(cmd, func(t *testing.T) {
			df := &parser.Dockerfile{
				Stages: []*parser.Stage{
					{
						Instructions: []parser.Instruction{
							&parser.CopyInstruction{
								Sources:     []string{"."},
								Destination: "/app",
							},
							&parser.RunInstruction{
								Command: cmd,
							},
						},
					},
				},
			}

			tr := &ReorderCopyTransform{}
			changed := tr.Transform(df, nil)

			if !changed {
				t.Errorf("expected command '%s' to be detected as install", cmd)
			}
		})
	}
}

func TestReorderCopyTransform_PreservesOtherInstructions(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.EnvInstruction{
						Variables: []parser.KeyValue{{Key: "NODE_ENV", Value: "production"}},
					},
					&parser.CopyInstruction{
						Sources:     []string{"."},
						Destination: "/app",
					},
					&parser.RunInstruction{
						Command: "npm install",
					},
					&parser.RunInstruction{
						Command: "npm run build",
					},
					&parser.CmdInstruction{
						Arguments: []string{"node", "app.js"},
						IsExec:    true,
					},
				},
			},
		},
	}

	tr := &ReorderCopyTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	instructions := df.Stages[0].Instructions
	if len(instructions) != 5 {
		t.Fatalf("expected 5 instructions, got %d", len(instructions))
	}

	// ENV should still be first
	_, ok := instructions[0].(*parser.EnvInstruction)
	if !ok {
		t.Error("ENV should remain first")
	}

	// npm install should be second
	run1, ok := instructions[1].(*parser.RunInstruction)
	if !ok || run1.Command != "npm install" {
		t.Error("npm install should be second")
	}

	// COPY should be third
	_, ok = instructions[2].(*parser.CopyInstruction)
	if !ok {
		t.Error("COPY should be third")
	}

	// npm run build should be fourth
	run2, ok := instructions[3].(*parser.RunInstruction)
	if !ok || run2.Command != "npm run build" {
		t.Error("npm run build should be fourth")
	}

	// CMD should still be last
	_, ok = instructions[4].(*parser.CmdInstruction)
	if !ok {
		t.Error("CMD should remain last")
	}
}

func TestIsBroadSource(t *testing.T) {
	tests := []struct {
		source   string
		expected bool
	}{
		{".", true},
		{"./", true},
		{"*", true},
		{"./*", true},
		{"package.json", false},
		{"src/", false},
		{"*.js", false},
		{"..", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			result := isBroadSource(tt.source)
			if result != tt.expected {
				t.Errorf("isBroadSource(%q) = %v, want %v", tt.source, result, tt.expected)
			}
		})
	}
}

func TestIsDependencyInstallCommand(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"npm install", true},
		{"npm ci", true},
		{"yarn install", true},
		{"pip install flask", true},
		{"go mod download", true},
		{"bundle install", true},
		{"composer install", true},
		{"cargo fetch", true},
		{"apt-get install -y curl", true},
		{"npm run build", false},
		{"go build", false},
		{"python app.py", false},
		{"chmod +x script.sh", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			result := isDependencyInstallCommand(tt.cmd)
			if result != tt.expected {
				t.Errorf("isDependencyInstallCommand(%q) = %v, want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}
