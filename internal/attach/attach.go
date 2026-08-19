// Package attach holds shared logic for wiring one repo onto one
// already-reachable box — used by both `create` (box + first repo) and
// `add` (Nth repo onto an existing box), so the two never drift apart.
package attach

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dikee/hetzner-code/internal/config"
	"github.com/dikee/hetzner-code/internal/github"
	"github.com/dikee/hetzner-code/internal/keys"
	"github.com/dikee/hetzner-code/internal/naming"
	"github.com/dikee/hetzner-code/internal/provision"
	"github.com/dikee/hetzner-code/internal/run"
	"github.com/dikee/hetzner-code/internal/sshutil"
	"github.com/dikee/hetzner-code/internal/state"
)

// AttachRepo registers a fresh deploy key for repoURL, clones it onto
// the box, and copies up any envFiles.
func AttachRepo(ip, identityPath, instanceName, repoURL string, branch *string, envFiles []string) (state.Repo, error) {
	repo, err := github.ParseRepoURL(repoURL)
	if err != nil {
		return state.Repo{}, err
	}

	fmt.Printf("  registering deploy key for %s ...\n", repo.Slug())
	keyDir := state.KeyDir(instanceName, repo.Name)
	privateKey, publicKey, err := keys.Generate(keyDir, fmt.Sprintf("hcode:%s:%s", instanceName, repo.Name))
	if err != nil {
		return state.Repo{}, err
	}
	title := naming.GenerateKeyTitle(instanceName, repo.Name)
	deployKeyID, err := github.AddDeployKey(repo, publicKey, title, true)
	if err != nil {
		return state.Repo{}, err
	}

	remoteKeyPath := fmt.Sprintf("%s/%s", config.RemoteKeyDir, repo.Name)
	dest := fmt.Sprintf("%s/%s", config.RemoteCodeDir, repo.Name)

	attachErr := func() error {
		fmt.Printf("  cloning %s onto the box ...\n", repo.Slug())
		if err := sshutil.CopyTo(ip, privateKey, remoteKeyPath, identityPath); err != nil {
			return err
		}
		cloneCmd := provision.CloneRepoCommand(repoURL, repo.Name, branch, remoteKeyPath, dest)
		if _, err := sshutil.RunRemote(ip, cloneCmd, identityPath); err != nil {
			return err
		}

		if len(envFiles) > 0 {
			fmt.Printf("  copying %d env file(s) ...\n", len(envFiles))
			for _, local := range envFiles {
				if filepath.IsAbs(local) {
					return run.Errorf(
						"--env-file %s is absolute — pass it relative to the repo root "+
							"(e.g. backend/.env), the same relative path is used on the box", local,
					)
				}
				if _, err := os.Stat(local); err != nil {
					return run.Errorf("--env-file %s does not exist locally", local)
				}
				remotePath := fmt.Sprintf("%s/%s", dest, filepath.ToSlash(local))
				if err := sshutil.CopyTo(ip, local, remotePath, identityPath); err != nil {
					return err
				}
			}
		} else {
			fmt.Printf(
				"  no --env-file given — if this repo needs one, copy it up yourself or "+
					"re-run with --env-file (see `hcode status %s`)\n", instanceName,
			)
		}
		return nil
	}()

	if err := keys.ForgetPrivateKey(privateKey); err != nil {
		return state.Repo{}, err
	}
	if attachErr != nil {
		return state.Repo{}, attachErr
	}

	return state.Repo{
		URL:            repoURL,
		Owner:          repo.Owner,
		Name:           repo.Name,
		Branch:         branch,
		DeployKeyID:    deployKeyID,
		DeployKeyTitle: title,
		Worktrees:      []string{},
	}, nil
}
