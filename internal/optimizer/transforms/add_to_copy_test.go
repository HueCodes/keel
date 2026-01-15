package transforms

import (
	"testing"

	"github.com/HueCodes/keel/internal/parser"
)

func TestAddToCopyTransform_Name(t *testing.T) {
	tr := &AddToCopyTransform{}
	if tr.Name() != "add-to-copy" {
		t.Errorf("expected name 'add-to-copy', got %s", tr.Name())
	}
}

func TestAddToCopyTransform_Rules(t *testing.T) {
	tr := &AddToCopyTransform{}
	rules := tr.Rules()
	if len(rules) != 1 || rules[0] != "BP002" {
		t.Errorf("expected rules ['BP002'], got %v", rules)
	}
}

func TestAddToCopyTransform_SimpleCopy(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.AddInstruction{
						Sources:     []string{"src/"},
						Destination: "/app/",
					},
				},
			},
		},
	}

	tr := &AddToCopyTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	// Verify it's now a COPY instruction
	copy, ok := df.Stages[0].Instructions[0].(*parser.CopyInstruction)
	if !ok {
		t.Fatal("expected instruction to be converted to CopyInstruction")
	}
	if len(copy.Sources) != 1 || copy.Sources[0] != "src/" {
		t.Errorf("expected sources ['src/'], got %v", copy.Sources)
	}
	if copy.Destination != "/app/" {
		t.Errorf("expected destination '/app/', got %s", copy.Destination)
	}
}

func TestAddToCopyTransform_TarFile(t *testing.T) {
	// ADD with tar file should NOT be transformed
	tarFiles := []string{
		"app.tar",
		"app.tar.gz",
		"app.tgz",
		"app.tar.bz2",
		"app.tar.xz",
		"app.txz",
	}

	for _, tarFile := range tarFiles {
		t.Run(tarFile, func(t *testing.T) {
			df := &parser.Dockerfile{
				Stages: []*parser.Stage{
					{
						Instructions: []parser.Instruction{
							&parser.AddInstruction{
								Sources:     []string{tarFile},
								Destination: "/app/",
							},
						},
					},
				},
			}

			tr := &AddToCopyTransform{}
			changed := tr.Transform(df, nil)

			if changed {
				t.Error("expected transform to NOT modify tar file ADD")
			}

			_, ok := df.Stages[0].Instructions[0].(*parser.AddInstruction)
			if !ok {
				t.Error("expected instruction to remain AddInstruction")
			}
		})
	}
}

func TestAddToCopyTransform_URL(t *testing.T) {
	// ADD with URL should NOT be transformed
	urls := []string{
		"http://example.com/file.txt",
		"https://example.com/file.txt",
		"ftp://example.com/file.txt",
		"HTTP://EXAMPLE.COM/FILE.TXT", // case insensitive
	}

	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			df := &parser.Dockerfile{
				Stages: []*parser.Stage{
					{
						Instructions: []parser.Instruction{
							&parser.AddInstruction{
								Sources:     []string{url},
								Destination: "/app/",
							},
						},
					},
				},
			}

			tr := &AddToCopyTransform{}
			changed := tr.Transform(df, nil)

			if changed {
				t.Error("expected transform to NOT modify URL ADD")
			}

			_, ok := df.Stages[0].Instructions[0].(*parser.AddInstruction)
			if !ok {
				t.Error("expected instruction to remain AddInstruction")
			}
		})
	}
}

func TestAddToCopyTransform_PreservesFlags(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.AddInstruction{
						Sources:     []string{"src/"},
						Destination: "/app/",
						Chown:       "appuser:appgroup",
						Chmod:       "755",
					},
				},
			},
		},
	}

	tr := &AddToCopyTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	copy, ok := df.Stages[0].Instructions[0].(*parser.CopyInstruction)
	if !ok {
		t.Fatal("expected instruction to be converted to CopyInstruction")
	}
	if copy.Chown != "appuser:appgroup" {
		t.Errorf("expected Chown 'appuser:appgroup', got %s", copy.Chown)
	}
	if copy.Chmod != "755" {
		t.Errorf("expected Chmod '755', got %s", copy.Chmod)
	}
}

func TestAddToCopyTransform_MultipleFiles(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.AddInstruction{
						Sources:     []string{"file1.txt", "file2.txt", "dir/"},
						Destination: "/app/",
					},
				},
			},
		},
	}

	tr := &AddToCopyTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	copy, ok := df.Stages[0].Instructions[0].(*parser.CopyInstruction)
	if !ok {
		t.Fatal("expected instruction to be converted to CopyInstruction")
	}
	if len(copy.Sources) != 3 {
		t.Errorf("expected 3 sources, got %d", len(copy.Sources))
	}
}

func TestAddToCopyTransform_MixedSources(t *testing.T) {
	// If ANY source is a tar file, keep ADD
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.AddInstruction{
						Sources:     []string{"file.txt", "app.tar.gz"},
						Destination: "/app/",
					},
				},
			},
		},
	}

	tr := &AddToCopyTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected transform to NOT modify mixed sources with tar")
	}

	_, ok := df.Stages[0].Instructions[0].(*parser.AddInstruction)
	if !ok {
		t.Error("expected instruction to remain AddInstruction")
	}
}

func TestAddToCopyTransform_MultipleStages(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.AddInstruction{
						Sources:     []string{"file1.txt"},
						Destination: "/app/",
					},
				},
			},
			{
				Instructions: []parser.Instruction{
					&parser.AddInstruction{
						Sources:     []string{"file2.txt"},
						Destination: "/app/",
					},
					&parser.AddInstruction{
						Sources:     []string{"http://example.com/file"},
						Destination: "/app/",
					},
				},
			},
		},
	}

	tr := &AddToCopyTransform{}
	changed := tr.Transform(df, nil)

	if !changed {
		t.Error("expected transform to report changes")
	}

	// First stage should be COPY
	_, ok := df.Stages[0].Instructions[0].(*parser.CopyInstruction)
	if !ok {
		t.Error("stage 0: expected COPY instruction")
	}

	// Second stage first instruction should be COPY
	_, ok = df.Stages[1].Instructions[0].(*parser.CopyInstruction)
	if !ok {
		t.Error("stage 1 inst 0: expected COPY instruction")
	}

	// Second stage second instruction should remain ADD (URL)
	_, ok = df.Stages[1].Instructions[1].(*parser.AddInstruction)
	if !ok {
		t.Error("stage 1 inst 1: expected ADD instruction (URL)")
	}
}

func TestAddToCopyTransform_PreservesOtherInstructions(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.RunInstruction{
						Command: "apt-get update",
					},
					&parser.AddInstruction{
						Sources:     []string{"src/"},
						Destination: "/app/",
					},
					&parser.EnvInstruction{
						Variables: []parser.KeyValue{{Key: "FOO", Value: "bar"}},
					},
				},
			},
		},
	}

	tr := &AddToCopyTransform{}
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

	_, ok = df.Stages[0].Instructions[1].(*parser.CopyInstruction)
	if !ok {
		t.Error("second instruction should be CopyInstruction")
	}

	_, ok = df.Stages[0].Instructions[2].(*parser.EnvInstruction)
	if !ok {
		t.Error("third instruction should be EnvInstruction")
	}
}

func TestAddToCopyTransform_NoAddInstructions(t *testing.T) {
	df := &parser.Dockerfile{
		Stages: []*parser.Stage{
			{
				Instructions: []parser.Instruction{
					&parser.CopyInstruction{
						Sources:     []string{"src/"},
						Destination: "/app/",
					},
				},
			},
		},
	}

	tr := &AddToCopyTransform{}
	changed := tr.Transform(df, nil)

	if changed {
		t.Error("expected transform to report no changes")
	}
}
