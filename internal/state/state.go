// Package state is local state: one JSON file per instance under
// ~/.hetzner-code/instances/.
//
// This is the only source of truth hcode trusts for "what did I
// create." `status --reconcile` cross-checks it against hcloud/GitHub
// directly, since a crash between steps can leave this file out of sync
// with the real world — see internal/commands' status command.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/dikee/hetzner-code/internal/config"
	"github.com/dikee/hetzner-code/internal/run"
)

// Repo is one codebase cloned onto an instance.
type Repo struct {
	URL            string   `json:"url"` // the original git@github.com:owner/repo.git the user passed
	Owner          string   `json:"owner"`
	Name           string   `json:"name"` // short repo name, also the directory name on the box
	Branch         *string  `json:"branch"`
	DeployKeyID    string   `json:"deploy_key_id"` // GitHub's numeric id for this deploy key, for delete
	DeployKeyTitle string   `json:"deploy_key_title"`
	Worktrees      []string `json:"worktrees"` // labels: "cc2", "cc3", ...
}

// Instance is one Hetzner box hcode created and is tracking.
type Instance struct {
	Name         string  `json:"name"`
	ServerID     string  `json:"server_id"`
	IP           string  `json:"ip"`
	Type         string  `json:"type"`
	Location     string  `json:"location"`
	LoginKey     string  `json:"login_key"`      // the hcloud SSH key *name* injected into the box
	LoginKeyPath string  `json:"login_key_path"` // local private key path used to actually SSH in as that identity
	CreatedAt    string  `json:"created_at"`     // ISO 8601, set by the caller
	Repos        []Repo  `json:"repos"`
	OpsDir       *string `json:"ops_dir"`       // remote path, if --ops-dir was copied up at create
	OpsDirLocal  *string `json:"ops_dir_local"` // its origin on your laptop — destroy syncs back here
}

func instanceDir(name string) string {
	return filepath.Join(config.InstancesDir, name)
}

func metaPath(name string) string {
	return filepath.Join(instanceDir(name), "meta.json")
}

// Save writes instance to its meta.json, creating the instance
// directory if needed.
func Save(instance Instance) error {
	for i := range instance.Repos {
		if instance.Repos[i].Worktrees == nil {
			instance.Repos[i].Worktrees = []string{}
		}
	}
	if instance.Repos == nil {
		instance.Repos = []Repo{}
	}
	if err := os.MkdirAll(instanceDir(instance.Name), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath(instance.Name), data, 0o644)
}

// Load reads a tracked instance by name.
func Load(name string) (Instance, error) {
	path := metaPath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Instance{}, run.Errorf("no tracked instance named '%s' (see `hcode status`)", name)
		}
		return Instance{}, err
	}
	var instance Instance
	if err := json.Unmarshal(data, &instance); err != nil {
		return Instance{}, err
	}
	return instance, nil
}

// Exists reports whether an instance by this name is already tracked.
func Exists(name string) bool {
	_, err := os.Stat(metaPath(name))
	return err == nil
}

// ListAll returns every tracked instance, sorted by name.
func ListAll() ([]Instance, error) {
	entries, err := os.ReadDir(config.InstancesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var out []Instance
	for _, name := range names {
		if _, err := os.Stat(metaPath(name)); err == nil {
			instance, err := Load(name)
			if err != nil {
				return nil, err
			}
			out = append(out, instance)
		}
	}
	return out, nil
}

// Delete removes an instance's local state directory entirely.
func Delete(name string) error {
	d := instanceDir(name)
	if _, err := os.Stat(d); err == nil {
		return os.RemoveAll(d)
	}
	return nil
}

// KeyDir is where a repo's local keypair lives before/while it's being
// uploaded. The private half is deleted after a successful scp — see
// internal/keys.ForgetPrivateKey.
func KeyDir(instanceName, repoName string) string {
	return filepath.Join(instanceDir(instanceName), "keys", repoName)
}
