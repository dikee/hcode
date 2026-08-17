from __future__ import annotations

from pathlib import Path

import click

from hetzner_code import github, ssh_util, state
from hetzner_code.config import REMOTE_CODE_DIR, REMOTE_KEY_DIR
from hetzner_code.run import HcodeError


def remove(*, repo_name: str, instance_name: str, yes: bool) -> None:
    instance = state.load(instance_name)
    match = next((r for r in instance.repos if r.name == repo_name), None)
    if match is None:
        available = ", ".join(r.name for r in instance.repos) or "(none)"
        raise HcodeError(
            f"'{repo_name}' isn't on '{instance_name}'. On it: {available}"
        )

    if not yes:
        click.confirm(
            f"Delete the deploy key for {match.owner}/{match.name} and remove it from "
            f"'{instance_name}'? The box and every other codebase on it stay up.",
            abort=True,
        )

    github.delete_deploy_key(
        github.RepoRef(owner=match.owner, name=match.name), match.deploy_key_id
    )
    ssh_util.run_remote(
        instance.ip,
        f"rm -rf {REMOTE_CODE_DIR}/{repo_name} {REMOTE_KEY_DIR}/{repo_name}",
        Path(instance.login_key_path),
    )

    instance.repos = [r for r in instance.repos if r.name != repo_name]
    state.save(instance)
    click.echo(f"removed {repo_name} from {instance_name}")
