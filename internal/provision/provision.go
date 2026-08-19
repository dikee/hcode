// Package provision holds boot-time provisioning. Deliberately no
// secrets in here: BaseProvisionScript becomes Hetzner's user-data,
// which is stored server-side as plaintext and readable through the
// API/console — see README.md's "where secrets live" section. Anything
// sensitive (repo deploy keys, .env files) goes over SSH after the box
// is already up.
package provision

import "fmt"

// BaseProvisionScript installs everything a fresh box needs before any
// repo-specific setup runs.
const BaseProvisionScript = `#!/bin/bash
set -euxo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y --no-install-recommends \
    git curl ca-certificates build-essential unzip jq \
    postgresql-client

# uv — Python package management, as this project's own conventions require.
curl -LsSf https://astral.sh/uv/install.sh | env UV_INSTALL_DIR=/usr/local/bin sh

# Node LTS, for Claude Code (npm-distributed).
curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -
apt-get install -y --no-install-recommends nodejs
npm install -g @anthropic-ai/claude-code

# Docker, for repos that ship a compose stack.
curl -fsSL https://get.docker.com | sh
systemctl enable --now docker

mkdir -p /root/code /root/.ssh/hcode
chmod 700 /root/.ssh/hcode
`

// CloneRepoCommand is run over SSH, after the private key is already on
// the box, to clone the repo with that key and wire git to keep using
// it for future push/pull — see README.md's per-repo key rationale.
func CloneRepoCommand(repoURL, repoName string, branch *string, keyPath, dest string) string {
	branchFlag := ""
	if branch != nil && *branch != "" {
		branchFlag = fmt.Sprintf("--branch %s ", *branch)
	}
	sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=no", keyPath)
	return fmt.Sprintf(
		"chmod 600 %s && GIT_SSH_COMMAND='%s' git clone %s%s %s && git -C %s config core.sshCommand '%s'",
		keyPath, sshCmd, branchFlag, repoURL, dest, dest, sshCmd,
	)
}

// AddWorktreeCommand is a new worktree of an already-cloned repo, on a
// fresh branch off whatever the main clone's HEAD currently is.
// Worktrees share the main clone's `.git` — and therefore its
// `core.sshCommand` — so no separate key setup is needed here.
func AddWorktreeCommand(mainDest, worktreeDest, branch string) string {
	return fmt.Sprintf("git -C %s worktree add %s -b %s HEAD", mainDest, worktreeDest, branch)
}
