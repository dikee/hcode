package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dikee/hcode/internal/config"
	"github.com/dikee/hcode/internal/github"
	"github.com/dikee/hcode/internal/hetzner"
	"github.com/dikee/hcode/internal/run"
	"github.com/dikee/hcode/internal/sshutil"
	"github.com/dikee/hcode/internal/state"
)

var aheadRe = regexp.MustCompile(`\[ahead (\d+)`)

// parseGitStatus turns `git status --porcelain=v1 -b` output into
// (uncommittedCount, aheadCount). The first line is always the branch
// line (e.g. "## main...origin/main [ahead 2]"); every line after it is
// one changed/untracked file.
func parseGitStatus(output string) (int, int) {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return 0, 0
	}
	ahead := 0
	if m := aheadRe.FindStringSubmatch(lines[0]); m != nil {
		ahead, _ = strconv.Atoi(m[1])
	}
	return len(lines) - 1, ahead
}

func gitRisk(ip, label, dest, identityPath string) string {
	result, err := sshutil.RunRemote(ip, fmt.Sprintf("git -C %s status --porcelain=v1 -b", dest), identityPath)
	if err != nil {
		return "" // dest doesn't exist, or isn't a git repo — nothing to warn about
	}
	uncommitted, ahead := parseGitStatus(result.Stdout)
	var parts []string
	if uncommitted > 0 {
		parts = append(parts, fmt.Sprintf("%d uncommitted file(s)", uncommitted))
	}
	if ahead > 0 {
		parts = append(parts, fmt.Sprintf("%d unpushed commit(s)", ahead))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("%s: %s", label, strings.Join(parts, ", "))
}

// gitRisks is everything about to be lost forever once the box is
// deleted — not a full backup, just a heads-up before an irreversible
// action.
func gitRisks(instance state.Instance) ([]string, error) {
	info, err := hetzner.DescribeServer(instance.ServerID)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	var risks []string
	for _, repo := range instance.Repos {
		if risk := gitRisk(instance.IP, repo.Name, fmt.Sprintf("%s/%s", config.RemoteCodeDir, repo.Name), instance.LoginKeyPath); risk != "" {
			risks = append(risks, risk)
		}
		for _, label := range repo.Worktrees {
			wtLabel := fmt.Sprintf("%s-%s", repo.Name, label)
			if risk := gitRisk(instance.IP, wtLabel, fmt.Sprintf("%s/%s", config.RemoteCodeDir, wtLabel), instance.LoginKeyPath); risk != "" {
				risks = append(risks, risk)
			}
		}
	}
	return risks, nil
}

func pullOpsDir(instance state.Instance) error {
	if instance.OpsDir == nil || instance.OpsDirLocal == nil {
		return nil
	}
	info, err := hetzner.DescribeServer(instance.ServerID)
	if err != nil {
		return err
	}
	if info == nil {
		fmt.Printf("  (server already gone — couldn't pull %s back)\n", *instance.OpsDir)
		return nil
	}
	fmt.Printf("  pulling %s -> %s ...\n", *instance.OpsDir, *instance.OpsDirLocal)
	if err := sshutil.SyncDirFrom(instance.IP, *instance.OpsDir, *instance.OpsDirLocal, instance.LoginKeyPath); err != nil {
		fmt.Printf("  warning: couldn't pull ops dir back: %s\n", err)
	}
	return nil
}

// DestroyOptions are every flag `hcode destroy` accepts.
type DestroyOptions struct {
	Name    string
	All     bool
	KeepKey bool
	Yes     bool
}

// Destroy runs `hcode destroy`.
func Destroy(opts DestroyOptions) error {
	var targets []state.Instance
	if opts.All {
		all, err := state.ListAll()
		if err != nil {
			return err
		}
		if len(all) == 0 {
			fmt.Println("nothing tracked, nothing to do")
			return nil
		}
		targets = all
	} else {
		if opts.Name == "" {
			return run.Errorf("pass an instance name or --all")
		}
		instance, err := state.Load(opts.Name)
		if err != nil {
			return err
		}
		targets = []state.Instance{instance}
	}

	for _, instance := range targets {
		repoNames := make([]string, len(instance.Repos))
		for i, r := range instance.Repos {
			repoNames[i] = r.Name
		}
		repos := strings.Join(repoNames, ", ")
		if repos == "" {
			repos = "(no repos)"
		}
		risks, err := gitRisks(instance)
		if err != nil {
			return err
		}
		prompt := fmt.Sprintf(
			"Destroy '%s' (%s, %s)? This deletes the box and every deploy key on it.",
			instance.Name, instance.IP, repos,
		)
		if len(risks) > 0 {
			var b strings.Builder
			b.WriteString(prompt)
			b.WriteString("\n  UNSAVED WORK WILL BE LOST:\n")
			for _, r := range risks {
				fmt.Fprintf(&b, "    - %s\n", r)
			}
			prompt = strings.TrimRight(b.String(), "\n")
		}

		if !opts.Yes {
			if err := confirm(prompt); err != nil {
				return err
			}
		} else if len(risks) > 0 {
			fmt.Printf("warning: destroying '%s' with unsaved work:\n", instance.Name)
			for _, r := range risks {
				fmt.Printf("  - %s\n", r)
			}
		}

		if err := pullOpsDir(instance); err != nil {
			return err
		}

		if !opts.KeepKey {
			for _, repo := range instance.Repos {
				if err := github.DeleteDeployKey(github.RepoRef{Owner: repo.Owner, Name: repo.Name}, repo.DeployKeyID); err != nil {
					return err
				}
			}
		}

		info, err := hetzner.DescribeServer(instance.ServerID)
		if err != nil {
			return err
		}
		if info != nil {
			if err := hetzner.DeleteServer(instance.ServerID); err != nil {
				return err
			}
		} else {
			fmt.Printf("  (server for '%s' was already gone on Hetzner)\n", instance.Name)
		}

		if err := state.Delete(instance.Name); err != nil {
			return err
		}
		fmt.Printf("destroyed %s\n", instance.Name)
	}
	return nil
}
