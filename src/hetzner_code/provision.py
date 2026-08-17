"""Boot-time provisioning. Deliberately no secrets in here: this whole
script becomes Hetzner's user-data, which is stored server-side as
plaintext and readable through the API/console — see README.md's
"where secrets live" section. Anything sensitive (repo deploy keys,
.env files) goes over SSH after the box is already up.
"""

from __future__ import annotations

BASE_PROVISION_SCRIPT = """#!/bin/bash
set -euxo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y --no-install-recommends \\
    git curl ca-certificates build-essential unzip jq \\
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
"""


def clone_repo_command(
    *,
    repo_url: str,
    repo_name: str,
    branch: str | None,
    key_path: str,
    dest: str,
) -> str:
    """Command run over SSH, after the private key is already on the
    box, to clone the repo with that key and wire git to keep using it
    for future push/pull — see README.md's per-repo key rationale."""
    branch_flag = f"--branch {branch} " if branch else ""
    return (
        f"chmod 600 {key_path} && "
        f"GIT_SSH_COMMAND='ssh -i {key_path} -o IdentitiesOnly=yes -o StrictHostKeyChecking=no' "
        f"git clone {branch_flag}{repo_url} {dest} && "
        f"git -C {dest} config core.sshCommand "
        f"'ssh -i {key_path} -o IdentitiesOnly=yes -o StrictHostKeyChecking=no'"
    )
