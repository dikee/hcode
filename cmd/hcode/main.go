// Command hcode manages disposable Hetzner boxes for Claude Code, each
// wired to a git repo through a per-repo GitHub deploy key.
package main

import (
	"fmt"
	"os"

	"github.com/dikee/hetzner-code/internal/run"
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "hcode",
		Short: "Disposable Hetzner boxes for Claude Code",
		Long: "Disposable Hetzner boxes for Claude Code, wired to a git repo via\n" +
			"a per-repo GitHub deploy key. Runs `hcloud` and `gh` under the hood —\n" +
			"both must already be authenticated.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newCreateCmd(),
		newAddCmd(),
		newRemoveCmd(),
		newDestroyCmd(),
		newStatusCmd(),
		newSSHCmd(),
		newPullCmd(),
		newWorkflowHelpCmd(),
	)
	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		if hErr, ok := err.(*run.HcodeError); ok {
			fmt.Fprintf(os.Stderr, "error: %s\n", hErr.Error())
		} else {
			fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		}
		os.Exit(1)
	}
}
