from __future__ import annotations

from pathlib import Path

from hetzner_code import ssh_util, state
from hetzner_code.config import REMOTE_CODE_DIR
from hetzner_code.run import HcodeError


def ssh(*, instance_name: str, repo_name: str | None) -> int:
    instance = state.load(instance_name)
    cwd = None
    if repo_name:
        if not any(r.name == repo_name for r in instance.repos):
            available = ", ".join(r.name for r in instance.repos) or "(none)"
            raise HcodeError(
                f"'{repo_name}' isn't on '{instance_name}'. On it: {available}"
            )
        cwd = f"{REMOTE_CODE_DIR}/{repo_name}"
    elif len(instance.repos) == 1:
        cwd = f"{REMOTE_CODE_DIR}/{instance.repos[0].name}"
    return ssh_util.interactive(instance.ip, Path(instance.login_key_path), cwd=cwd)
