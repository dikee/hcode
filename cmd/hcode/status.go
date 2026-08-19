package main

import (
	"github.com/dikee/hetzner-code/internal/commands"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	opts := commands.StatusOptions{}

	cmd := &cobra.Command{
		Use:   "status [NAME]",
		Short: "List tracked instances, or show detail for one.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Name = args[0]
			}
			return commands.Status(opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.JSONOutput, "json", false, "Machine-readable output.")
	flags.BoolVar(&opts.Reconcile, "reconcile", false, "Cross-check against hcloud/GitHub for orphans.")

	return cmd
}
