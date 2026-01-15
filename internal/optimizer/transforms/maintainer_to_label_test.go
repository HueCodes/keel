package transforms

import (
	"testing"

	"github.com/HueCodes/keel/internal/lexer"
	"github.com/HueCodes/keel/internal/parser"
)

func TestMaintainerToLabelTransform_Name(t *testing.T) {
	tr := &MaintainerToLabelTransform{}
	if tr.Name() != "maintainer-to-label" {
		t.Errorf("expected name 'maintainer-to-label', got %s", tr.Name())
	}
}

func TestMaintainerToLabelTransform_Rules(t *testing.T) {
	tr := &MaintainerToLabelTransform{}
	rules := tr.Rules()
	if len(rules) != 1 || rules[0] != "BP004" {
		t.Errorf("expected rules ['BP004'], got %v", rules)
	}
}

func TestMaintainerToLabelTransform_Simple(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.MaintainerInstruction{
						Maintainer: "John Doe",
					},
				},
			},
		},
	}

	tr := &MaintainerToLabelTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	label, ok := df.Stages[0].Instructions[0].(*parser.LabelInstruction)
	if !ok {
		t.Fatal("expected instruction to be converted to LabelInstruction")
	}
	if len(label.Labels) != 1 {
		t.Fatalf("expected 1 label, got %d", len(label.Labels))
	}
	if label.Labels[0].Key != "maintainer" {
		t.Errorf("expected key 'maintainer', got %s", label.Labels[0].Key)
	}
	if label.Labels[0].Value != "John Doe" {
		t.Errorf("expected value 'John Doe', got %s", label.Labels[0].Value)
	}
}

func TestMaintainerToLabelTransform_WithEmail(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.MaintainerInstruction{
						Maintainer: "John Doe <john@example.com>",
					},
				},
			},
		},
	}

	tr := &MaintainerToLabelTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	label, ok := df.Stages[0].Instructions[0].(*parser.LabelInstruction)
	if !ok {
		t.Fatal("expected instruction to be converted to LabelInstruction")
	}
	if label.Labels[0].Value != "John Doe <john@example.com>" {
		t.Errorf("expected value with email, got %s", label.Labels[0].Value)
	}
}

func TestMaintainerToLabelTransform_Quoted(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.MaintainerInstruction{
						Maintainer: `"John Doe" <john@example.com>`,
					},
				},
			},
		},
	}

	tr := &MaintainerToLabelTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	label, ok := df.Stages[0].Instructions[0].(*parser.LabelInstruction)
	if !ok {
		t.Fatal("expected instruction to be converted to LabelInstruction")
	}
	// Value should be preserved as-is
	if label.Labels[0].Value != `"John Doe" <john@example.com>` {
		t.Errorf("expected quoted value preserved, got %s", label.Labels[0].Value)
	}
}

func TestMaintainerToLabelTransform_NoMaintainer(t *testing.T) {
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

	tr := &MaintainerToLabelTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected transform to report no changes")
	}

	_, ok := df.Stages[0].Instructions[0].(*parser.RunInstruction)
	if !ok {
		t.Error("expected instruction to remain RunInstruction")
	}
}

func TestMaintainerToLabelTransform_MultipleStages(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.MaintainerInstruction{
						Maintainer: "John Doe",
					},
				},
			},
			{
				Instructions: []parser.Instruction{
					&parser.MaintainerInstruction{
						Maintainer: "Jane Doe",
					},
				},
			},
		},
	}

	tr := &MaintainerToLabelTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	label1, ok := df.Stages[0].Instructions[0].(*parser.LabelInstruction)
	if !ok {
		t.Fatal("stage 0: expected LabelInstruction")
	}
	if label1.Labels[0].Value != "John Doe" {
		t.Errorf("stage 0: expected 'John Doe', got %s", label1.Labels[0].Value)
	}

	label2, ok := df.Stages[1].Instructions[0].(*parser.LabelInstruction)
	if !ok {
		t.Fatal("stage 1: expected LabelInstruction")
	}
	if label2.Labels[0].Value != "Jane Doe" {
		t.Errorf("stage 1: expected 'Jane Doe', got %s", label2.Labels[0].Value)
	}
}

func TestMaintainerToLabelTransform_PreservesOtherInstructions(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Command: "apt-get update",
					},
					&parser.MaintainerInstruction{
						Maintainer: "John Doe",
					},
					&parser.EnvInstruction{
						Variables: []parser.KeyValue{{Key: "FOO", Value: "bar"}},
					},
				},
			},
		},
	}

	tr := &MaintainerToLabelTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	if len(df.Stages[0].Instructions) != 3 {
		t.Errorf("expected 3 instructions, got %d", len(df.Stages[0].Instructions))
	}

	_, ok := df.Stages[0].Instructions[0].(*parser.RunInstruction)
	if !ok {
		t.Error("first instruction should be RunInstruction")
	}

	_, ok = df.Stages[0].Instructions[1].(*parser.LabelInstruction)
	if !ok {
		t.Error("second instruction should be LabelInstruction")
	}

	_, ok = df.Stages[0].Instructions[2].(*parser.EnvInstruction)
	if !ok {
		t.Error("third instruction should be EnvInstruction")
	}
}

func TestMaintainerToLabelTransform_PreservesPosition(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.MaintainerInstruction{
						BaseInstruction: parser.BaseInstruction{
							StartPos: lexer.Position{Line: 5, Column: 1},
							EndPos:   lexer.Position{Line: 5, Column: 25},
						},
						Maintainer: "John Doe",
					},
				},
			},
		},
	}

	tr := &MaintainerToLabelTransform{}
	tr.Transform(df, nil)

	label := df.Stages[0].Instructions[0].(*parser.LabelInstruction)
	if label.Pos().Line != 5 {
		t.Errorf("expected line 5, got %d", label.Pos().Line)
	}
}
