package reporter

import (
	"io"

	"github.com/HueCodes/keel/internal/analyzer"
)

// Reporter is the interface for outputting analysis results
type Reporter interface {
	Report(result *analyzer.Result, source string) error
}

// Format represents the output format
type Format string

const (
	FormatTerminal Format = "terminal"
	FormatJSON     Format = "json"
	FormatSARIF    Format = "sarif"
	FormatMarkdown Format = "markdown"
	FormatGitHub   Format = "github"
)

// New creates a reporter for the given format
func New(format Format, w io.Writer, opts ...Option) Reporter {
	cfg := &Config{
		Writer:    w,
		UseColors: true,
		Verbose:   false,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	switch format {
	case FormatJSON:
		return &JSONReporter{cfg: cfg}
	case FormatSARIF:
		return &SARIFReporter{cfg: cfg}
	case FormatMarkdown:
		return &MarkdownReporter{cfg: cfg}
	case FormatGitHub:
		return &GitHubReporter{cfg: cfg}
	default:
		return &TerminalReporter{cfg: cfg}
	}
}

// Config holds reporter configuration
type Config struct {
	Writer    io.Writer
	UseColors bool
	Verbose   bool
}

// Option is a function that configures a reporter
type Option func(*Config)

// WithColors enables or disables colors
func WithColors(enabled bool) Option {
	return func(c *Config) {
		c.UseColors = enabled
	}
}

// WithVerbose enables verbose output
func WithVerbose(enabled bool) Option {
	return func(c *Config) {
		c.Verbose = enabled
	}
}
