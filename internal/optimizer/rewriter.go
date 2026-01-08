package optimizer

import (
	"fmt"
	"strings"

	"github.com/HueCodes/keel/internal/parser"
)

// Rewriter converts an AST back to Dockerfile text
type Rewriter struct {
	indent     string
	lineLength int
}

// RewriterOption configures a Rewriter
type RewriterOption func(*Rewriter)

// NewRewriter creates a new Rewriter
func NewRewriter(opts ...RewriterOption) *Rewriter {
	r := &Rewriter{
		indent:     "    ",
		lineLength: 80,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithIndent sets the indentation string
func WithIndent(indent string) RewriterOption {
	return func(r *Rewriter) {
		r.indent = indent
	}
}

// WithLineLength sets the max line length for wrapping
func WithLineLength(length int) RewriterOption {
	return func(r *Rewriter) {
		r.lineLength = length
	}
}

// Rewrite converts the AST back to Dockerfile text
func (r *Rewriter) Rewrite(df *parser.Dockerfile) string {
	var sb strings.Builder

	// Write escape directive if non-default
	if df.Escape != '\\' && df.Escape != 0 {
		sb.WriteString(fmt.Sprintf("# escape=%c\n", df.Escape))
	}

	// Write top-level comments
	for _, comment := range df.Comments {
		sb.WriteString(comment.Text)
		sb.WriteString("\n")
	}

	// Write stages
	for i, stage := range df.Stages {
		if i > 0 {
			sb.WriteString("\n")
		}
		r.writeStage(&sb, stage)
	}

	return sb.String()
}

func (r *Rewriter) writeStage(sb *strings.Builder, stage *parser.Stage) {
	// Write stage comments
	for _, comment := range stage.Comments {
		sb.WriteString(comment.Text)
		sb.WriteString("\n")
	}

	// Write FROM instruction
	if stage.From != nil {
		r.writeFrom(sb, stage.From)
	}

	// Write other instructions
	for _, inst := range stage.Instructions {
		r.writeInstruction(sb, inst)
	}
}

func (r *Rewriter) writeFrom(sb *strings.Builder, from *parser.FromInstruction) {
	sb.WriteString("FROM ")

	if from.Platform != "" {
		sb.WriteString("--platform=")
		sb.WriteString(from.Platform)
		sb.WriteString(" ")
	}

	sb.WriteString(from.Image)

	if from.Tag != "" {
		sb.WriteString(":")
		sb.WriteString(from.Tag)
	}

	if from.Digest != "" {
		sb.WriteString("@")
		sb.WriteString(from.Digest)
	}

	if from.AsName != "" {
		sb.WriteString(" AS ")
		sb.WriteString(from.AsName)
	}

	sb.WriteString("\n")
}

func (r *Rewriter) writeInstruction(sb *strings.Builder, inst parser.Instruction) {
	switch v := inst.(type) {
	case *parser.RunInstruction:
		r.writeRun(sb, v)
	case *parser.CmdInstruction:
		r.writeCmd(sb, v)
	case *parser.EntrypointInstruction:
		r.writeEntrypoint(sb, v)
	case *parser.CopyInstruction:
		r.writeCopy(sb, v)
	case *parser.AddInstruction:
		r.writeAdd(sb, v)
	case *parser.EnvInstruction:
		r.writeEnv(sb, v)
	case *parser.ArgInstruction:
		r.writeArg(sb, v)
	case *parser.LabelInstruction:
		r.writeLabel(sb, v)
	case *parser.ExposeInstruction:
		r.writeExpose(sb, v)
	case *parser.VolumeInstruction:
		r.writeVolume(sb, v)
	case *parser.UserInstruction:
		r.writeUser(sb, v)
	case *parser.WorkdirInstruction:
		r.writeWorkdir(sb, v)
	case *parser.ShellInstruction:
		r.writeShell(sb, v)
	case *parser.HealthcheckInstruction:
		r.writeHealthcheck(sb, v)
	case *parser.StopsignalInstruction:
		r.writeStopsignal(sb, v)
	case *parser.OnbuildInstruction:
		r.writeOnbuild(sb, v)
	case *parser.MaintainerInstruction:
		r.writeMaintainer(sb, v)
	}
}

func (r *Rewriter) writeRun(sb *strings.Builder, run *parser.RunInstruction) {
	sb.WriteString("RUN ")

	if run.Mount != "" {
		sb.WriteString("--mount=")
		sb.WriteString(run.Mount)
		sb.WriteString(" ")
	}

	if run.Network != "" {
		sb.WriteString("--network=")
		sb.WriteString(run.Network)
		sb.WriteString(" ")
	}

	if run.Heredoc != nil {
		sb.WriteString(run.Heredoc.Content)
	} else if run.IsExec {
		r.writeExecForm(sb, run.Arguments)
	} else {
		// Format long commands with line continuation
		cmd := run.Command
		if strings.Contains(cmd, " && ") || strings.Contains(cmd, " \\\n") {
			r.writeMultilineCommand(sb, cmd)
		} else {
			sb.WriteString(cmd)
		}
	}

	sb.WriteString("\n")
}

func (r *Rewriter) writeMultilineCommand(sb *strings.Builder, cmd string) {
	// Split by && and format nicely
	parts := strings.Split(cmd, " && ")
	if len(parts) == 1 {
		// Check for existing line continuations
		parts = strings.Split(cmd, " \\\n")
		if len(parts) == 1 {
			sb.WriteString(cmd)
			return
		}
	}

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if i == 0 {
			sb.WriteString(part)
		} else {
			sb.WriteString(" \\\n")
			sb.WriteString(r.indent)
			sb.WriteString("&& ")
			sb.WriteString(part)
		}
	}
}

func (r *Rewriter) writeCmd(sb *strings.Builder, cmd *parser.CmdInstruction) {
	sb.WriteString("CMD ")
	if cmd.IsExec {
		r.writeExecForm(sb, cmd.Arguments)
	} else {
		sb.WriteString(cmd.Command)
	}
	sb.WriteString("\n")
}

func (r *Rewriter) writeEntrypoint(sb *strings.Builder, ep *parser.EntrypointInstruction) {
	sb.WriteString("ENTRYPOINT ")
	if ep.IsExec {
		r.writeExecForm(sb, ep.Arguments)
	} else {
		sb.WriteString(ep.Command)
	}
	sb.WriteString("\n")
}

func (r *Rewriter) writeCopy(sb *strings.Builder, cp *parser.CopyInstruction) {
	sb.WriteString("COPY ")

	if cp.From != "" {
		sb.WriteString("--from=")
		sb.WriteString(cp.From)
		sb.WriteString(" ")
	}
	if cp.Chown != "" {
		sb.WriteString("--chown=")
		sb.WriteString(cp.Chown)
		sb.WriteString(" ")
	}
	if cp.Chmod != "" {
		sb.WriteString("--chmod=")
		sb.WriteString(cp.Chmod)
		sb.WriteString(" ")
	}
	if cp.Link {
		sb.WriteString("--link ")
	}

	for _, src := range cp.Sources {
		sb.WriteString(src)
		sb.WriteString(" ")
	}
	sb.WriteString(cp.Destination)
	sb.WriteString("\n")
}

func (r *Rewriter) writeAdd(sb *strings.Builder, add *parser.AddInstruction) {
	sb.WriteString("ADD ")

	if add.Chown != "" {
		sb.WriteString("--chown=")
		sb.WriteString(add.Chown)
		sb.WriteString(" ")
	}
	if add.Chmod != "" {
		sb.WriteString("--chmod=")
		sb.WriteString(add.Chmod)
		sb.WriteString(" ")
	}
	if add.Checksum != "" {
		sb.WriteString("--checksum=")
		sb.WriteString(add.Checksum)
		sb.WriteString(" ")
	}

	for _, src := range add.Sources {
		sb.WriteString(src)
		sb.WriteString(" ")
	}
	sb.WriteString(add.Destination)
	sb.WriteString("\n")
}

func (r *Rewriter) writeEnv(sb *strings.Builder, env *parser.EnvInstruction) {
	sb.WriteString("ENV ")
	for i, kv := range env.Variables {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(kv.Key)
		sb.WriteString("=")
		if strings.Contains(kv.Value, " ") {
			sb.WriteString("\"")
			sb.WriteString(kv.Value)
			sb.WriteString("\"")
		} else {
			sb.WriteString(kv.Value)
		}
	}
	sb.WriteString("\n")
}

func (r *Rewriter) writeArg(sb *strings.Builder, arg *parser.ArgInstruction) {
	sb.WriteString("ARG ")
	sb.WriteString(arg.Name)
	if arg.HasDefault {
		sb.WriteString("=")
		sb.WriteString(arg.DefaultValue)
	}
	sb.WriteString("\n")
}

func (r *Rewriter) writeLabel(sb *strings.Builder, label *parser.LabelInstruction) {
	sb.WriteString("LABEL ")
	for i, kv := range label.Labels {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(kv.Key)
		sb.WriteString("=")
		if strings.Contains(kv.Value, " ") {
			sb.WriteString("\"")
			sb.WriteString(kv.Value)
			sb.WriteString("\"")
		} else {
			sb.WriteString(kv.Value)
		}
	}
	sb.WriteString("\n")
}

func (r *Rewriter) writeExpose(sb *strings.Builder, expose *parser.ExposeInstruction) {
	sb.WriteString("EXPOSE ")
	for i, port := range expose.Ports {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(port.Port)
		if port.Protocol != "" {
			sb.WriteString("/")
			sb.WriteString(port.Protocol)
		}
	}
	sb.WriteString("\n")
}

func (r *Rewriter) writeVolume(sb *strings.Builder, vol *parser.VolumeInstruction) {
	sb.WriteString("VOLUME ")
	if len(vol.Paths) > 1 {
		r.writeExecForm(sb, vol.Paths)
	} else if len(vol.Paths) == 1 {
		sb.WriteString(vol.Paths[0])
	}
	sb.WriteString("\n")
}

func (r *Rewriter) writeUser(sb *strings.Builder, user *parser.UserInstruction) {
	sb.WriteString("USER ")
	sb.WriteString(user.User)
	if user.Group != "" {
		sb.WriteString(":")
		sb.WriteString(user.Group)
	}
	sb.WriteString("\n")
}

func (r *Rewriter) writeWorkdir(sb *strings.Builder, wd *parser.WorkdirInstruction) {
	sb.WriteString("WORKDIR ")
	sb.WriteString(wd.Path)
	sb.WriteString("\n")
}

func (r *Rewriter) writeShell(sb *strings.Builder, shell *parser.ShellInstruction) {
	sb.WriteString("SHELL ")
	r.writeExecForm(sb, shell.Shell)
	sb.WriteString("\n")
}

func (r *Rewriter) writeHealthcheck(sb *strings.Builder, hc *parser.HealthcheckInstruction) {
	sb.WriteString("HEALTHCHECK ")

	if hc.None {
		sb.WriteString("NONE")
		sb.WriteString("\n")
		return
	}

	if hc.Interval != "" {
		sb.WriteString("--interval=")
		sb.WriteString(hc.Interval)
		sb.WriteString(" ")
	}
	if hc.Timeout != "" {
		sb.WriteString("--timeout=")
		sb.WriteString(hc.Timeout)
		sb.WriteString(" ")
	}
	if hc.StartPeriod != "" {
		sb.WriteString("--start-period=")
		sb.WriteString(hc.StartPeriod)
		sb.WriteString(" ")
	}
	if hc.Retries != "" {
		sb.WriteString("--retries=")
		sb.WriteString(hc.Retries)
		sb.WriteString(" ")
	}

	sb.WriteString("CMD ")
	if hc.IsExec {
		r.writeExecForm(sb, hc.Arguments)
	} else {
		sb.WriteString(hc.Command)
	}
	sb.WriteString("\n")
}

func (r *Rewriter) writeStopsignal(sb *strings.Builder, ss *parser.StopsignalInstruction) {
	sb.WriteString("STOPSIGNAL ")
	sb.WriteString(ss.Signal)
	sb.WriteString("\n")
}

func (r *Rewriter) writeOnbuild(sb *strings.Builder, ob *parser.OnbuildInstruction) {
	sb.WriteString("ONBUILD ")
	if ob.Instruction != nil {
		// Write the nested instruction without newline
		var nested strings.Builder
		r.writeInstruction(&nested, ob.Instruction)
		sb.WriteString(strings.TrimRight(nested.String(), "\n"))
	}
	sb.WriteString("\n")
}

func (r *Rewriter) writeMaintainer(sb *strings.Builder, maint *parser.MaintainerInstruction) {
	// Convert deprecated MAINTAINER to LABEL
	sb.WriteString("LABEL maintainer=\"")
	sb.WriteString(maint.Maintainer)
	sb.WriteString("\"\n")
}

func (r *Rewriter) writeExecForm(sb *strings.Builder, args []string) {
	sb.WriteString("[")
	for i, arg := range args {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("\"")
		sb.WriteString(arg)
		sb.WriteString("\"")
	}
	sb.WriteString("]")
}
