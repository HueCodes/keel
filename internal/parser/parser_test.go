package parser

import (
	"testing"
)

func TestParseSimpleDockerfile(t *testing.T) {
	input := `FROM ubuntu:22.04
RUN apt-get update
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(df.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(df.Stages))
	}

	stage := df.Stages[0]
	if stage.From.Image != "ubuntu" {
		t.Errorf("expected image 'ubuntu', got %q", stage.From.Image)
	}
	if stage.From.Tag != "22.04" {
		t.Errorf("expected tag '22.04', got %q", stage.From.Tag)
	}

	if len(stage.Instructions) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(stage.Instructions))
	}

	run, ok := stage.Instructions[0].(*RunInstruction)
	if !ok {
		t.Fatalf("expected RunInstruction, got %T", stage.Instructions[0])
	}
	if run.Command == "" {
		t.Error("expected non-empty command")
	}
}

func TestParseMultiStage(t *testing.T) {
	input := `FROM golang:1.21 AS builder
RUN go build -o /app

FROM alpine:3.18
COPY --from=builder /app /app
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(df.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(df.Stages))
	}

	// First stage
	if df.Stages[0].Name != "builder" {
		t.Errorf("expected stage name 'builder', got %q", df.Stages[0].Name)
	}
	if df.Stages[0].From.Image != "golang" {
		t.Errorf("expected image 'golang', got %q", df.Stages[0].From.Image)
	}

	// Second stage
	if df.Stages[1].From.Image != "alpine" {
		t.Errorf("expected image 'alpine', got %q", df.Stages[1].From.Image)
	}

	// COPY with --from
	if len(df.Stages[1].Instructions) != 1 {
		t.Fatalf("expected 1 instruction in second stage, got %d", len(df.Stages[1].Instructions))
	}
	copy, ok := df.Stages[1].Instructions[0].(*CopyInstruction)
	if !ok {
		t.Fatalf("expected CopyInstruction, got %T", df.Stages[1].Instructions[0])
	}
	if copy.From != "builder" {
		t.Errorf("expected --from=builder, got %q", copy.From)
	}
}

func TestParseEnv(t *testing.T) {
	input := `FROM alpine
ENV FOO=bar BAZ="hello world"
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	env := df.Stages[0].Instructions[0].(*EnvInstruction)
	if len(env.Variables) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(env.Variables))
	}
	if env.Variables[0].Key != "FOO" || env.Variables[0].Value != "bar" {
		t.Errorf("expected FOO=bar, got %s=%s", env.Variables[0].Key, env.Variables[0].Value)
	}
}

func TestParseLabel(t *testing.T) {
	input := `FROM alpine
LABEL maintainer="test@example.com" version="1.0"
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	label := df.Stages[0].Instructions[0].(*LabelInstruction)
	if len(label.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(label.Labels))
	}
}

func TestParseExecForm(t *testing.T) {
	input := `FROM alpine
CMD ["echo", "hello", "world"]
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	cmd := df.Stages[0].Instructions[0].(*CmdInstruction)
	if !cmd.IsExec {
		t.Error("expected exec form")
	}
	if len(cmd.Arguments) != 3 {
		t.Fatalf("expected 3 arguments, got %d", len(cmd.Arguments))
	}
	if cmd.Arguments[0] != "echo" {
		t.Errorf("expected 'echo', got %q", cmd.Arguments[0])
	}
}

func TestParseExpose(t *testing.T) {
	input := `FROM alpine
EXPOSE 80 443/tcp 8080/udp
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	expose := df.Stages[0].Instructions[0].(*ExposeInstruction)
	if len(expose.Ports) != 3 {
		t.Fatalf("expected 3 ports, got %d", len(expose.Ports))
	}
}

func TestParseUser(t *testing.T) {
	input := `FROM alpine
USER nobody:nogroup
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	user := df.Stages[0].Instructions[0].(*UserInstruction)
	if user.User != "nobody" {
		t.Errorf("expected user 'nobody', got %q", user.User)
	}
	if user.Group != "nogroup" {
		t.Errorf("expected group 'nogroup', got %q", user.Group)
	}
}

func TestParseHealthcheck(t *testing.T) {
	input := `FROM alpine
HEALTHCHECK --interval=30s --timeout=10s CMD curl -f http://localhost/ || exit 1
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	hc := df.Stages[0].Instructions[0].(*HealthcheckInstruction)
	if hc.None {
		t.Error("expected HEALTHCHECK with command, not NONE")
	}
	if hc.Interval != "30s" {
		t.Errorf("expected interval '30s', got %q", hc.Interval)
	}
	if hc.Timeout != "10s" {
		t.Errorf("expected timeout '10s', got %q", hc.Timeout)
	}
}

func TestParseHealthcheckNone(t *testing.T) {
	input := `FROM alpine
HEALTHCHECK NONE
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	hc := df.Stages[0].Instructions[0].(*HealthcheckInstruction)
	if !hc.None {
		t.Error("expected HEALTHCHECK NONE")
	}
}

func TestParseArg(t *testing.T) {
	input := `FROM alpine
ARG VERSION=1.0
ARG NAME
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	arg1 := df.Stages[0].Instructions[0].(*ArgInstruction)
	if arg1.Name != "VERSION" {
		t.Errorf("expected name 'VERSION', got %q", arg1.Name)
	}
	if !arg1.HasDefault {
		t.Error("expected HasDefault to be true")
	}
	if arg1.DefaultValue != "1.0" {
		t.Errorf("expected default '1.0', got %q", arg1.DefaultValue)
	}

	arg2 := df.Stages[0].Instructions[1].(*ArgInstruction)
	if arg2.Name != "NAME" {
		t.Errorf("expected name 'NAME', got %q", arg2.Name)
	}
	if arg2.HasDefault {
		t.Error("expected HasDefault to be false")
	}
}

func TestParseCopyFlags(t *testing.T) {
	input := `FROM alpine
COPY --chmod=755 --chown=root:root src/ /app/
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	copy := df.Stages[0].Instructions[0].(*CopyInstruction)
	if copy.Chmod != "755" {
		t.Errorf("expected chmod '755', got %q", copy.Chmod)
	}
	if copy.Chown != "root:root" {
		t.Errorf("expected chown 'root:root', got %q", copy.Chown)
	}
}

func TestParseWorkdir(t *testing.T) {
	input := `FROM alpine
WORKDIR /app
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	wd := df.Stages[0].Instructions[0].(*WorkdirInstruction)
	if wd.Path != "/app" {
		t.Errorf("expected path '/app', got %q", wd.Path)
	}
}

func TestParseVolume(t *testing.T) {
	input := `FROM alpine
VOLUME ["/data", "/logs"]
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	vol := df.Stages[0].Instructions[0].(*VolumeInstruction)
	if len(vol.Paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(vol.Paths))
	}
}

func TestParseShell(t *testing.T) {
	input := `FROM alpine
SHELL ["/bin/bash", "-c"]
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	shell := df.Stages[0].Instructions[0].(*ShellInstruction)
	if len(shell.Shell) != 2 {
		t.Fatalf("expected 2 shell args, got %d", len(shell.Shell))
	}
	if shell.Shell[0] != "/bin/bash" {
		t.Errorf("expected '/bin/bash', got %q", shell.Shell[0])
	}
}

func TestParseMaintainer(t *testing.T) {
	input := `FROM alpine
MAINTAINER test@example.com
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	maint := df.Stages[0].Instructions[0].(*MaintainerInstruction)
	if maint.Maintainer != "test@example.com" {
		t.Errorf("expected 'test@example.com', got %q", maint.Maintainer)
	}
}

func TestParseOnbuild(t *testing.T) {
	input := `FROM alpine
ONBUILD RUN echo "triggered"
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	onbuild := df.Stages[0].Instructions[0].(*OnbuildInstruction)
	if onbuild.Instruction == nil {
		t.Fatal("expected nested instruction")
	}
	_, ok := onbuild.Instruction.(*RunInstruction)
	if !ok {
		t.Errorf("expected nested RUN, got %T", onbuild.Instruction)
	}
}

func TestParseFromPlatform(t *testing.T) {
	input := `FROM --platform=linux/amd64 alpine:3.18
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	from := df.Stages[0].From
	if from.Platform != "linux/amd64" {
		t.Errorf("expected platform 'linux/amd64', got %q", from.Platform)
	}
}

func TestParseFromDigest(t *testing.T) {
	input := `FROM alpine@sha256:abc123
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	from := df.Stages[0].From
	if from.Image != "alpine" {
		t.Errorf("expected image 'alpine', got %q", from.Image)
	}
	if from.Digest != "sha256" {
		t.Errorf("expected digest 'sha256:abc123', got %q", from.Digest)
	}
}

func TestGetInstructions(t *testing.T) {
	input := `FROM alpine
RUN echo 1
RUN echo 2
CMD ["echo", "done"]
`
	df, _ := Parse(input)

	runs := GetInstructions[*RunInstruction](df)
	if len(runs) != 2 {
		t.Errorf("expected 2 RUN instructions, got %d", len(runs))
	}

	cmds := GetInstructions[*CmdInstruction](df)
	if len(cmds) != 1 {
		t.Errorf("expected 1 CMD instruction, got %d", len(cmds))
	}
}

func TestHasInstruction(t *testing.T) {
	input := `FROM alpine
RUN echo hello
`
	df, _ := Parse(input)

	if !HasInstruction[*RunInstruction](df) {
		t.Error("expected to find RUN instruction")
	}
	if HasInstruction[*UserInstruction](df) {
		t.Error("did not expect to find USER instruction")
	}
}

func TestParseComments(t *testing.T) {
	input := `# This is a comment
FROM alpine
# Another comment
RUN echo hello
`
	df, errs := Parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(df.Comments) == 0 && len(df.Stages[0].Comments) == 0 {
		t.Error("expected to find comments")
	}
}

func TestPortSpecPrivileged(t *testing.T) {
	tests := []struct {
		port       string
		privileged bool
	}{
		{"80", true},
		{"443", true},
		{"22", true},
		{"1023", true},
		{"1024", false},
		{"8080", false},
		{"3000", false},
	}

	for _, tt := range tests {
		ps := PortSpec{Port: tt.port}
		if ps.IsPrivilegedPort() != tt.privileged {
			t.Errorf("port %s: expected privileged=%v, got %v", tt.port, tt.privileged, ps.IsPrivilegedPort())
		}
	}
}
