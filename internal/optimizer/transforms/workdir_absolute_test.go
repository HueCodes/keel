package transforms

import (
	"testing"

	"github.com/HueCodes/keel/internal/parser"
)

func TestWorkdirAbsoluteTransform_Name(t *testing.T) {
	tr := &WorkdirAbsoluteTransform{}
	if tr.Name() != "workdir-absolute" {
		t.Errorf("expected name 'workdir-absolute', got %s", tr.Name())
	}
}

func TestWorkdirAbsoluteTransform_Rules(t *testing.T) {
	tr := &WorkdirAbsoluteTransform{}
	rules := tr.Rules()
	if len(rules) != 1 || rules[0] != "BP005" {
		t.Errorf("expected rules ['BP005'], got %v", rules)
	}
}

func TestWorkdirAbsoluteTransform_RelativePath(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.WorkdirInstruction{
						Path: "app",
					},
				},
			},
		},
	}

	tr := &WorkdirAbsoluteTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	wd := df.Stages[0].Instructions[0].(*parser.WorkdirInstruction)
	if wd.Path != "/app" {
		t.Errorf("expected '/app', got '%s'", wd.Path)
	}
}

func TestWorkdirAbsoluteTransform_AlreadyAbsolute(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.WorkdirInstruction{
						Path: "/app",
					},
				},
			},
		},
	}

	tr := &WorkdirAbsoluteTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected transform to report no changes")
	}

	wd := df.Stages[0].Instructions[0].(*parser.WorkdirInstruction)
	if wd.Path != "/app" {
		t.Errorf("expected '/app', got '%s'", wd.Path)
	}
}

func TestWorkdirAbsoluteTransform_ChainedRelative(t *testing.T) {
	// WORKDIR /app then WORKDIR src should become WORKDIR /app then WORKDIR /app/src
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.WorkdirInstruction{
						Path: "/app",
					},
					&parser.WorkdirInstruction{
						Path: "src",
					},
				},
			},
		},
	}

	tr := &WorkdirAbsoluteTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	wd1 := df.Stages[0].Instructions[0].(*parser.WorkdirInstruction)
	if wd1.Path != "/app" {
		t.Errorf("first WORKDIR: expected '/app', got '%s'", wd1.Path)
	}

	wd2 := df.Stages[0].Instructions[1].(*parser.WorkdirInstruction)
	if wd2.Path != "/app/src" {
		t.Errorf("second WORKDIR: expected '/app/src', got '%s'", wd2.Path)
	}
}

func TestWorkdirAbsoluteTransform_MultipleChained(t *testing.T) {
	// Start from /, then go to app, then to src, then to nested
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.WorkdirInstruction{
						Path: "app",
					},
					&parser.WorkdirInstruction{
						Path: "src",
					},
					&parser.WorkdirInstruction{
						Path: "nested",
					},
				},
			},
		},
	}

	tr := &WorkdirAbsoluteTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	wd1 := df.Stages[0].Instructions[0].(*parser.WorkdirInstruction)
	if wd1.Path != "/app" {
		t.Errorf("first WORKDIR: expected '/app', got '%s'", wd1.Path)
	}

	wd2 := df.Stages[0].Instructions[1].(*parser.WorkdirInstruction)
	if wd2.Path != "/app/src" {
		t.Errorf("second WORKDIR: expected '/app/src', got '%s'", wd2.Path)
	}

	wd3 := df.Stages[0].Instructions[2].(*parser.WorkdirInstruction)
	if wd3.Path != "/app/src/nested" {
		t.Errorf("third WORKDIR: expected '/app/src/nested', got '%s'", wd3.Path)
	}
}

func TestWorkdirAbsoluteTransform_Variable(t *testing.T) {
	// Variable paths should be skipped
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.WorkdirInstruction{
						Path: "$APP_DIR",
					},
				},
			},
		},
	}

	tr := &WorkdirAbsoluteTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected transform to NOT modify variable path")
	}

	wd := df.Stages[0].Instructions[0].(*parser.WorkdirInstruction)
	if wd.Path != "$APP_DIR" {
		t.Errorf("expected '$APP_DIR', got '%s'", wd.Path)
	}
}

func TestWorkdirAbsoluteTransform_VariableBraces(t *testing.T) {
	// Variable with braces should be skipped
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.WorkdirInstruction{
						Path: "${APP_DIR}",
					},
				},
			},
		},
	}

	tr := &WorkdirAbsoluteTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected transform to NOT modify variable path")
	}
}

func TestWorkdirAbsoluteTransform_MultiStage(t *testing.T) {
	// Each stage should start fresh from /
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.WorkdirInstruction{
						Path: "stage1",
					},
				},
			},
			{
				Instructions: []parser.Instruction{
					&parser.WorkdirInstruction{
						Path: "stage2",
					},
				},
			},
		},
	}

	tr := &WorkdirAbsoluteTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	wd1 := df.Stages[0].Instructions[0].(*parser.WorkdirInstruction)
	if wd1.Path != "/stage1" {
		t.Errorf("stage 0: expected '/stage1', got '%s'", wd1.Path)
	}

	wd2 := df.Stages[1].Instructions[0].(*parser.WorkdirInstruction)
	if wd2.Path != "/stage2" {
		t.Errorf("stage 1: expected '/stage2', got '%s'", wd2.Path)
	}
}

func TestWorkdirAbsoluteTransform_DotDot(t *testing.T) {
	// Handle .. in paths
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.WorkdirInstruction{
						Path: "/app/src",
					},
					&parser.WorkdirInstruction{
						Path: "../lib",
					},
				},
			},
		},
	}

	tr := &WorkdirAbsoluteTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	wd2 := df.Stages[1-1].Instructions[1].(*parser.WorkdirInstruction)
	if wd2.Path != "/app/lib" {
		t.Errorf("expected '/app/lib', got '%s'", wd2.Path)
	}
}

func TestWorkdirAbsoluteTransform_NestedPath(t *testing.T) {
	// Handle nested relative paths
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.WorkdirInstruction{
						Path: "subdir/nested",
					},
				},
			},
		},
	}

	tr := &WorkdirAbsoluteTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	wd := df.Stages[0].Instructions[0].(*parser.WorkdirInstruction)
	if wd.Path != "/subdir/nested" {
		t.Errorf("expected '/subdir/nested', got '%s'", wd.Path)
	}
}

func TestWorkdirAbsoluteTransform_MixedInstructions(t *testing.T) {
	// Ensure we handle other instructions mixed with WORKDIR
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Command: "apt-get update",
					},
					&parser.WorkdirInstruction{
						Path: "app",
					},
					&parser.CopyInstruction{
						Sources:     []string{"."},
						Destination: ".",
					},
					&parser.WorkdirInstruction{
						Path: "src",
					},
				},
			},
		},
	}

	tr := &WorkdirAbsoluteTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	wd1 := df.Stages[0].Instructions[1].(*parser.WorkdirInstruction)
	if wd1.Path != "/app" {
		t.Errorf("first WORKDIR: expected '/app', got '%s'", wd1.Path)
	}

	wd2 := df.Stages[0].Instructions[3].(*parser.WorkdirInstruction)
	if wd2.Path != "/app/src" {
		t.Errorf("second WORKDIR: expected '/app/src', got '%s'", wd2.Path)
	}
}

func TestWorkdirAbsoluteTransform_AbsoluteThenRelative(t *testing.T) {
	// After an absolute WORKDIR, relative should be based on that
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.WorkdirInstruction{
						Path: "/home/app",
					},
					&parser.WorkdirInstruction{
						Path: "data",
					},
				},
			},
		},
	}

	tr := &WorkdirAbsoluteTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	wd2 := df.Stages[0].Instructions[1].(*parser.WorkdirInstruction)
	if wd2.Path != "/home/app/data" {
		t.Errorf("expected '/home/app/data', got '%s'", wd2.Path)
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		base     string
		rel      string
		expected string
	}{
		{"/", "app", "/app"},
		{"/app", "src", "/app/src"},
		{"/app/", "src", "/app/src"},
		{"/app/src", "../lib", "/app/lib"},
		{"/", "a/b/c", "/a/b/c"},
		{"/app", "./src", "/app/src"},
	}

	for _, tt := range tests {
		t.Run(tt.base+"+"+tt.rel, func(t *testing.T) {
			result := joinPath(tt.base, tt.rel)
			if result != tt.expected {
				t.Errorf("joinPath(%q, %q) = %q, want %q", tt.base, tt.rel, result, tt.expected)
			}
		})
	}
}
