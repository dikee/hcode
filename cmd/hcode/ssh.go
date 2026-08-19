package main

import (
	"os"

	"github.com/dikee/hetzner-code/internal/commands"
	"github.com/spf13/cobra"
)

func newSSHCmd() *cobra.Command {
	opts := commands.SSHOptions{}

	cmd := &cobra.Command{
		Use:   "ssh NAME",
		Short: "SSH into a tracked instance.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.InstanceName = args[0]
			exitCode, err := commands.SSH(opts)
			if err != nil {
				return err
			}
			if exitCode != 0 {
				os.Exit(exitCode)
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.RepoName, "repo", "", "cd into this repo's directory on connect.")
	flags.StringVar(&opts.Worktree, "worktree", "", "cd into this worktree (e.g. cc2) instead of the repo's main clone.")
	flags.StringArrayVarP(&opts.Forwards, "forward", "L", nil, "Forward a local port to the box, repeatable. Accepts ssh's own "+
		"PORT:HOST:PORT syntax, or shorthand: 8000 (same port both ends), 8000:9000 (different remote port).")

	return cmd
}
