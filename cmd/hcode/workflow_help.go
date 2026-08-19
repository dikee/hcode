package main

import (
	"fmt"

	"github.com/dikee/hcode/internal/workflowhelp"
	"github.com/spf13/cobra"
)

func newWorkflowHelpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "workflow-help",
		Short: "Print the multi-lane orchestrator + worker-lane workflow.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// workflowhelp.Text already ends in "\n" — click.echo (the
			// Python original) always appends one more, so match that
			// trailing blank line exactly.
			fmt.Print(workflowhelp.Text + "\n")
			return nil
		},
	}
}
