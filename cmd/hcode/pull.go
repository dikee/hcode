package main

import (
	"github.com/dikee/hetzner-code/internal/commands"
	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	opts := commands.PullOptions{}

	cmd := &cobra.Command{
		Use:   "pull NAME REMOTE_PATH [LOCAL_PATH]",
		Short: "Copy a file or directory back from an instance. LOCAL_PATH\ndefaults to REMOTE_PATH's basename in the current directory.",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.InstanceName = args[0]
			opts.RemotePath = args[1]
			if len(args) == 3 {
				opts.LocalPath = args[2]
			}
			return commands.Pull(opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.RepoName, "repo", "", "Resolve REMOTE_PATH relative to this repo's clone dir instead of an absolute path.")

	return cmd
}
