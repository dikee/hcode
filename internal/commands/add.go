package commands

import (
	"fmt"
	"strings"

	"github.com/dikee/hcode/internal/attach"
	"github.com/dikee/hcode/internal/github"
	"github.com/dikee/hcode/internal/run"
	"github.com/dikee/hcode/internal/state"
)

// AddOptions are every flag `hcode add` accepts.
type AddOptions struct {
	RepoURL      string
	InstanceName string
	Branch       string
	EnvFiles     []string
}

// Add runs `hcode add`.
func Add(opts AddOptions) error {
	instance, err := state.Load(opts.InstanceName)
	if err != nil {
		return err
	}
	repo, err := github.ParseRepoURL(opts.RepoURL)
	if err != nil {
		return err
	}

	for _, r := range instance.Repos {
		if r.Name == repo.Name {
			return run.Errorf("'%s' is already on instance '%s'", repo.Name, opts.InstanceName)
		}
	}

	if len(instance.Repos) > 0 {
		names := make([]string, len(instance.Repos))
		for i, r := range instance.Repos {
			names[i] = r.Name
		}
		fmt.Printf(
			"note: '%s' already runs [%s] — adding another codebase means they share this "+
				"box's CPU. Your call; `hcode status` shows the type.\n",
			opts.InstanceName, strings.Join(names, ", "),
		)
	}

	fmt.Printf("attaching %s to '%s' (%s) ...\n", repo.Slug(), opts.InstanceName, instance.IP)
	var branchPtr *string
	if opts.Branch != "" {
		branchPtr = &opts.Branch
	}
	repoState, err := attach.AttachRepo(instance.IP, instance.LoginKeyPath, opts.InstanceName, opts.RepoURL, branchPtr, opts.EnvFiles)
	if err != nil {
		return err
	}
	instance.Repos = append(instance.Repos, repoState)
	if err := state.Save(instance); err != nil {
		return err
	}

	fmt.Printf("done: %s is on %s at /root/code/%s\n", repo.Name, opts.InstanceName, repo.Name)
	fmt.Printf("  ssh:  hcode ssh %s --repo %s\n", opts.InstanceName, repo.Name)
	return nil
}
