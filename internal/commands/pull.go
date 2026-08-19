package commands

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dikee/hetzner-code/internal/config"
	"github.com/dikee/hetzner-code/internal/run"
	"github.com/dikee/hetzner-code/internal/sshutil"
	"github.com/dikee/hetzner-code/internal/state"
)

// PullOptions are every flag `hcode pull` accepts.
type PullOptions struct {
	InstanceName string
	RemotePath   string
	LocalPath    string
	RepoName     string
}

// Pull runs `hcode pull`.
func Pull(opts PullOptions) error {
	instance, err := state.Load(opts.InstanceName)
	if err != nil {
		return err
	}

	resolvedRemote := opts.RemotePath
	if opts.RepoName != "" {
		found := false
		for _, r := range instance.Repos {
			if r.Name == opts.RepoName {
				found = true
				break
			}
		}
		if !found {
			return run.Errorf("'%s' isn't on '%s'. On it: %s", opts.RepoName, opts.InstanceName, reposList(instance))
		}
		if !strings.HasPrefix(opts.RemotePath, "/") {
			resolvedRemote = fmt.Sprintf("%s/%s/%s", config.RemoteCodeDir, opts.RepoName, opts.RemotePath)
		}
	}

	resolvedLocal := opts.LocalPath
	if resolvedLocal == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		resolvedLocal = filepath.Join(cwd, path.Base(opts.RemotePath))
	}

	if err := sshutil.CopyFrom(instance.IP, resolvedRemote, resolvedLocal, instance.LoginKeyPath); err != nil {
		return err
	}
	fmt.Printf("pulled %s -> %s\n", resolvedRemote, resolvedLocal)
	return nil
}
