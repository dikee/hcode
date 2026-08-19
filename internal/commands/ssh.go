package commands

import (
	"fmt"
	"strings"

	"github.com/dikee/hcode/internal/config"
	"github.com/dikee/hcode/internal/forwards"
	"github.com/dikee/hcode/internal/run"
	"github.com/dikee/hcode/internal/sshutil"
	"github.com/dikee/hcode/internal/state"
)

func resolveCwd(instance state.Instance, repoName, worktree string) (string, error) {
	if worktree != "" {
		var candidates []state.Repo
		for _, r := range instance.Repos {
			for _, w := range r.Worktrees {
				if w == worktree {
					candidates = append(candidates, r)
					break
				}
			}
		}
		if repoName != "" {
			filtered := candidates[:0]
			for _, r := range candidates {
				if r.Name == repoName {
					filtered = append(filtered, r)
				}
			}
			candidates = filtered
		}
		if len(candidates) == 0 {
			suffix := ""
			if repoName != "" {
				suffix = fmt.Sprintf(" on repo '%s'", repoName)
			}
			return "", run.Errorf(
				"no worktree '%s' found%s (see `hcode status %s`)", worktree, suffix, instance.Name,
			)
		}
		if len(candidates) > 1 {
			names := make([]string, len(candidates))
			for i, r := range candidates {
				names[i] = r.Name
			}
			return "", run.Errorf(
				"worktree '%s' exists on more than one repo (%s) — disambiguate with --repo",
				worktree, strings.Join(names, ", "),
			)
		}
		return fmt.Sprintf("%s/%s-%s", config.RemoteCodeDir, candidates[0].Name, worktree), nil
	}

	if repoName != "" {
		found := false
		for _, r := range instance.Repos {
			if r.Name == repoName {
				found = true
				break
			}
		}
		if !found {
			return "", run.Errorf("'%s' isn't on '%s'. On it: %s", repoName, instance.Name, reposList(instance))
		}
		return fmt.Sprintf("%s/%s", config.RemoteCodeDir, repoName), nil
	}

	if len(instance.Repos) == 1 {
		return fmt.Sprintf("%s/%s", config.RemoteCodeDir, instance.Repos[0].Name), nil
	}
	return "", nil
}

// SSHOptions are every flag `hcode ssh` accepts.
type SSHOptions struct {
	InstanceName string
	RepoName     string
	Worktree     string
	Forwards     []string
}

// SSH runs `hcode ssh`, returning the exit code of the interactive
// session.
func SSH(opts SSHOptions) (int, error) {
	instance, err := state.Load(opts.InstanceName)
	if err != nil {
		return 1, err
	}
	cwd, err := resolveCwd(instance, opts.RepoName, opts.Worktree)
	if err != nil {
		return 1, err
	}
	normalized := make([]string, 0, len(opts.Forwards))
	for _, f := range opts.Forwards {
		nf, err := forwards.NormalizeForward(f)
		if err != nil {
			return 1, err
		}
		normalized = append(normalized, nf)
	}
	return sshutil.Interactive(instance.IP, instance.LoginKeyPath, cwd, normalized), nil
}
