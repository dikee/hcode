from __future__ import annotations

import click

from hetzner_code import github, hetzner, state
from hetzner_code.run import HcodeError


def destroy(*, name: str | None, all_: bool, keep_key: bool, yes: bool) -> None:
    if all_:
        targets = state.list_all()
        if not targets:
            click.echo("nothing tracked, nothing to do")
            return
    else:
        if not name:
            raise HcodeError("pass an instance name or --all")
        targets = [state.load(name)]

    for instance in targets:
        repos = ", ".join(r.name for r in instance.repos) or "(no repos)"
        if not yes:
            click.confirm(
                f"Destroy '{instance.name}' ({instance.ip}, {repos})? "
                "This deletes the box and every deploy key on it.",
                abort=True,
            )

        if not keep_key:
            for repo in instance.repos:
                github.delete_deploy_key(
                    github.RepoRef(owner=repo.owner, name=repo.name), repo.deploy_key_id
                )

        if hetzner.describe_server(instance.server_id) is not None:
            hetzner.delete_server(instance.server_id)
        else:
            click.echo(f"  (server for '{instance.name}' was already gone on Hetzner)")

        state.delete(instance.name)
        click.echo(f"destroyed {instance.name}")
