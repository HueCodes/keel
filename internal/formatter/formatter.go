package formatter

import (
	"fmt"
	"strings"

	"github.com/HueCodes/keel/internal/parser"
)

// Options configures the formatter behavior
type Options struct {
	IndentString         string // Indent string (default "    ")
	MaxLineLength        int    // Max line length before wrapping (default 80)
	AlignBackslashes     bool   // Align continuation backslashes
	AlignMultiValue      bool   // Align multi-value ENV/LABEL instructions
	RemoveExcessBlanks   bool   // Remove multiple consecutive blank lines
	MaxConsecutiveBlanks int    // Max consecutive blank lines to keep
}

// DefaultOptions returns the default formatting options
func DefaultOptions() Options {
	return Options{
		IndentString:         "    ",
		MaxLineLength:        80,
		AlignBackslashes:     true,
		AlignMultiValue:      true,
		RemoveExcessBlanks:   true,
		MaxConsecutiveBlanks: 1,
	}
}

// Formatter formats Dockerfiles for consistent style
type Formatter struct {
	opts Options
}

// New creates a new Formatter with the given options
func New(opts Options) *Formatter {
	return &Formatter{opts: opts}
}

// Result holds the formatting result
type Result struct {
	Original   string
	Formatted  string
	HasChanges bool
}

// Format formats a parsed Dockerfile
func (f *Formatter) Format(df *parser.Dockerfile) string {
	var sb strings.Builder

	// Handle escape directive if not default
	if df.Escape != 0 && df.Escape != '\\' {
		sb.WriteString(fmt.Sprintf("# escape=%c\n", df.Escape))
	}

	// Format top-level comments
	for _, comment := range df.Comments {
		f.writeComment(&sb, comment)
	}

	// Format stages
	for i, stage := range df.Stages {
		if i > 0 {
			sb.WriteString("\n")
		}
		f.formatStage(&sb, stage)
	}

	result := sb.String()

	// Clean up excessive blank lines
	if f.opts.RemoveExcessBlanks {
		result = f.normalizeBlankLines(result)
	}

	return result
}

// FormatSource parses and formats source code
func (f *Formatter) FormatSource(source string) (*Result, error) {
	df, parseErrors := parser.Parse(source)
	if len(parseErrors) > 0 {
		return nil, fmt.Errorf("parse error: %v", parseErrors[0])
	}

	formatted := f.Format(df)

	return &Result{
		Original:   source,
		Formatted:  formatted,
		HasChanges: source != formatted,
	}, nil
}

// formatStage formats a single build stage
func (f *Formatter) formatStage(sb *strings.Builder, stage *parser.Stage) {
	// Write stage comments
	for _, comment := range stage.Comments {
		f.writeComment(sb, comment)
	}

	// Write FROM instruction
	if stage.From != nil {
		f.writeFrom(sb, stage.From)
	}

	// Write other instructions
	for _, inst := range stage.Instructions {
		f.writeInstruction(sb, inst)
	}
}

// writeComment writes a comment
func (f *Formatter) writeComment(sb *strings.Builder, comment *parser.Comment) {
	sb.WriteString(comment.Text)
	sb.WriteString("\n")
}

// writeInstruction writes any instruction
func (f *Formatter) writeInstruction(sb *strings.Builder, inst parser.Instruction) {
	switch v := inst.(type) {
	case *parser.RunInstruction:
		f.writeRun(sb, v)
	case *parser.CopyInstruction:
		f.writeCopy(sb, v)
	case *parser.AddInstruction:
		f.writeAdd(sb, v)
	case *parser.EnvInstruction:
		f.writeEnv(sb, v)
	case *parser.ArgInstruction:
		f.writeArg(sb, v)
	case *parser.LabelInstruction:
		f.writeLabel(sb, v)
	case *parser.WorkdirInstruction:
		f.writeWorkdir(sb, v)
	case *parser.UserInstruction:
		f.writeUser(sb, v)
	case *parser.ExposeInstruction:
		f.writeExpose(sb, v)
	case *parser.VolumeInstruction:
		f.writeVolume(sb, v)
	case *parser.CmdInstruction:
		f.writeCmd(sb, v)
	case *parser.EntrypointInstruction:
		f.writeEntrypoint(sb, v)
	case *parser.HealthcheckInstruction:
		f.writeHealthcheck(sb, v)
	case *parser.ShellInstruction:
		f.writeShell(sb, v)
	case *parser.StopsignalInstruction:
		f.writeStopsignal(sb, v)
	case *parser.OnbuildInstruction:
		f.writeOnbuild(sb, v)
	case *parser.MaintainerInstruction:
		// Convert deprecated MAINTAINER to LABEL
		f.writeLabel(sb, &parser.LabelInstruction{
			Labels: []parser.KeyValue{{Key: "maintainer", Value: v.Maintainer}},
		})
	}
}

// writeFrom writes a FROM instruction
func (f *Formatter) writeFrom(sb *strings.Builder, from *parser.FromInstruction) {
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

// writeRun writes a RUN instruction
func (f *Formatter) writeRun(sb *strings.Builder, run *parser.RunInstruction) {
	sb.WriteString("RUN ")

	// Write flags
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
	if run.Security != "" {
		sb.WriteString("--security=")
		sb.WriteString(run.Security)
		sb.WriteString(" ")
	}

	if run.Heredoc != nil {
		sb.WriteString(run.Heredoc.Content)
	} else if run.IsExec {
		f.writeExecForm(sb, run.Arguments)
	} else {
		f.writeShellCommand(sb, run.Command)
	}

	sb.WriteString("\n")
}

// writeCopy writes a COPY instruction
func (f *Formatter) writeCopy(sb *strings.Builder, copy *parser.CopyInstruction) {
	sb.WriteString("COPY ")

	// Write flags
	if copy.From != "" {
		sb.WriteString("--from=")
		sb.WriteString(copy.From)
		sb.WriteString(" ")
	}
	if copy.Chown != "" {
		sb.WriteString("--chown=")
		sb.WriteString(copy.Chown)
		sb.WriteString(" ")
	}
	if copy.Chmod != "" {
		sb.WriteString("--chmod=")
		sb.WriteString(copy.Chmod)
		sb.WriteString(" ")
	}
	if copy.Link {
		sb.WriteString("--link ")
	}

	// Write sources and destination
	for _, src := range copy.Sources {
		sb.WriteString(f.quoteIfNeeded(src))
		sb.WriteString(" ")
	}
	sb.WriteString(f.quoteIfNeeded(copy.Destination))
	sb.WriteString("\n")
}

// writeAdd writes an ADD instruction
func (f *Formatter) writeAdd(sb *strings.Builder, add *parser.AddInstruction) {
	sb.WriteString("ADD ")

	// Write flags
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

	// Write sources and destination
	for _, src := range add.Sources {
		sb.WriteString(f.quoteIfNeeded(src))
		sb.WriteString(" ")
	}
	sb.WriteString(f.quoteIfNeeded(add.Destination))
	sb.WriteString("\n")
}

// writeEnv writes an ENV instruction with optional alignment
func (f *Formatter) writeEnv(sb *strings.Builder, env *parser.EnvInstruction) {
	sb.WriteString("ENV ")

	if !f.opts.AlignMultiValue || len(env.Variables) <= 1 {
		// Single line format
		for i, kv := range env.Variables {
			if i > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(kv.Key)
			sb.WriteString("=")
			sb.WriteString(f.quoteIfNeeded(kv.Value))
		}
		sb.WriteString("\n")
		return
	}

	// Multi-line aligned format
	maxKeyLen := 0
	for _, kv := range env.Variables {
		if len(kv.Key) > maxKeyLen {
			maxKeyLen = len(kv.Key)
		}
	}

	for i, kv := range env.Variables {
		if i > 0 {
			sb.WriteString(" \\\n")
			sb.WriteString(f.opts.IndentString)
		}
		sb.WriteString(kv.Key)
		sb.WriteString(strings.Repeat(" ", maxKeyLen-len(kv.Key)))
		sb.WriteString("=")
		sb.WriteString(f.quoteIfNeeded(kv.Value))
	}
	sb.WriteString("\n")
}

// writeArg writes an ARG instruction
func (f *Formatter) writeArg(sb *strings.Builder, arg *parser.ArgInstruction) {
	sb.WriteString("ARG ")
	sb.WriteString(arg.Name)
	if arg.HasDefault {
		sb.WriteString("=")
		sb.WriteString(f.quoteIfNeeded(arg.DefaultValue))
	}
	sb.WriteString("\n")
}

// writeLabel writes a LABEL instruction with optional alignment
func (f *Formatter) writeLabel(sb *strings.Builder, label *parser.LabelInstruction) {
	sb.WriteString("LABEL ")

	if !f.opts.AlignMultiValue || len(label.Labels) <= 1 {
		// Single line format
		for i, kv := range label.Labels {
			if i > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(f.quoteIfNeeded(kv.Key))
			sb.WriteString("=")
			sb.WriteString(f.quoteIfNeeded(kv.Value))
		}
		sb.WriteString("\n")
		return
	}

	// Multi-line aligned format
	maxKeyLen := 0
	for _, kv := range label.Labels {
		quoted := f.quoteIfNeeded(kv.Key)
		if len(quoted) > maxKeyLen {
			maxKeyLen = len(quoted)
		}
	}

	for i, kv := range label.Labels {
		if i > 0 {
			sb.WriteString(" \\\n")
			sb.WriteString(f.opts.IndentString)
		}
		quoted := f.quoteIfNeeded(kv.Key)
		sb.WriteString(quoted)
		sb.WriteString(strings.Repeat(" ", maxKeyLen-len(quoted)))
		sb.WriteString("=")
		sb.WriteString(f.quoteIfNeeded(kv.Value))
	}
	sb.WriteString("\n")
}

// writeWorkdir writes a WORKDIR instruction
func (f *Formatter) writeWorkdir(sb *strings.Builder, wd *parser.WorkdirInstruction) {
	sb.WriteString("WORKDIR ")
	sb.WriteString(wd.Path)
	sb.WriteString("\n")
}

// writeUser writes a USER instruction
func (f *Formatter) writeUser(sb *strings.Builder, user *parser.UserInstruction) {
	sb.WriteString("USER ")
	sb.WriteString(user.User)
	if user.Group != "" {
		sb.WriteString(":")
		sb.WriteString(user.Group)
	}
	sb.WriteString("\n")
}

// writeExpose writes an EXPOSE instruction
func (f *Formatter) writeExpose(sb *strings.Builder, expose *parser.ExposeInstruction) {
	sb.WriteString("EXPOSE ")
	for i, port := range expose.Ports {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(port.Port)
		if port.Protocol != "" && port.Protocol != "tcp" {
			sb.WriteString("/")
			sb.WriteString(port.Protocol)
		}
	}
	sb.WriteString("\n")
}

// writeVolume writes a VOLUME instruction
func (f *Formatter) writeVolume(sb *strings.Builder, vol *parser.VolumeInstruction) {
	sb.WriteString("VOLUME ")
	if len(vol.Paths) == 1 {
		sb.WriteString(f.quoteIfNeeded(vol.Paths[0]))
	} else {
		f.writeExecForm(sb, vol.Paths)
	}
	sb.WriteString("\n")
}

// writeCmd writes a CMD instruction
func (f *Formatter) writeCmd(sb *strings.Builder, cmd *parser.CmdInstruction) {
	sb.WriteString("CMD ")
	if cmd.IsExec {
		f.writeExecForm(sb, cmd.Arguments)
	} else {
		sb.WriteString(cmd.Command)
	}
	sb.WriteString("\n")
}

// writeEntrypoint writes an ENTRYPOINT instruction
func (f *Formatter) writeEntrypoint(sb *strings.Builder, ep *parser.EntrypointInstruction) {
	sb.WriteString("ENTRYPOINT ")
	if ep.IsExec {
		f.writeExecForm(sb, ep.Arguments)
	} else {
		sb.WriteString(ep.Command)
	}
	sb.WriteString("\n")
}

// writeHealthcheck writes a HEALTHCHECK instruction
func (f *Formatter) writeHealthcheck(sb *strings.Builder, hc *parser.HealthcheckInstruction) {
	sb.WriteString("HEALTHCHECK ")

	if hc.None {
		sb.WriteString("NONE")
		sb.WriteString("\n")
		return
	}

	// Write options
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
		f.writeExecForm(sb, hc.Arguments)
	} else {
		sb.WriteString(hc.Command)
	}
	sb.WriteString("\n")
}

// writeShell writes a SHELL instruction
func (f *Formatter) writeShell(sb *strings.Builder, shell *parser.ShellInstruction) {
	sb.WriteString("SHELL ")
	f.writeExecForm(sb, shell.Shell)
	sb.WriteString("\n")
}

// writeStopsignal writes a STOPSIGNAL instruction
func (f *Formatter) writeStopsignal(sb *strings.Builder, ss *parser.StopsignalInstruction) {
	sb.WriteString("STOPSIGNAL ")
	sb.WriteString(ss.Signal)
	sb.WriteString("\n")
}

// writeOnbuild writes an ONBUILD instruction
func (f *Formatter) writeOnbuild(sb *strings.Builder, ob *parser.OnbuildInstruction) {
	sb.WriteString("ONBUILD ")
	// Write the nested instruction inline
	var nested strings.Builder
	f.writeInstruction(&nested, ob.Instruction)
	sb.WriteString(strings.TrimSuffix(nested.String(), "\n"))
	sb.WriteString("\n")
}

// writeExecForm writes JSON exec form ["cmd", "arg1", "arg2"]
func (f *Formatter) writeExecForm(sb *strings.Builder, args []string) {
	sb.WriteString("[")
	for i, arg := range args {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("\"")
		sb.WriteString(escapeJSONString(arg))
		sb.WriteString("\"")
	}
	sb.WriteString("]")
}

// writeShellCommand writes a shell command with optional line continuation
func (f *Formatter) writeShellCommand(sb *strings.Builder, cmd string) {
	cmd = strings.TrimSpace(cmd)

	// Check if command has multiple parts
	if !strings.Contains(cmd, " && ") {
		sb.WriteString(cmd)
		return
	}

	// Split on && and format with line continuations
	parts := strings.Split(cmd, " && ")

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if i == 0 {
			sb.WriteString(part)
		} else {
			sb.WriteString(" \\\n")
			sb.WriteString(f.opts.IndentString)
			sb.WriteString("&& ")
			sb.WriteString(part)
		}
	}
}

// quoteIfNeeded adds quotes around a value if it contains special characters
func (f *Formatter) quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}

	// Check if quoting is needed
	needsQuotes := false
	for _, c := range s {
		if c == ' ' || c == '\t' || c == '"' || c == '\'' || c == '\\' || c == '$' || c == '=' {
			needsQuotes = true
			break
		}
	}

	if !needsQuotes {
		return s
	}

	// Use double quotes and escape
	return "\"" + escapeJSONString(s) + "\""
}

// escapeJSONString escapes a string for JSON/Dockerfile
func escapeJSONString(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString("\\\"")
		case '\\':
			sb.WriteString("\\\\")
		case '\n':
			sb.WriteString("\\n")
		case '\r':
			sb.WriteString("\\r")
		case '\t':
			sb.WriteString("\\t")
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// normalizeBlankLines removes excessive consecutive blank lines
func (f *Formatter) normalizeBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	blankCount := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= f.opts.MaxConsecutiveBlanks {
				result = append(result, line)
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
