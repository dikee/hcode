package main

import (
	"github.com/dikee/hetzner-code/internal/commands"
	"github.com/dikee/hetzner-code/internal/config"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	opts := commands.CreateOptions{
		ServerType:   config.DefaultType,
		Location:     config.DefaultLocation,
		LoginKeyPath: "~/.ssh/id_ed25519",
	}

	cmd := &cobra.Command{
		Use:   "create REPO_URL",
		Short: "Create a box and clone REPO_URL onto it.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RepoURL = args[0]
			return commands.Create(opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Name, "name", "", "Instance name. Default: <repo>-<random>.")
	flags.StringVar(&opts.Branch, "branch", "", "Branch to check out. Default: repo's default branch.")
	flags.StringVar(&opts.ServerType, "type", opts.ServerType, "hcloud server type.")
	flags.StringVar(&opts.Location, "location", opts.Location, "hcloud location.")
	flags.StringVar(&opts.LoginKey, "login-key", "", "hcloud SSH key name for your own access. Default: first on the account.")
	flags.StringVar(&opts.LoginKeyPath, "login-key-path", opts.LoginKeyPath, "Local private key matching --login-key, used to actually SSH in.")
	flags.StringArrayVar(&opts.EnvFiles, "env-file", nil, "Local .env to copy up, repeatable. Path is relative to the repo root on both ends.")
	flags.IntVar(&opts.Worktrees, "worktrees", 0, "Add N extra git worktrees (cc2, cc3, ...) for parallel lanes on their own branches.")
	flags.StringVar(&opts.OpsDir, "ops-dir", "", "Local directory to copy up as a sibling of the repo/worktrees (e.g. an "+
		"orchestration mailbox folder) — never inside any single worktree, so every lane can see it.")
	flags.StringVar(&opts.PostClone, "post-clone", "", "A script inside the repo (path relative to its root, e.g. bin/bootstrap.sh) "+
		"to run once, after cloning — for repo-specific setup (installing a database server, running migrations) "+
		"that's the repo's business, not hcode's. Runs with output streamed live, since these can take a while.")
	flags.StringVar(&opts.PostWorktree, "post-worktree", "", "A script (path relative to each worktree's root) to run once per worktree, "+
		"after --env-files are copied into it. HCODE_WORKTREE_LABEL (cc2, cc3, ...) is set in its environment, "+
		"so the repo's own script can do lane-specific setup (e.g. a per-lane database) that hcode has no business knowing about.")
	flags.StringArrayVarP(&opts.Forwards, "forward", "L", nil, "Forward a local port to the box on the post-create SSH session, repeatable. "+
		"Accepts ssh's own PORT:HOST:PORT syntax, or shorthand: 8000 (same port both ends), 8000:9000 (different remote port). "+
		"No effect with --no-attach.")
	flags.BoolVar(&opts.NoAttach, "no-attach", false, "Don't SSH in after creation; just print connection info.")

	return cmd
}
