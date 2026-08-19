package main

import (
	"github.com/dikee/hetzner-code/internal/commands"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	opts := commands.AddOptions{}

	cmd := &cobra.Command{
		Use:   "add REPO_URL",
		Short: "Clone a second (third, ...) repo onto an already-running instance.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RepoURL = args[0]
			return commands.Add(opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.InstanceName, "instance", "", "Existing instance to add this repo to.")
	flags.StringVar(&opts.Branch, "branch", "", "Branch to check out. Default: repo's default branch.")
	flags.StringArrayVar(&opts.EnvFiles, "env-file", nil, "Local .env to copy up, repeatable.")
	cmd.MarkFlagRequired("instance")

	return cmd
}
