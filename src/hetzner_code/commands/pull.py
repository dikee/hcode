from __future__ import annotations

from pathlib import Path

import click

from hetzner_code import ssh_util, state
from hetzner_code.config import REMOTE_CODE_DIR
from hetzner_code.run import HcodeError


def pull(
    *,
    instance_name: str,
    remote_path: str,
    local_path: str | None,
    repo_name: str | None,
) -> None:
    instance = state.load(instance_name)

    resolved_remote = remote_path
    if repo_name:
        if not any(r.name == repo_name for r in instance.repos):
            available = ", ".join(r.name for r in instance.repos) or "(none)"
            raise HcodeError(
                f"'{repo_name}' isn't on '{instance_name}'. On it: {available}"
            )
        if not remote_path.startswith("/"):
            resolved_remote = f"{REMOTE_CODE_DIR}/{repo_name}/{remote_path}"

    resolved_local = (
        Path(local_path) if local_path else Path.cwd() / Path(remote_path).name
    )

    ssh_util.copy_from(
        instance.ip, resolved_remote, resolved_local, Path(instance.login_key_path)
    )
    click.echo(f"pulled {resolved_remote} -> {resolved_local}")
