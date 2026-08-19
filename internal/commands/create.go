package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dikee/hcode/internal/attach"
	"github.com/dikee/hcode/internal/config"
	"github.com/dikee/hcode/internal/forwards"
	"github.com/dikee/hcode/internal/github"
	"github.com/dikee/hcode/internal/hetzner"
	"github.com/dikee/hcode/internal/naming"
	"github.com/dikee/hcode/internal/provision"
	"github.com/dikee/hcode/internal/run"
	"github.com/dikee/hcode/internal/sshutil"
	"github.com/dikee/hcode/internal/state"
)

// CreateOptions are every flag `hcode create` accepts.
type CreateOptions struct {
	RepoURL      string
	Name         string
	Branch       string
	ServerType   string
	Location     string
	LoginKey     string
	LoginKeyPath string
	EnvFiles     []string
	Worktrees    int
	OpsDir       string
	PostClone    string
	PostWorktree string
	Forwards     []string
	NoAttach     bool
}

// Create runs `hcode create`.
func Create(opts CreateOptions) error {
	repo, err := github.ParseRepoURL(opts.RepoURL)
	if err != nil {
		return err
	}
	instanceName := opts.Name
	if instanceName == "" {
		instanceName = naming.GenerateInstanceName(repo.Name)
	}
	if state.Exists(instanceName) {
		return run.Errorf("instance '%s' already exists — pick another --name", instanceName)
	}

	if len(opts.Forwards) > 0 && opts.NoAttach {
		fmt.Println("note: --forward has no effect with --no-attach — nothing stays open to tunnel through")
	}

	resolvedLoginKey := opts.LoginKey
	if resolvedLoginKey == "" {
		resolvedLoginKey, err = hetzner.PickLoginKey()
		if err != nil {
			return err
		}
	}
	identityPath, err := expandUser(opts.LoginKeyPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(identityPath); err != nil {
		return run.Errorf("--login-key-path %s does not exist", identityPath)
	}

	var resolvedOpsDir string
	if opts.OpsDir != "" {
		resolvedOpsDir, err = expandUser(opts.OpsDir)
		if err != nil {
			return err
		}
		info, err := os.Stat(resolvedOpsDir)
		if err != nil || !info.IsDir() {
			return run.Errorf("--ops-dir %s isn't a directory", resolvedOpsDir)
		}
	}

	var branchPtr *string
	if opts.Branch != "" {
		branchPtr = &opts.Branch
	}

	fmt.Printf("[1/7] checking %s matches '%s' on Hetzner ...\n", identityPath, resolvedLoginKey)
	if err := hetzner.VerifyLoginKey(resolvedLoginKey, identityPath); err != nil {
		return err
	}

	fmt.Printf("[2/7] creating %s in %s (%s) ...\n", opts.ServerType, opts.Location, instanceName)
	scriptFile, err := os.CreateTemp("", "hcode-provision-*.sh")
	if err != nil {
		return err
	}
	scriptPath := scriptFile.Name()
	if _, err := scriptFile.WriteString(provision.BaseProvisionScript); err != nil {
		scriptFile.Close()
		os.Remove(scriptPath)
		return err
	}
	scriptFile.Close()
	serverID, ip, err := hetzner.CreateServer(
		instanceName, opts.ServerType, opts.Location, resolvedLoginKey, scriptPath,
		map[string]string{"hcode-instance": instanceName},
	)
	os.Remove(scriptPath)
	if err != nil {
		return err
	}

	fmt.Printf("[3/7] waiting for %s to finish booting (this installs Docker/Node/uv, ~1-2 min) ...\n", ip)
	if err := sshutil.WaitForSSH(ip, identityPath, 0); err != nil {
		return err
	}

	fmt.Println("[4/7] attaching the repo ...")
	repoState, err := attach.AttachRepo(ip, identityPath, instanceName, opts.RepoURL, branchPtr, opts.EnvFiles)
	if err != nil {
		return err
	}

	if opts.Worktrees > 0 {
		fmt.Printf("[5/7] adding %d worktree(s) for parallel lanes ...\n", opts.Worktrees)
		mainDest := fmt.Sprintf("%s/%s", config.RemoteCodeDir, repo.Name)
		for n := 2; n < opts.Worktrees+2; n++ {
			label := fmt.Sprintf("cc%d", n)
			worktreeDest := fmt.Sprintf("%s/%s-%s", config.RemoteCodeDir, repo.Name, label)
			cmd := provision.AddWorktreeCommand(mainDest, worktreeDest, fmt.Sprintf("%s/base", label))
			if _, err := sshutil.RunRemote(ip, cmd, identityPath); err != nil {
				return err
			}
			repoState.Worktrees = append(repoState.Worktrees, label)
			fmt.Printf("  %s  (branch %s/base)\n", worktreeDest, label)

			for _, local := range opts.EnvFiles {
				remotePath := fmt.Sprintf("%s/%s", worktreeDest, filepath.ToSlash(local))
				if err := sshutil.CopyTo(ip, local, remotePath, identityPath); err != nil {
					return err
				}
				fmt.Printf("    + %s (worktrees don't inherit untracked files on their own)\n", local)
			}

			if opts.PostWorktree != "" {
				fmt.Printf("    running %s for %s ...\n", opts.PostWorktree, label)
				cmd := fmt.Sprintf("cd %s && HCODE_WORKTREE_LABEL=%s bash %s", worktreeDest, label, opts.PostWorktree)
				if err := sshutil.RunRemoteStreaming(ip, cmd, identityPath); err != nil {
					return err
				}
			}
		}
	} else {
		fmt.Println("[5/7] no --worktrees requested, skipping")
	}

	var remoteOpsDir *string
	if resolvedOpsDir != "" {
		remote := fmt.Sprintf("%s/%s_ops", config.RemoteCodeDir, repo.Name)
		fmt.Printf("[6/7] copying %s -> %s ...\n", resolvedOpsDir, remote)
		if err := sshutil.CopyDirTo(ip, resolvedOpsDir, remote, identityPath); err != nil {
			return err
		}
		remoteOpsDir = &remote
	} else {
		fmt.Println("[6/7] no --ops-dir given, skipping")
	}

	if opts.PostClone != "" {
		mainDest := fmt.Sprintf("%s/%s", config.RemoteCodeDir, repo.Name)
		fmt.Printf("[7/7] running %s (repo-defined setup — output below) ...\n", opts.PostClone)
		cmd := fmt.Sprintf("cd %s && bash %s", mainDest, opts.PostClone)
		if err := sshutil.RunRemoteStreaming(ip, cmd, identityPath); err != nil {
			return err
		}
	} else {
		fmt.Println("[7/7] no --post-clone given, skipping")
	}

	var opsDirLocal *string
	if resolvedOpsDir != "" {
		opsDirLocal = &resolvedOpsDir
	}
	instance := state.Instance{
		Name:         instanceName,
		ServerID:     serverID,
		IP:           ip,
		Type:         opts.ServerType,
		Location:     opts.Location,
		LoginKey:     resolvedLoginKey,
		LoginKeyPath: identityPath,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Repos:        []state.Repo{repoState},
		OpsDir:       remoteOpsDir,
		OpsDirLocal:  opsDirLocal,
	}
	if err := state.Save(instance); err != nil {
		return err
	}

	fmt.Printf("\n%s is up at %s\n", instanceName, ip)
	fmt.Printf("  ssh:    hcode ssh %s\n", instanceName)
	if opts.Worktrees > 0 {
		fmt.Printf("  ssh:    hcode ssh %s --worktree cc2  (etc.)\n", instanceName)
	}
	fmt.Printf("  add:    hcode add <repo-url> --instance %s\n", instanceName)
	fmt.Printf("  kill:   hcode destroy %s\n", instanceName)

	if !opts.NoAttach {
		normalized := make([]string, 0, len(opts.Forwards))
		for _, f := range opts.Forwards {
			nf, err := forwards.NormalizeForward(f)
			if err != nil {
				return err
			}
			normalized = append(normalized, nf)
		}
		sshutil.Interactive(ip, identityPath, fmt.Sprintf("%s/%s", config.RemoteCodeDir, repo.Name), normalized)
	}
	return nil
}
