from __future__ import annotations

import json
from datetime import UTC, datetime

import click

from hetzner_code import hetzner, state


def _age(created_at: str) -> str:
    created = datetime.fromisoformat(created_at)
    delta = datetime.now(UTC) - created
    hours = delta.total_seconds() / 3600
    if hours < 1:
        return f"{int(hours * 60)}m"
    if hours < 24:
        return f"{hours:.1f}h"
    return f"{hours / 24:.1f}d"


def _repo_str(repo: state.Repo) -> str:
    base = f"{repo.owner}/{repo.name}@{repo.branch or 'default'}"
    if repo.worktrees:
        base += f" [{','.join(repo.worktrees)}]"
    return base


def _repos_str(instance: state.Instance) -> str:
    if not instance.repos:
        return "(none)"
    return ", ".join(_repo_str(r) for r in instance.repos)


def status(*, name: str | None, json_output: bool, reconcile: bool) -> None:
    instances = [state.load(name)] if name else state.list_all()

    if json_output:
        click.echo(json.dumps([_row(i) for i in instances], indent=2))
    elif not instances:
        click.echo("nothing tracked (see `hcode create`)")
    else:
        _print_table(instances)

    if reconcile:
        _reconcile(instances)


def _row(instance: state.Instance) -> dict:
    return {
        "name": instance.name,
        "ip": instance.ip,
        "type": instance.type,
        "location": instance.location,
        "age": _age(instance.created_at),
        "repos": [_repo_str(r) for r in instance.repos],
        "ops_dir": instance.ops_dir,
    }


def _print_table(instances: list[state.Instance]) -> None:
    rows = [
        (
            i.name,
            i.ip,
            i.type,
            i.location,
            _age(i.created_at),
            _repos_str(i),
            i.ops_dir or "-",
        )
        for i in instances
    ]
    headers = ("NAME", "IP", "TYPE", "LOCATION", "AGE", "REPOS", "OPS")
    widths = [
        max(len(headers[c]), max(len(r[c]) for r in rows)) for c in range(len(headers))
    ]
    click.echo("  ".join(h.ljust(w) for h, w in zip(headers, widths)))
    for row in rows:
        click.echo("  ".join(str(c).ljust(w) for c, w in zip(row, widths)))


def _reconcile(instances: list[state.Instance]) -> None:
    tracked_ids = {i.server_id for i in instances}
    live = hetzner.list_managed_servers()
    orphan_servers = [s for s in live if str(s["id"]) not in tracked_ids]
    if orphan_servers:
        click.echo("\norphaned servers (hcode-labeled, not in local state):")
        for s in orphan_servers:
            click.echo(
                f"  {s['name']}  id={s['id']}  ip={s['public_net']['ipv4']['ip']}"
            )
        click.echo("  clean up by hand: hcloud server delete <name>")

    for instance in instances:
        live_server = hetzner.describe_server(instance.server_id)
        if live_server is None:
            click.echo(
                f"\n'{instance.name}' is tracked locally but no longer exists on Hetzner "
                f"— its deploy key(s) may still be live on GitHub. Run:\n"
                f"  hcode destroy {instance.name} --keep-key\nto clear local state, "
                "then remove the deploy key(s) by hand if `gh` calls fail."
            )
