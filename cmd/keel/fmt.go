package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func fmtCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "fmt [file]",
		Short: "Format Dockerfile (style only)",
		Long:  "Format a Dockerfile for consistent style without making semantic changes.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				file = args[0]
			}
			if file == "" {
				file = "Dockerfile"
			}

			// TODO: Implement formatter
			fmt.Printf("Would format %s here...\n", file)
			fmt.Println("Fmt command not yet implemented.")
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Dockerfile path (default \"Dockerfile\")")

	return cmd
}
