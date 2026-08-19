package main

import (
	"github.com/dikee/hcode/internal/commands"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	opts := commands.RemoveOptions{}

	cmd := &cobra.Command{
		Use:   "remove REPO_NAME",
		Short: "Remove one codebase from an instance — deletes its deploy key and\nits clone directory, leaves the box and every other codebase up.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RepoName = args[0]
			return commands.Remove(opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.InstanceName, "instance", "", "Instance the repo is on.")
	flags.BoolVar(&opts.Yes, "yes", false, "Skip the confirmation prompt.")
	cmd.MarkFlagRequired("instance")

	return cmd
}
