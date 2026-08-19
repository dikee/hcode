package main

import (
	"github.com/dikee/hetzner-code/internal/commands"
	"github.com/spf13/cobra"
)

func newDestroyCmd() *cobra.Command {
	opts := commands.DestroyOptions{}

	cmd := &cobra.Command{
		Use:   "destroy [NAME]",
		Short: "Destroy an instance: the box, and every deploy key it holds.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Name = args[0]
			}
			return commands.Destroy(opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.All, "all", false, "Destroy every tracked instance.")
	flags.BoolVar(&opts.KeepKey, "keep-key", false, "Delete the box but leave every deploy key on GitHub.")
	flags.BoolVar(&opts.Yes, "yes", false, "Skip the confirmation prompt.")

	return cmd
}
