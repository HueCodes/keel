package parser

import (
	"strings"

	"github.com/HueCodes/keel/internal/lexer"
)

// Node is the interface implemented by all AST nodes
type Node interface {
	Pos() lexer.Position
	End() lexer.Position
	node()
}

// Instruction is a Dockerfile instruction
type Instruction interface {
	Node
	instructionName() string
}

// Dockerfile represents a complete Dockerfile
type Dockerfile struct {
	Stages   []*Stage          // build stages
	Comments []*Comment        // top-level comments
	Escape   rune              // escape character (default \)
	StartPos lexer.Position
	EndPos   lexer.Position
}

func (d *Dockerfile) Pos() lexer.Position { return d.StartPos }
func (d *Dockerfile) End() lexer.Position { return d.EndPos }
func (d *Dockerfile) node()               {}

// Stage represents a build stage (FROM ... until next FROM or EOF)
type Stage struct {
	Name         string         // stage name (from AS clause)
	From         *FromInstruction
	Instructions []Instruction
	Comments     []*Comment
	StartPos     lexer.Position
	EndPos       lexer.Position
}

func (s *Stage) Pos() lexer.Position { return s.StartPos }
func (s *Stage) End() lexer.Position { return s.EndPos }
func (s *Stage) node()               {}

// Comment represents a comment line
type Comment struct {
	Text     string
	StartPos lexer.Position
	EndPos   lexer.Position
}

func (c *Comment) Pos() lexer.Position { return c.StartPos }
func (c *Comment) End() lexer.Position { return c.EndPos }
func (c *Comment) node()               {}

// BaseInstruction contains common instruction fields
type BaseInstruction struct {
	StartPos lexer.Position
	EndPos   lexer.Position
	RawText  string     // original text
	Comments []*Comment // inline comments
}

func (b *BaseInstruction) Pos() lexer.Position { return b.StartPos }
func (b *BaseInstruction) End() lexer.Position { return b.EndPos }
func (b *BaseInstruction) node()               {}

// FromInstruction represents FROM instruction
type FromInstruction struct {
	BaseInstruction
	Image    string // image name
	Tag      string // tag (after :)
	Digest   string // digest (after @)
	Platform string // --platform flag value
	AsName   string // AS name
}

func (f *FromInstruction) instructionName() string { return "FROM" }

// ImageRef returns the full image reference
func (f *FromInstruction) ImageRef() string {
	ref := f.Image
	if f.Tag != "" {
		ref += ":" + f.Tag
	}
	if f.Digest != "" {
		ref += "@" + f.Digest
	}
	return ref
}

// RunInstruction represents RUN instruction
type RunInstruction struct {
	BaseInstruction
	Command   string   // shell form command
	Arguments []string // exec form arguments
	IsExec    bool     // true if exec form ["cmd", "arg"]
	Heredoc   *Heredoc // heredoc content if present
	Mount     string   // --mount flag
	Network   string   // --network flag
	Security  string   // --security flag
}

func (r *RunInstruction) instructionName() string { return "RUN" }

// Heredoc represents heredoc content in RUN instructions
type Heredoc struct {
	Delimiter string
	Content   string
	StripTabs bool
}

// CmdInstruction represents CMD instruction
type CmdInstruction struct {
	BaseInstruction
	Command   string   // shell form
	Arguments []string // exec form
	IsExec    bool
}

func (c *CmdInstruction) instructionName() string { return "CMD" }

// EntrypointInstruction represents ENTRYPOINT instruction
type EntrypointInstruction struct {
	BaseInstruction
	Command   string
	Arguments []string
	IsExec    bool
}

func (e *EntrypointInstruction) instructionName() string { return "ENTRYPOINT" }

// CopyInstruction represents COPY instruction
type CopyInstruction struct {
	BaseInstruction
	Sources     []string
	Destination string
	From        string // --from flag
	Chown       string // --chown flag
	Chmod       string // --chmod flag
	Link        bool   // --link flag
}

func (c *CopyInstruction) instructionName() string { return "COPY" }

// AddInstruction represents ADD instruction
type AddInstruction struct {
	BaseInstruction
	Sources     []string
	Destination string
	Chown       string
	Chmod       string
	Checksum    string // --checksum flag
}

func (a *AddInstruction) instructionName() string { return "ADD" }

// EnvInstruction represents ENV instruction
type EnvInstruction struct {
	BaseInstruction
	Variables []KeyValue
}

func (e *EnvInstruction) instructionName() string { return "ENV" }

// ArgInstruction represents ARG instruction
type ArgInstruction struct {
	BaseInstruction
	Name         string
	DefaultValue string
	HasDefault   bool
}

func (a *ArgInstruction) instructionName() string { return "ARG" }

// LabelInstruction represents LABEL instruction
type LabelInstruction struct {
	BaseInstruction
	Labels []KeyValue
}

func (l *LabelInstruction) instructionName() string { return "LABEL" }

// KeyValue represents a key=value pair
type KeyValue struct {
	Key   string
	Value string
}

// ExposeInstruction represents EXPOSE instruction
type ExposeInstruction struct {
	BaseInstruction
	Ports []PortSpec
}

func (e *ExposeInstruction) instructionName() string { return "EXPOSE" }

// PortSpec represents a port specification
type PortSpec struct {
	Port     string
	Protocol string // tcp or udp
}

// VolumeInstruction represents VOLUME instruction
type VolumeInstruction struct {
	BaseInstruction
	Paths []string
}

func (v *VolumeInstruction) instructionName() string { return "VOLUME" }

// UserInstruction represents USER instruction
type UserInstruction struct {
	BaseInstruction
	User  string
	Group string
}

func (u *UserInstruction) instructionName() string { return "USER" }

// WorkdirInstruction represents WORKDIR instruction
type WorkdirInstruction struct {
	BaseInstruction
	Path string
}

func (w *WorkdirInstruction) instructionName() string { return "WORKDIR" }

// ShellInstruction represents SHELL instruction
type ShellInstruction struct {
	BaseInstruction
	Shell []string
}

func (s *ShellInstruction) instructionName() string { return "SHELL" }

// HealthcheckInstruction represents HEALTHCHECK instruction
type HealthcheckInstruction struct {
	BaseInstruction
	None        bool   // HEALTHCHECK NONE
	Interval    string // --interval
	Timeout     string // --timeout
	StartPeriod string // --start-period
	Retries     string // --retries
	Command     string // CMD command
	Arguments   []string
	IsExec      bool
}

func (h *HealthcheckInstruction) instructionName() string { return "HEALTHCHECK" }

// StopsignalInstruction represents STOPSIGNAL instruction
type StopsignalInstruction struct {
	BaseInstruction
	Signal string
}

func (s *StopsignalInstruction) instructionName() string { return "STOPSIGNAL" }

// OnbuildInstruction represents ONBUILD instruction
type OnbuildInstruction struct {
	BaseInstruction
	Instruction Instruction // nested instruction
}

func (o *OnbuildInstruction) instructionName() string { return "ONBUILD" }

// MaintainerInstruction represents deprecated MAINTAINER instruction
type MaintainerInstruction struct {
	BaseInstruction
	Maintainer string
}

func (m *MaintainerInstruction) instructionName() string { return "MAINTAINER" }

// Visitor interface for AST traversal
type Visitor interface {
	VisitDockerfile(*Dockerfile) bool
	VisitStage(*Stage) bool
	VisitInstruction(Instruction) bool
}

// Walk traverses the AST calling visitor methods
func Walk(v Visitor, node Node) {
	switch n := node.(type) {
	case *Dockerfile:
		if !v.VisitDockerfile(n) {
			return
		}
		for _, stage := range n.Stages {
			Walk(v, stage)
		}
	case *Stage:
		if !v.VisitStage(n) {
			return
		}
		if n.From != nil {
			v.VisitInstruction(n.From)
		}
		for _, inst := range n.Instructions {
			v.VisitInstruction(inst)
		}
	}
}

// InstructionName returns the name of an instruction
func InstructionName(inst Instruction) string {
	return inst.instructionName()
}

// GetInstructions returns all instructions of a specific type from a Dockerfile
func GetInstructions[T Instruction](df *Dockerfile) []T {
	var result []T
	for _, stage := range df.Stages {
		if from, ok := any(stage.From).(T); ok {
			result = append(result, from)
		}
		for _, inst := range stage.Instructions {
			if typed, ok := inst.(T); ok {
				result = append(result, typed)
			}
		}
	}
	return result
}

// HasInstruction returns true if the Dockerfile contains the specified instruction type
func HasInstruction[T Instruction](df *Dockerfile) bool {
	for _, stage := range df.Stages {
		if _, ok := any(stage.From).(T); ok {
			return true
		}
		for _, inst := range stage.Instructions {
			if _, ok := inst.(T); ok {
				return true
			}
		}
	}
	return false
}

// IsPrivilegedPort returns true if the port is below 1024
func (p PortSpec) IsPrivilegedPort() bool {
	port := strings.TrimSuffix(p.Port, "/tcp")
	port = strings.TrimSuffix(port, "/udp")
	// Check if it's a range
	if strings.Contains(port, "-") {
		parts := strings.Split(port, "-")
		if len(parts) == 2 {
			port = parts[0]
		}
	}
	// Parse the port number
	if len(port) == 0 {
		return false
	}
	val := 0
	for _, c := range port {
		if c >= '0' && c <= '9' {
			val = val*10 + int(c-'0')
		} else {
			return false
		}
	}
	return val > 0 && val < 1024
}
