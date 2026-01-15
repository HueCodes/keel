package lexer

import (
	"sync"
)

// TokenSlicePool provides a pool of token slices to reduce allocations
var TokenSlicePool = &tokenSlicePool{
	pool: sync.Pool{
		New: func() interface{} {
			// Pre-allocate for common Dockerfile size (64 tokens)
			s := make([]Token, 0, 64)
			return &s
		},
	},
}

type tokenSlicePool struct {
	pool sync.Pool
}

// Get retrieves a token slice from the pool
func (p *tokenSlicePool) Get() *[]Token {
	return p.pool.Get().(*[]Token)
}

// Put returns a token slice to the pool
func (p *tokenSlicePool) Put(s *[]Token) {
	// Reset length but keep capacity
	*s = (*s)[:0]
	p.pool.Put(s)
}

// LexerPool provides a pool of lexers to reduce allocations
var LexerPool = &lexerPool{
	pool: sync.Pool{
		New: func() interface{} {
			return &Lexer{}
		},
	},
}

type lexerPool struct {
	pool sync.Pool
}

// Get retrieves a lexer from the pool and initializes it with the given input
func (p *lexerPool) Get(input string) *Lexer {
	l := p.pool.Get().(*Lexer)
	l.Reset(input)
	return l
}

// Put returns a lexer to the pool
func (p *lexerPool) Put(l *Lexer) {
	// Clear references to allow GC
	l.input = ""
	p.pool.Put(l)
}

// Reset reinitializes the lexer with new input
func (l *Lexer) Reset(input string) {
	l.input = input
	l.pos = 0
	l.readPos = 0
	l.ch = 0
	l.line = 1
	l.column = 0
	l.startLine = 0
	l.startColumn = 0
	l.startOffset = 0
	l.escapeChar = '\\'
	l.atLineStart = true
	l.inInstruction = false
	l.readChar()
}

// TokenizePooled tokenizes input using pooled resources
// The returned slice must be returned to the pool when done via TokenSlicePool.Put
func TokenizePooled(input string) *[]Token {
	l := LexerPool.Get(input)
	defer LexerPool.Put(l)

	tokens := TokenSlicePool.Get()
	for {
		tok := l.NextToken()
		*tokens = append(*tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens
}
