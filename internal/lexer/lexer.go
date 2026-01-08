package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lexer tokenizes Dockerfile content
type Lexer struct {
	input        string
	pos          int  // current position in input (points to current char)
	readPos      int  // current reading position (after current char)
	ch           rune // current character
	line         int  // current line number (1-based)
	column       int  // current column number (1-based)
	startLine    int  // line at start of current token
	startColumn  int  // column at start of current token
	startOffset  int  // offset at start of current token
	escapeChar   rune // escape character (default \)
	atLineStart  bool // true if at the start of a line (for instruction detection)
	inInstruction bool // true if we're parsing instruction arguments
}

// New creates a new Lexer for the given input
func New(input string) *Lexer {
	l := &Lexer{
		input:       input,
		line:        1,
		column:      0,
		escapeChar:  '\\',
		atLineStart: true,
	}
	l.readChar()
	return l
}

// readChar reads the next character
func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		l.ch, _ = utf8.DecodeRuneInString(l.input[l.readPos:])
	}
	l.pos = l.readPos
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
	l.readPos += utf8.RuneLen(l.ch)
}

// peekChar returns the next character without advancing
func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
	return r
}

// peekAhead returns the character n positions ahead
func (l *Lexer) peekAhead(n int) rune {
	pos := l.readPos
	for i := 0; i < n-1; i++ {
		if pos >= len(l.input) {
			return 0
		}
		_, size := utf8.DecodeRuneInString(l.input[pos:])
		pos += size
	}
	if pos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[pos:])
	return r
}

// markStart marks the start position for the current token
func (l *Lexer) markStart() {
	l.startLine = l.line
	l.startColumn = l.column
	l.startOffset = l.pos
}

// makeToken creates a token with the current position info
func (l *Lexer) makeToken(typ TokenType, literal string) Token {
	return Token{
		Type:    typ,
		Literal: literal,
		Pos: Position{
			Line:   l.startLine,
			Column: l.startColumn,
			Offset: l.startOffset,
		},
		EndPos: Position{
			Line:   l.line,
			Column: l.column,
			Offset: l.pos,
		},
	}
}

// NextToken returns the next token
func (l *Lexer) NextToken() Token {
	// Skip whitespace (but not newlines)
	l.skipWhitespace()

	l.markStart()

	// Check for EOF
	if l.ch == 0 {
		return l.makeToken(TokenEOF, "")
	}

	// Check for newline
	if l.ch == '\n' {
		l.readChar()
		l.atLineStart = true
		l.inInstruction = false
		return l.makeToken(TokenNewline, "\n")
	}

	// Check for comment (# at start of meaningful content on a line)
	if l.ch == '#' {
		return l.readComment()
	}

	// Check for line continuation
	if l.ch == l.escapeChar && l.peekChar() == '\n' {
		l.readChar() // consume escape
		l.readChar() // consume newline
		return l.NextToken() // continue to next meaningful token
	}

	// Check for heredoc start
	if l.ch == '<' && l.peekChar() == '<' {
		return l.readHeredocStart()
	}

	// Single character tokens
	switch l.ch {
	case '=':
		l.readChar()
		return l.makeToken(TokenEquals, "=")
	case ':':
		l.readChar()
		return l.makeToken(TokenColon, ":")
	case '@':
		l.readChar()
		return l.makeToken(TokenAt, "@")
	case ',':
		l.readChar()
		return l.makeToken(TokenComma, ",")
	case '[':
		l.readChar()
		return l.makeToken(TokenLeftBracket, "[")
	case ']':
		l.readChar()
		return l.makeToken(TokenRightBracket, "]")
	case l.escapeChar:
		l.readChar()
		return l.makeToken(TokenBackslash, string(l.escapeChar))
	}

	// Quoted strings
	if l.ch == '"' || l.ch == '\'' {
		return l.readString()
	}

	// Variable
	if l.ch == '$' {
		return l.readVariable()
	}

	// Flag (--something)
	if l.ch == '-' && l.peekChar() == '-' {
		return l.readFlag()
	}

	// Word or instruction
	if isWordChar(l.ch) {
		return l.readWord()
	}

	// Unknown character - return as word
	ch := l.ch
	l.readChar()
	return l.makeToken(TokenWord, string(ch))
}

// skipWhitespace skips spaces and tabs (not newlines)
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

// readComment reads a comment until end of line
func (l *Lexer) readComment() Token {
	l.markStart()
	start := l.pos

	// Check for escape directive: # escape=X or # escape = X
	if l.line == 1 || (l.line == 2 && l.startLine == 1) {
		// Could be escape directive
		savedPos := l.pos
		savedReadPos := l.readPos
		savedLine := l.line
		savedColumn := l.column
		savedCh := l.ch

		l.readChar() // skip #
		l.skipWhitespace()

		// Check for "escape"
		word := l.readBareWord()
		if strings.ToLower(word) == "escape" {
			l.skipWhitespace()
			if l.ch == '=' {
				l.readChar()
				l.skipWhitespace()
				if l.ch != 0 && l.ch != '\n' {
					escapeChar := l.ch
					l.escapeChar = escapeChar
					// Read to end of line
					for l.ch != 0 && l.ch != '\n' {
						l.readChar()
					}
					return l.makeToken(TokenEscapeDirective, l.input[start:l.pos])
				}
			}
		}

		// Not an escape directive, restore position
		l.pos = savedPos
		l.readPos = savedReadPos
		l.line = savedLine
		l.column = savedColumn
		l.ch = savedCh
	}

	// Regular comment
	for l.ch != 0 && l.ch != '\n' {
		l.readChar()
	}
	return l.makeToken(TokenComment, l.input[start:l.pos])
}

// readBareWord reads a word without creating a token
func (l *Lexer) readBareWord() string {
	start := l.pos
	for isWordChar(l.ch) {
		l.readChar()
	}
	return l.input[start:l.pos]
}

// readWord reads a word or instruction keyword
func (l *Lexer) readWord() Token {
	start := l.pos
	for isWordChar(l.ch) {
		l.readChar()
	}
	literal := l.input[start:l.pos]

	// Check if it's an instruction keyword (only at line start)
	if l.atLineStart {
		upper := strings.ToUpper(literal)
		if tokType := LookupKeyword(upper); tokType != TokenWord {
			l.atLineStart = false
			l.inInstruction = true
			return l.makeToken(tokType, literal)
		}
	}

	l.atLineStart = false
	return l.makeToken(TokenWord, literal)
}

// readString reads a quoted string
func (l *Lexer) readString() Token {
	quote := l.ch
	start := l.pos
	l.readChar() // consume opening quote

	for l.ch != 0 {
		if l.ch == l.escapeChar && (l.peekChar() == quote || l.peekChar() == l.escapeChar) {
			l.readChar() // skip escape
			l.readChar() // skip escaped char
			continue
		}
		if l.ch == quote {
			l.readChar() // consume closing quote
			break
		}
		if l.ch == '\n' {
			// Unterminated string, stop here
			break
		}
		l.readChar()
	}

	return l.makeToken(TokenString, l.input[start:l.pos])
}

// readVariable reads a variable reference ($VAR or ${VAR})
func (l *Lexer) readVariable() Token {
	start := l.pos
	l.readChar() // consume $

	if l.ch == '{' {
		// ${VAR} or ${VAR:-default} or ${VAR:+alternative}
		l.readChar() // consume {
		depth := 1
		for l.ch != 0 && depth > 0 {
			if l.ch == '{' {
				depth++
			} else if l.ch == '}' {
				depth--
			}
			l.readChar()
		}
	} else {
		// $VAR
		for isVarChar(l.ch) {
			l.readChar()
		}
	}

	return l.makeToken(TokenVariable, l.input[start:l.pos])
}

// readFlag reads a flag (--name or --name=value)
func (l *Lexer) readFlag() Token {
	start := l.pos
	l.readChar() // consume first -
	l.readChar() // consume second -

	// Read flag name
	for isWordChar(l.ch) || l.ch == '-' {
		l.readChar()
	}

	// Check for =value
	if l.ch == '=' {
		l.readChar() // consume =
		// Read value (could be quoted)
		if l.ch == '"' || l.ch == '\'' {
			quote := l.ch
			l.readChar()
			for l.ch != 0 && l.ch != quote && l.ch != '\n' {
				if l.ch == l.escapeChar {
					l.readChar()
				}
				l.readChar()
			}
			if l.ch == quote {
				l.readChar()
			}
		} else {
			// Unquoted value
			for l.ch != 0 && l.ch != ' ' && l.ch != '\t' && l.ch != '\n' {
				l.readChar()
			}
		}
	}

	return l.makeToken(TokenFlag, l.input[start:l.pos])
}

// readHeredocStart reads the start of a heredoc (<<EOF or <<-EOF)
func (l *Lexer) readHeredocStart() Token {
	start := l.pos
	l.readChar() // consume first <
	l.readChar() // consume second <

	// Check for - (strip leading tabs)
	stripTabs := false
	if l.ch == '-' {
		stripTabs = true
		l.readChar()
	}

	// Read delimiter
	var delimiter string
	if l.ch == '"' || l.ch == '\'' {
		// Quoted delimiter (no variable expansion)
		quote := l.ch
		l.readChar()
		delimStart := l.pos
		for l.ch != 0 && l.ch != quote && l.ch != '\n' {
			l.readChar()
		}
		delimiter = l.input[delimStart:l.pos]
		if l.ch == quote {
			l.readChar()
		}
	} else {
		// Unquoted delimiter
		delimStart := l.pos
		for isWordChar(l.ch) {
			l.readChar()
		}
		delimiter = l.input[delimStart:l.pos]
	}

	// Now read the heredoc content until we find the delimiter
	// First, consume the rest of the line (heredoc starts on next line)
	for l.ch != 0 && l.ch != '\n' {
		l.readChar()
	}
	if l.ch == '\n' {
		l.readChar()
	}

	// Read until we find delimiter on its own line
	contentStart := l.pos
	for l.ch != 0 {
		lineStart := l.pos
		// Skip leading whitespace if stripTabs
		if stripTabs {
			for l.ch == '\t' {
				l.readChar()
			}
		}

		// Check if this line is the delimiter
		wordStart := l.pos
		for isWordChar(l.ch) {
			l.readChar()
		}
		word := l.input[wordStart:l.pos]

		// Skip trailing whitespace
		for l.ch == ' ' || l.ch == '\t' {
			l.readChar()
		}

		if word == delimiter && (l.ch == '\n' || l.ch == 0) {
			// Found the end delimiter
			// Content is everything before this line
			content := l.input[contentStart:lineStart]
			if l.ch == '\n' {
				l.readChar()
			}
			_ = content // heredoc content captured
			break
		}

		// Not the delimiter, continue to end of line
		for l.ch != 0 && l.ch != '\n' {
			l.readChar()
		}
		if l.ch == '\n' {
			l.readChar()
		}
	}

	return l.makeToken(TokenHeredoc, l.input[start:l.pos])
}

// Tokenize returns all tokens from the input
func (l *Lexer) Tokenize() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens
}

// isWordChar returns true if r can be part of a word
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || r == '/'
}

// isVarChar returns true if r can be part of a variable name
func isVarChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
