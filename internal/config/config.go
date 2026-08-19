// Package config holds paths and defaults. Nothing here talks to a network.
package config

import (
	"os"
	"path/filepath"
)

var (
	StateDir     = filepath.Join(homeDir(), ".hcode")
	InstancesDir = filepath.Join(StateDir, "instances")
)

const (
	DefaultType     = "ccx33"
	DefaultLocation = "nbg1"
	DefaultImage    = "ubuntu-24.04"

	RemoteCodeDir = "/root/code"
	RemoteKeyDir  = "/root/.ssh/hcode"
)

// OwnerLabel is carried by every hcode-created server so `status
// --reconcile` can tell "ours, untracked locally" apart from "not ours
// at all" on an account that runs other things too.
var OwnerLabel = map[string]string{"managed-by": "hcode"}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return h
}
