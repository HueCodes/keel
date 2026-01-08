package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate default config file",
		Long:  "Generate a default .keel.yaml configuration file in the current directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := ".keel.yaml"

			if _, err := os.Stat(configFile); err == nil {
				return fmt.Errorf("%s already exists", configFile)
			}

			config := `# Keel configuration file
# See https://github.com/HueCodes/keel for documentation

# Minimum severity to report: error, warning, info, hint
severity: warning

# Rules configuration
rules:
  # Security rules
  SEC001:
    enabled: true
  SEC002:
    enabled: true
  SEC003:
    enabled: true
    # allowed_tags:
    #   - "latest"  # Allow latest for specific images

  # Performance rules
  PERF001:
    enabled: true
  PERF004:
    enabled: true
    max_consecutive: 3  # Warn if more than 3 consecutive RUN instructions

  # Best practice rules
  BP001:
    enabled: true
  BP002:
    enabled: true

  # Style rules
  STY001:
    enabled: true

# Ignore patterns (glob syntax)
ignore_paths:
  - "test/**"
  - "examples/**"

# Output format configuration
format:
  max_line_length: 120
  indent: 4
`

			if err := os.WriteFile(configFile, []byte(config), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", configFile, err)
			}

			fmt.Printf("Created %s\n", configFile)
			return nil
		},
	}

	return cmd
}
