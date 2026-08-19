package commands

import (
	"fmt"
	"strings"

	"github.com/dikee/hcode/internal/config"
	"github.com/dikee/hcode/internal/github"
	"github.com/dikee/hcode/internal/run"
	"github.com/dikee/hcode/internal/sshutil"
	"github.com/dikee/hcode/internal/state"
)

// RemoveOptions are every flag `hcode remove` accepts.
type RemoveOptions struct {
	RepoName     string
	InstanceName string
	Yes          bool
}

// Remove runs `hcode remove`.
func Remove(opts RemoveOptions) error {
	instance, err := state.Load(opts.InstanceName)
	if err != nil {
		return err
	}
	var match *state.Repo
	for i := range instance.Repos {
		if instance.Repos[i].Name == opts.RepoName {
			match = &instance.Repos[i]
			break
		}
	}
	if match == nil {
		available := reposList(instance)
		return run.Errorf("'%s' isn't on '%s'. On it: %s", opts.RepoName, opts.InstanceName, available)
	}

	if !opts.Yes {
		if err := confirm(fmt.Sprintf(
			"Delete the deploy key for %s/%s and remove it from '%s'? The box and every "+
				"other codebase on it stay up.",
			match.Owner, match.Name, opts.InstanceName,
		)); err != nil {
			return err
		}
	}

	if err := github.DeleteDeployKey(github.RepoRef{Owner: match.Owner, Name: match.Name}, match.DeployKeyID); err != nil {
		return err
	}
	cmd := fmt.Sprintf("rm -rf %s/%s %s/%s", config.RemoteCodeDir, opts.RepoName, config.RemoteKeyDir, opts.RepoName)
	if _, err := sshutil.RunRemote(instance.IP, cmd, instance.LoginKeyPath); err != nil {
		return err
	}

	kept := make([]state.Repo, 0, len(instance.Repos))
	for _, r := range instance.Repos {
		if r.Name != opts.RepoName {
			kept = append(kept, r)
		}
	}
	instance.Repos = kept
	if err := state.Save(instance); err != nil {
		return err
	}
	fmt.Printf("removed %s from %s\n", opts.RepoName, opts.InstanceName)
	return nil
}

func reposList(instance state.Instance) string {
	if len(instance.Repos) == 0 {
		return "(none)"
	}
	names := make([]string, len(instance.Repos))
	for i, r := range instance.Repos {
		names[i] = r.Name
	}
	return strings.Join(names, ", ")
}
