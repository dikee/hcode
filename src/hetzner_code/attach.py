"""Shared logic for wiring one repo onto one already-reachable box —
used by both `create` (box + first repo) and `add` (Nth repo onto an
existing box), so the two never drift apart.
"""

from __future__ import annotations

from pathlib import Path

import click

from hetzner_code import github, keys, ssh_util, state
from hetzner_code.config import REMOTE_CODE_DIR, REMOTE_KEY_DIR
from hetzner_code.naming import generate_key_title
from hetzner_code.provision import clone_repo_command
from hetzner_code.run import HcodeError


def attach_repo(
    *,
    ip: str,
    identity_path: Path,
    instance_name: str,
    repo_url: str,
    branch: str | None,
    env_files: tuple[str, ...],
) -> state.Repo:
    repo = github.parse_repo_url(repo_url)

    click.echo(f"  registering deploy key for {repo.slug} ...")
    key_dir = state.key_dir(instance_name, repo.name)
    private_key, public_key = keys.generate(
        key_dir, comment=f"hcode:{instance_name}:{repo.name}"
    )
    title = generate_key_title(instance_name, repo.name)
    deploy_key_id = github.add_deploy_key(repo, public_key_path=public_key, title=title)

    remote_key_path = f"{REMOTE_KEY_DIR}/{repo.name}"
    dest = f"{REMOTE_CODE_DIR}/{repo.name}"

    try:
        click.echo(f"  cloning {repo.slug} onto the box ...")
        ssh_util.copy_to(ip, private_key, remote_key_path, identity_path)
        ssh_util.run_remote(
            ip,
            clone_repo_command(
                repo_url=repo_url,
                repo_name=repo.name,
                branch=branch,
                key_path=remote_key_path,
                dest=dest,
            ),
            identity_path,
        )

        if env_files:
            click.echo(f"  copying {len(env_files)} env file(s) ...")
            for local in env_files:
                local_path = Path(local)
                if local_path.is_absolute():
                    raise HcodeError(
                        f"--env-file {local} is absolute — pass it relative to the repo "
                        "root (e.g. backend/.env), the same relative path is used on the box"
                    )
                if not local_path.exists():
                    raise HcodeError(f"--env-file {local} does not exist locally")
                remote_path = f"{dest}/{local_path.as_posix()}"
                ssh_util.copy_to(ip, local_path, remote_path, identity_path)
        else:
            click.echo(
                "  no --env-file given — if this repo needs one, copy it up "
                f"yourself or re-run with --env-file (see `hcode status {instance_name}`)"
            )
    finally:
        keys.forget_private_key(private_key)

    return state.Repo(
        url=repo_url,
        owner=repo.owner,
        name=repo.name,
        branch=branch,
        deploy_key_id=deploy_key_id,
        deploy_key_title=title,
    )
