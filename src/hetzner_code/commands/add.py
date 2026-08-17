from __future__ import annotations

from pathlib import Path

import click

from hetzner_code import attach, state
from hetzner_code.github import parse_repo_url
from hetzner_code.run import HcodeError


def add(
    *,
    repo_url: str,
    instance_name: str,
    branch: str | None,
    env_files: tuple[str, ...],
) -> None:
    instance = state.load(instance_name)
    repo = parse_repo_url(repo_url)

    if any(r.name == repo.name for r in instance.repos):
        raise HcodeError(f"'{repo.name}' is already on instance '{instance_name}'")

    if instance.repos:
        existing = ", ".join(r.name for r in instance.repos)
        click.echo(
            f"note: '{instance_name}' already runs [{existing}] — adding another codebase "
            "means they share this box's CPU. Your call; `hcode status` shows the type."
        )

    click.echo(f"attaching {repo.slug} to '{instance_name}' ({instance.ip}) ...")
    repo_state = attach.attach_repo(
        ip=instance.ip,
        identity_path=Path(instance.login_key_path),
        instance_name=instance_name,
        repo_url=repo_url,
        branch=branch,
        env_files=env_files,
    )
    instance.repos.append(repo_state)
    state.save(instance)

    click.echo(f"done: {repo.name} is on {instance_name} at /root/code/{repo.name}")
    click.echo(f"  ssh:  hcode ssh {instance_name} --repo {repo.name}")
