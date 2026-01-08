package parser

import (
	"fmt"
	"strings"

	"github.com/HueCodes/keel/internal/lexer"
)

// Parser parses Dockerfile tokens into an AST
type Parser struct {
	tokens  []lexer.Token
	pos     int
	current lexer.Token
	errors  []ParseError
}

// ParseError represents a parsing error
type ParseError struct {
	Message string
	Pos     lexer.Position
}

func (e ParseError) Error() string {
	return fmt.Sprintf("%s at %s", e.Message, e.Pos)
}

// New creates a new Parser
func New(tokens []lexer.Token) *Parser {
	p := &Parser{
		tokens: tokens,
		pos:    0,
	}
	if len(tokens) > 0 {
		p.current = tokens[0]
	}
	return p
}

// Parse parses the input and returns a Dockerfile AST
func Parse(input string) (*Dockerfile, []ParseError) {
	l := lexer.New(input)
	tokens := l.Tokenize()
	p := New(tokens)
	df := p.ParseDockerfile()
	return df, p.errors
}

// advance moves to the next token
func (p *Parser) advance() {
	p.pos++
	if p.pos < len(p.tokens) {
		p.current = p.tokens[p.pos]
	} else {
		p.current = lexer.Token{Type: lexer.TokenEOF}
	}
}

// peek returns the next token without advancing
func (p *Parser) peek() lexer.Token {
	if p.pos+1 < len(p.tokens) {
		return p.tokens[p.pos+1]
	}
	return lexer.Token{Type: lexer.TokenEOF}
}

// skipNewlines advances past any newline tokens
func (p *Parser) skipNewlines() {
	for p.current.Type == lexer.TokenNewline {
		p.advance()
	}
}

// skipComments advances past any comment tokens, collecting them
func (p *Parser) skipCommentsAndNewlines() []*Comment {
	var comments []*Comment
	for p.current.Type == lexer.TokenNewline || p.current.Type == lexer.TokenComment {
		if p.current.Type == lexer.TokenComment {
			comments = append(comments, &Comment{
				Text:     p.current.Literal,
				StartPos: p.current.Pos,
				EndPos:   p.current.EndPos,
			})
		}
		p.advance()
	}
	return comments
}

// error records a parsing error
func (p *Parser) error(msg string) {
	p.errors = append(p.errors, ParseError{
		Message: msg,
		Pos:     p.current.Pos,
	})
}

// ParseDockerfile parses a complete Dockerfile
func (p *Parser) ParseDockerfile() *Dockerfile {
	df := &Dockerfile{
		Escape: '\\',
	}

	if len(p.tokens) > 0 {
		df.StartPos = p.tokens[0].Pos
	}

	// Handle escape directive at the start
	if p.current.Type == lexer.TokenEscapeDirective {
		// Extract escape char from directive
		text := p.current.Literal
		if idx := strings.Index(text, "="); idx != -1 {
			rest := strings.TrimSpace(text[idx+1:])
			if len(rest) > 0 {
				df.Escape = rune(rest[0])
			}
		}
		p.advance()
	}

	// Collect initial comments
	df.Comments = p.skipCommentsAndNewlines()

	// Parse stages
	for p.current.Type != lexer.TokenEOF {
		if p.current.Type == lexer.TokenFrom {
			stage := p.parseStage()
			if stage != nil {
				df.Stages = append(df.Stages, stage)
			}
		} else if p.current.Type == lexer.TokenComment {
			df.Comments = append(df.Comments, &Comment{
				Text:     p.current.Literal,
				StartPos: p.current.Pos,
				EndPos:   p.current.EndPos,
			})
			p.advance()
		} else if p.current.Type == lexer.TokenNewline {
			p.advance()
		} else {
			// Instruction outside of stage - error but try to recover
			p.error("instruction outside of build stage")
			p.skipToNextInstruction()
		}
	}

	if len(p.tokens) > 0 {
		df.EndPos = p.tokens[len(p.tokens)-1].EndPos
	}

	return df
}

// parseStage parses a build stage starting with FROM
func (p *Parser) parseStage() *Stage {
	stage := &Stage{
		StartPos: p.current.Pos,
	}

	// Parse FROM instruction
	from := p.parseFrom()
	if from == nil {
		return nil
	}
	stage.From = from
	stage.Name = from.AsName

	// Parse instructions until next FROM or EOF
	for p.current.Type != lexer.TokenEOF && p.current.Type != lexer.TokenFrom {
		comments := p.skipCommentsAndNewlines()
		stage.Comments = append(stage.Comments, comments...)

		if p.current.Type == lexer.TokenEOF || p.current.Type == lexer.TokenFrom {
			break
		}

		inst := p.parseInstruction()
		if inst != nil {
			stage.Instructions = append(stage.Instructions, inst)
		}
	}

	if len(stage.Instructions) > 0 {
		stage.EndPos = stage.Instructions[len(stage.Instructions)-1].End()
	} else {
		stage.EndPos = stage.From.End()
	}

	return stage
}

// parseInstruction parses a single instruction
func (p *Parser) parseInstruction() Instruction {
	switch p.current.Type {
	case lexer.TokenFrom:
		return p.parseFrom()
	case lexer.TokenRun:
		return p.parseRun()
	case lexer.TokenCmd:
		return p.parseCmd()
	case lexer.TokenEntrypoint:
		return p.parseEntrypoint()
	case lexer.TokenCopy:
		return p.parseCopy()
	case lexer.TokenAdd:
		return p.parseAdd()
	case lexer.TokenEnv:
		return p.parseEnv()
	case lexer.TokenArg:
		return p.parseArg()
	case lexer.TokenLabel:
		return p.parseLabel()
	case lexer.TokenExpose:
		return p.parseExpose()
	case lexer.TokenVolume:
		return p.parseVolume()
	case lexer.TokenUser:
		return p.parseUser()
	case lexer.TokenWorkdir:
		return p.parseWorkdir()
	case lexer.TokenShell:
		return p.parseShell()
	case lexer.TokenHealthcheck:
		return p.parseHealthcheck()
	case lexer.TokenStopsignal:
		return p.parseStopsignal()
	case lexer.TokenOnbuild:
		return p.parseOnbuild()
	case lexer.TokenMaintainer:
		return p.parseMaintainer()
	default:
		p.error(fmt.Sprintf("unexpected token: %s", p.current.Type))
		p.skipToNextInstruction()
		return nil
	}
}

// skipToNextInstruction skips to the next line that starts with an instruction
func (p *Parser) skipToNextInstruction() {
	for p.current.Type != lexer.TokenEOF {
		if p.current.Type == lexer.TokenNewline {
			p.advance()
			if p.current.IsInstruction() {
				return
			}
		} else {
			p.advance()
		}
	}
}

// collectLine collects all tokens until newline or EOF
func (p *Parser) collectLine() []lexer.Token {
	var tokens []lexer.Token
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		tokens = append(tokens, p.current)
		p.advance()
	}
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}
	return tokens
}

// collectWords collects word and string tokens from line
func (p *Parser) collectWords() []string {
	var words []string
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		switch p.current.Type {
		case lexer.TokenWord, lexer.TokenString, lexer.TokenVariable:
			words = append(words, p.current.Literal)
		}
		p.advance()
	}
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}
	return words
}

// parseFrom parses FROM instruction
func (p *Parser) parseFrom() *FromInstruction {
	inst := &FromInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	startPos := p.pos
	p.advance() // consume FROM

	// Check for --platform flag
	if p.current.Type == lexer.TokenFlag {
		flag := p.current.Literal
		if strings.HasPrefix(flag, "--platform=") {
			inst.Platform = strings.TrimPrefix(flag, "--platform=")
		}
		p.advance()
	}

	// Parse image reference
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		switch p.current.Type {
		case lexer.TokenWord:
			word := p.current.Literal
			upperWord := strings.ToUpper(word)
			if upperWord == "AS" {
				p.advance()
				if p.current.Type == lexer.TokenWord {
					inst.AsName = p.current.Literal
					p.advance()
				}
			} else if inst.Image == "" {
				inst.Image = word
				p.advance()
			} else {
				p.advance()
			}
		case lexer.TokenColon:
			p.advance()
			if p.current.Type == lexer.TokenWord {
				inst.Tag = p.current.Literal
				p.advance()
			}
		case lexer.TokenAt:
			p.advance()
			if p.current.Type == lexer.TokenWord {
				inst.Digest = p.current.Literal
				p.advance()
			}
		case lexer.TokenVariable:
			// Image can be a variable
			if inst.Image == "" {
				inst.Image = p.current.Literal
			}
			p.advance()
		default:
			p.advance()
		}
	}

	// Build raw text
	endPos := p.pos
	if endPos > startPos && endPos <= len(p.tokens) {
		var parts []string
		for i := startPos; i < endPos && i < len(p.tokens); i++ {
			if p.tokens[i].Type != lexer.TokenNewline {
				parts = append(parts, p.tokens[i].Literal)
			}
		}
		inst.RawText = strings.Join(parts, " ")
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseRun parses RUN instruction
func (p *Parser) parseRun() *RunInstruction {
	inst := &RunInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume RUN

	// Check for flags
	for p.current.Type == lexer.TokenFlag {
		flag := p.current.Literal
		if strings.HasPrefix(flag, "--mount=") {
			inst.Mount = strings.TrimPrefix(flag, "--mount=")
		} else if strings.HasPrefix(flag, "--network=") {
			inst.Network = strings.TrimPrefix(flag, "--network=")
		} else if strings.HasPrefix(flag, "--security=") {
			inst.Security = strings.TrimPrefix(flag, "--security=")
		}
		p.advance()
	}

	// Check for heredoc
	if p.current.Type == lexer.TokenHeredoc {
		inst.Heredoc = &Heredoc{
			Content: p.current.Literal,
		}
		p.advance()
	} else if p.current.Type == lexer.TokenLeftBracket {
		// Exec form
		inst.IsExec = true
		inst.Arguments = p.parseExecForm()
	} else {
		// Shell form - collect rest of line
		inst.Command = p.collectRestOfLine()
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// collectRestOfLine collects the rest of the line as a string, preserving proper spacing
func (p *Parser) collectRestOfLine() string {
	var sb strings.Builder
	var lastEnd lexer.Position
	first := true

	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		if !first {
			// Add space only if there was whitespace between tokens in the source
			// If the current token starts right after the previous one ended, no space
			if p.current.Pos.Offset > lastEnd.Offset {
				sb.WriteString(" ")
			}
		}
		sb.WriteString(p.current.Literal)
		lastEnd = p.current.EndPos
		first = false
		p.advance()
	}
	return sb.String()
}

// collectRestOfLineRaw collects the rest of the line preserving original spacing
func (p *Parser) collectRestOfLineRaw() string {
	var parts []string
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		parts = append(parts, p.current.Literal)
		p.advance()
	}
	return strings.Join(parts, "")
}

// parseExecForm parses ["cmd", "arg", ...] form
func (p *Parser) parseExecForm() []string {
	var args []string
	p.advance() // consume [

	for p.current.Type != lexer.TokenRightBracket && p.current.Type != lexer.TokenEOF {
		if p.current.Type == lexer.TokenString {
			// Remove quotes
			s := p.current.Literal
			if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') {
				s = s[1 : len(s)-1]
			}
			args = append(args, s)
		}
		p.advance()
	}
	if p.current.Type == lexer.TokenRightBracket {
		p.advance()
	}
	return args
}

// parseCmd parses CMD instruction
func (p *Parser) parseCmd() *CmdInstruction {
	inst := &CmdInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume CMD

	if p.current.Type == lexer.TokenLeftBracket {
		inst.IsExec = true
		inst.Arguments = p.parseExecForm()
	} else {
		inst.Command = p.collectRestOfLine()
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseEntrypoint parses ENTRYPOINT instruction
func (p *Parser) parseEntrypoint() *EntrypointInstruction {
	inst := &EntrypointInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume ENTRYPOINT

	if p.current.Type == lexer.TokenLeftBracket {
		inst.IsExec = true
		inst.Arguments = p.parseExecForm()
	} else {
		inst.Command = p.collectRestOfLine()
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseCopy parses COPY instruction
func (p *Parser) parseCopy() *CopyInstruction {
	inst := &CopyInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume COPY

	// Parse flags
	for p.current.Type == lexer.TokenFlag {
		flag := p.current.Literal
		if strings.HasPrefix(flag, "--from=") {
			inst.From = strings.TrimPrefix(flag, "--from=")
		} else if strings.HasPrefix(flag, "--chown=") {
			inst.Chown = strings.TrimPrefix(flag, "--chown=")
		} else if strings.HasPrefix(flag, "--chmod=") {
			inst.Chmod = strings.TrimPrefix(flag, "--chmod=")
		} else if flag == "--link" {
			inst.Link = true
		}
		p.advance()
	}

	// Parse sources and destination
	var paths []string
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		if p.current.Type == lexer.TokenWord || p.current.Type == lexer.TokenString || p.current.Type == lexer.TokenVariable {
			path := p.current.Literal
			// Remove quotes if present
			if len(path) >= 2 && (path[0] == '"' || path[0] == '\'') {
				path = path[1 : len(path)-1]
			}
			paths = append(paths, path)
		}
		p.advance()
	}

	if len(paths) > 0 {
		inst.Destination = paths[len(paths)-1]
		inst.Sources = paths[:len(paths)-1]
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseAdd parses ADD instruction
func (p *Parser) parseAdd() *AddInstruction {
	inst := &AddInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume ADD

	// Parse flags
	for p.current.Type == lexer.TokenFlag {
		flag := p.current.Literal
		if strings.HasPrefix(flag, "--chown=") {
			inst.Chown = strings.TrimPrefix(flag, "--chown=")
		} else if strings.HasPrefix(flag, "--chmod=") {
			inst.Chmod = strings.TrimPrefix(flag, "--chmod=")
		} else if strings.HasPrefix(flag, "--checksum=") {
			inst.Checksum = strings.TrimPrefix(flag, "--checksum=")
		}
		p.advance()
	}

	// Parse sources and destination
	var paths []string
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		if p.current.Type == lexer.TokenWord || p.current.Type == lexer.TokenString {
			path := p.current.Literal
			if len(path) >= 2 && (path[0] == '"' || path[0] == '\'') {
				path = path[1 : len(path)-1]
			}
			paths = append(paths, path)
		}
		p.advance()
	}

	if len(paths) > 0 {
		inst.Destination = paths[len(paths)-1]
		inst.Sources = paths[:len(paths)-1]
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseEnv parses ENV instruction
func (p *Parser) parseEnv() *EnvInstruction {
	inst := &EnvInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume ENV

	// Parse key=value pairs
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		if p.current.Type == lexer.TokenWord {
			key := p.current.Literal
			p.advance()

			var value string
			if p.current.Type == lexer.TokenEquals {
				p.advance()
				if p.current.Type == lexer.TokenWord || p.current.Type == lexer.TokenString || p.current.Type == lexer.TokenVariable {
					value = p.current.Literal
					// Remove quotes
					if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
						value = value[1 : len(value)-1]
					}
					p.advance()
				}
			} else if p.current.Type == lexer.TokenWord || p.current.Type == lexer.TokenString {
				// Old syntax: ENV key value
				value = p.current.Literal
				if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
					value = value[1 : len(value)-1]
				}
				p.advance()
			}

			inst.Variables = append(inst.Variables, KeyValue{Key: key, Value: value})
		} else {
			p.advance()
		}
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseArg parses ARG instruction
func (p *Parser) parseArg() *ArgInstruction {
	inst := &ArgInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume ARG

	if p.current.Type == lexer.TokenWord {
		inst.Name = p.current.Literal
		p.advance()

		if p.current.Type == lexer.TokenEquals {
			p.advance()
			inst.HasDefault = true
			if p.current.Type == lexer.TokenWord || p.current.Type == lexer.TokenString {
				inst.DefaultValue = p.current.Literal
				if len(inst.DefaultValue) >= 2 && (inst.DefaultValue[0] == '"' || inst.DefaultValue[0] == '\'') {
					inst.DefaultValue = inst.DefaultValue[1 : len(inst.DefaultValue)-1]
				}
				p.advance()
			}
		}
	}

	// Skip rest of line
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		p.advance()
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseLabel parses LABEL instruction
func (p *Parser) parseLabel() *LabelInstruction {
	inst := &LabelInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume LABEL

	// Parse key=value pairs
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		if p.current.Type == lexer.TokenWord || p.current.Type == lexer.TokenString {
			key := p.current.Literal
			if len(key) >= 2 && (key[0] == '"' || key[0] == '\'') {
				key = key[1 : len(key)-1]
			}
			p.advance()

			var value string
			if p.current.Type == lexer.TokenEquals {
				p.advance()
				if p.current.Type == lexer.TokenWord || p.current.Type == lexer.TokenString {
					value = p.current.Literal
					if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
						value = value[1 : len(value)-1]
					}
					p.advance()
				}
			}

			inst.Labels = append(inst.Labels, KeyValue{Key: key, Value: value})
		} else {
			p.advance()
		}
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseExpose parses EXPOSE instruction
func (p *Parser) parseExpose() *ExposeInstruction {
	inst := &ExposeInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume EXPOSE

	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		if p.current.Type == lexer.TokenWord {
			portStr := p.current.Literal
			port := PortSpec{Port: portStr}

			// Check for protocol
			if strings.Contains(portStr, "/") {
				parts := strings.Split(portStr, "/")
				port.Port = parts[0]
				if len(parts) > 1 {
					port.Protocol = parts[1]
				}
			}

			inst.Ports = append(inst.Ports, port)
		}
		p.advance()
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseVolume parses VOLUME instruction
func (p *Parser) parseVolume() *VolumeInstruction {
	inst := &VolumeInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume VOLUME

	if p.current.Type == lexer.TokenLeftBracket {
		// JSON form
		inst.Paths = p.parseExecForm()
	} else {
		// Space-separated paths
		for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
			if p.current.Type == lexer.TokenWord || p.current.Type == lexer.TokenString {
				path := p.current.Literal
				if len(path) >= 2 && (path[0] == '"' || path[0] == '\'') {
					path = path[1 : len(path)-1]
				}
				inst.Paths = append(inst.Paths, path)
			}
			p.advance()
		}
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseUser parses USER instruction
func (p *Parser) parseUser() *UserInstruction {
	inst := &UserInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume USER

	if p.current.Type == lexer.TokenWord || p.current.Type == lexer.TokenVariable {
		userGroup := p.current.Literal
		if strings.Contains(userGroup, ":") {
			parts := strings.SplitN(userGroup, ":", 2)
			inst.User = parts[0]
			inst.Group = parts[1]
		} else {
			inst.User = userGroup
		}
		p.advance()

		// Check for :group
		if p.current.Type == lexer.TokenColon {
			p.advance()
			if p.current.Type == lexer.TokenWord {
				inst.Group = p.current.Literal
				p.advance()
			}
		}
	}

	// Skip rest of line
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		p.advance()
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseWorkdir parses WORKDIR instruction
func (p *Parser) parseWorkdir() *WorkdirInstruction {
	inst := &WorkdirInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume WORKDIR

	var parts []string
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		if p.current.Type == lexer.TokenWord || p.current.Type == lexer.TokenVariable || p.current.Type == lexer.TokenString {
			parts = append(parts, p.current.Literal)
		}
		p.advance()
	}
	inst.Path = strings.Join(parts, "")

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseShell parses SHELL instruction
func (p *Parser) parseShell() *ShellInstruction {
	inst := &ShellInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume SHELL

	if p.current.Type == lexer.TokenLeftBracket {
		inst.Shell = p.parseExecForm()
	}

	// Skip rest of line
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		p.advance()
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseHealthcheck parses HEALTHCHECK instruction
func (p *Parser) parseHealthcheck() *HealthcheckInstruction {
	inst := &HealthcheckInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume HEALTHCHECK

	// Check for NONE
	if p.current.Type == lexer.TokenWord && strings.ToUpper(p.current.Literal) == "NONE" {
		inst.None = true
		p.advance()
	} else {
		// Parse flags
		for p.current.Type == lexer.TokenFlag {
			flag := p.current.Literal
			if strings.HasPrefix(flag, "--interval=") {
				inst.Interval = strings.TrimPrefix(flag, "--interval=")
			} else if strings.HasPrefix(flag, "--timeout=") {
				inst.Timeout = strings.TrimPrefix(flag, "--timeout=")
			} else if strings.HasPrefix(flag, "--start-period=") {
				inst.StartPeriod = strings.TrimPrefix(flag, "--start-period=")
			} else if strings.HasPrefix(flag, "--retries=") {
				inst.Retries = strings.TrimPrefix(flag, "--retries=")
			}
			p.advance()
		}

		// Parse CMD (can be TokenCmd or a word "CMD")
		if p.current.Type == lexer.TokenCmd || (p.current.Type == lexer.TokenWord && strings.ToUpper(p.current.Literal) == "CMD") {
			p.advance()
			if p.current.Type == lexer.TokenLeftBracket {
				inst.IsExec = true
				inst.Arguments = p.parseExecForm()
			} else {
				inst.Command = p.collectRestOfLine()
			}
		}
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseStopsignal parses STOPSIGNAL instruction
func (p *Parser) parseStopsignal() *StopsignalInstruction {
	inst := &StopsignalInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume STOPSIGNAL

	if p.current.Type == lexer.TokenWord {
		inst.Signal = p.current.Literal
		p.advance()
	}

	// Skip rest of line
	for p.current.Type != lexer.TokenNewline && p.current.Type != lexer.TokenEOF {
		p.advance()
	}

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}

// parseOnbuild parses ONBUILD instruction
func (p *Parser) parseOnbuild() *OnbuildInstruction {
	inst := &OnbuildInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume ONBUILD

	// Parse nested instruction - might be a word token since we're not at line start
	if p.current.IsInstruction() {
		inst.Instruction = p.parseInstruction()
	} else if p.current.Type == lexer.TokenWord {
		// Check if the word is an instruction keyword
		keyword := strings.ToUpper(p.current.Literal)
		tokType := lexer.LookupKeyword(keyword)
		if tokType != lexer.TokenWord {
			// It's an instruction keyword, parse it
			// Temporarily update current token type for parsing
			p.current.Type = tokType
			inst.Instruction = p.parseInstruction()
		}
	}

	inst.EndPos = p.current.Pos
	return inst
}

// parseMaintainer parses MAINTAINER instruction
func (p *Parser) parseMaintainer() *MaintainerInstruction {
	inst := &MaintainerInstruction{
		BaseInstruction: BaseInstruction{
			StartPos: p.current.Pos,
		},
	}

	p.advance() // consume MAINTAINER

	inst.Maintainer = p.collectRestOfLineRaw()

	inst.EndPos = p.current.Pos
	if p.current.Type == lexer.TokenNewline {
		p.advance()
	}

	return inst
}
