package transforms

import (
	"testing"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/parser"
)

func TestRemoveSudoTransform_Name(t *testing.T) {
	tr := &RemoveSudoTransform{}
	if tr.Name() != "remove-sudo" {
		t.Errorf("expected name 'remove-sudo', got %s", tr.Name())
	}
}

func TestRemoveSudoTransform_Rules(t *testing.T) {
	tr := &RemoveSudoTransform{}
	rules := tr.Rules()
	if len(rules) != 1 || rules[0] != "SEC005" {
		t.Errorf("expected rules ['SEC005'], got %v", rules)
	}
}

func TestRemoveSudoTransform_SimpleSudo(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Command: "sudo apt-get update",
					},
				},
			},
		},
	}

	tr := &RemoveSudoTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	run := df.Stages[0].Instructions[0].(*parser.RunInstruction)
	if run.Command != "apt-get update" {
		t.Errorf("expected 'apt-get update', got '%s'", run.Command)
	}
}

func TestRemoveSudoTransform_SudoWithFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sudo -E",
			input:    "sudo -E pip install foo",
			expected: "pip install foo",
		},
		{
			name:     "sudo -n",
			input:    "sudo -n apt-get install -y curl",
			expected: "apt-get install -y curl",
		},
		{
			name:     "sudo with multiple flags",
			input:    "sudo -E -n pip install foo",
			expected: "pip install foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := &parser.Dockerfile{
				Stages: []*parser.Stage{
					{
						Instructions: []parser.Instruction{
							&parser.RunInstruction{
								Command: tt.input,
							},
						},
					},
				},
			}

			tr := &RemoveSudoTransform{}
			tr.Transform(df, nil)

			run := df.Stages[0].Instructions[0].(*parser.RunInstruction)
			if run.Command != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, run.Command)
			}
		})
	}
}

func TestRemoveSudoTransform_SudoInChain(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Command: "apt-get update && sudo apt-get install -y curl",
					},
				},
			},
		},
	}

	tr := &RemoveSudoTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	run := df.Stages[0].Instructions[0].(*parser.RunInstruction)
	expected := "apt-get update && apt-get install -y curl"
	if run.Command != expected {
		t.Errorf("expected '%s', got '%s'", expected, run.Command)
	}
}

func TestRemoveSudoTransform_SudoUser(t *testing.T) {
	// sudo -u should NOT be transformed - it needs USER instruction instead
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Command: "sudo -u appuser npm install",
					},
				},
			},
		},
	}

	tr := &RemoveSudoTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected transform to NOT modify sudo -u")
	}

	run := df.Stages[0].Instructions[0].(*parser.RunInstruction)
	if run.Command != "sudo -u appuser npm install" {
		t.Errorf("expected command to be unchanged, got '%s'", run.Command)
	}
}

func TestRemoveSudoTransform_Heredoc(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Heredoc: &parser.Heredoc{
							Delimiter: "EOF",
							Content:   "sudo apt-get update\nsudo apt-get install -y curl",
						},
					},
				},
			},
		},
	}

	tr := &RemoveSudoTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	run := df.Stages[0].Instructions[0].(*parser.RunInstruction)
	expected := "apt-get update apt-get install -y curl"
	if run.Heredoc.Content != expected {
		t.Errorf("expected '%s', got '%s'", expected, run.Heredoc.Content)
	}
}

func TestRemoveSudoTransform_NoSudo(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Command: "apt-get update",
					},
				},
			},
		},
	}

	tr := &RemoveSudoTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected transform to report no changes")
	}
}

func TestRemoveSudoTransform_SudoSubstring(t *testing.T) {
	// Words like "pseudocode" should not match
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Command: "echo pseudocode",
					},
				},
			},
		},
	}

	tr := &RemoveSudoTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected transform to NOT modify 'pseudocode'")
	}

	run := df.Stages[0].Instructions[0].(*parser.RunInstruction)
	if run.Command != "echo pseudocode" {
		t.Errorf("expected command to be unchanged, got '%s'", run.Command)
	}
}

func TestRemoveSudoTransform_ExecForm(t *testing.T) {
	// Exec form should not be transformed
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						IsExec:    true,
						Arguments: []string{"sudo", "apt-get", "update"},
					},
				},
			},
		},
	}

	tr := &RemoveSudoTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected transform to NOT modify exec form")
	}
}

func TestRemoveSudoTransform_MultipleStages(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Command: "sudo apt-get update",
					},
				},
			},
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Command: "sudo npm install",
					},
				},
			},
		},
	}

	tr := &RemoveSudoTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	run1 := df.Stages[0].Instructions[0].(*parser.RunInstruction)
	if run1.Command != "apt-get update" {
		t.Errorf("stage 0: expected 'apt-get update', got '%s'", run1.Command)
	}

	run2 := df.Stages[1].Instructions[0].(*parser.RunInstruction)
	if run2.Command != "npm install" {
		t.Errorf("stage 1: expected 'npm install', got '%s'", run2.Command)
	}
}

func TestRemoveSudoTransform_WithDiagnostics(t *testing.T) {
	// Verify transform works correctly with diagnostics passed in
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Command: "sudo apt-get update",
					},
				},
			},
		},
	}

	diags := []analyzer.Diagnostic{
		{Rule: "SEC005"},
	}

	tr := &RemoveSudoTransform{}
	changed := tr.Transform(df, diags)

	if !changed {
		t.Error("expected transform to report changes")
	}
}
