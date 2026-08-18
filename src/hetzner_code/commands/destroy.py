from __future__ import annotations

import re
from pathlib import Path

import click

from hetzner_code import github, hetzner, ssh_util, state
from hetzner_code.config import REMOTE_CODE_DIR
from hetzner_code.run import HcodeError


def _parse_git_status(output: str) -> tuple[int, int]:
    """`git status --porcelain=v1 -b` -> (uncommitted_count, ahead_count).
    The first line is always the branch line (e.g. "## main...origin/main
    [ahead 2]"); every line after it is one changed/untracked file."""
    lines = [line for line in output.splitlines() if line]
    if not lines:
        return 0, 0
    match = re.search(r"\[ahead (\d+)", lines[0])
    ahead = int(match.group(1)) if match else 0
    return len(lines) - 1, ahead


def _git_risk(ip: str, label: str, dest: str, identity_path: Path) -> str | None:
    try:
        result = ssh_util.run_remote(
            ip, f"git -C {dest} status --porcelain=v1 -b", identity_path
        )
    except HcodeError:
        return None  # dest doesn't exist, or isn't a git repo — nothing to warn about
    uncommitted, ahead = _parse_git_status(result.stdout)
    parts = []
    if uncommitted:
        parts.append(f"{uncommitted} uncommitted file(s)")
    if ahead:
        parts.append(f"{ahead} unpushed commit(s)")
    return f"{label}: {', '.join(parts)}" if parts else None


def _git_risks(instance: state.Instance) -> list[str]:
    """Everything about to be lost forever once the box is deleted — not
    a full backup, just a heads-up before an irreversible action."""
    if hetzner.describe_server(instance.server_id) is None:
        return []
    identity_path = Path(instance.login_key_path)
    risks = []
    for repo in instance.repos:
        risk = _git_risk(
            instance.ip, repo.name, f"{REMOTE_CODE_DIR}/{repo.name}", identity_path
        )
        if risk:
            risks.append(risk)
        for label in repo.worktrees:
            wt_label = f"{repo.name}-{label}"
            risk = _git_risk(
                instance.ip, wt_label, f"{REMOTE_CODE_DIR}/{wt_label}", identity_path
            )
            if risk:
                risks.append(risk)
    return risks


def _pull_ops_dir(instance: state.Instance) -> None:
    if not (instance.ops_dir and instance.ops_dir_local):
        return
    if hetzner.describe_server(instance.server_id) is None:
        click.echo(f"  (server already gone — couldn't pull {instance.ops_dir} back)")
        return
    click.echo(f"  pulling {instance.ops_dir} -> {instance.ops_dir_local} ...")
    try:
        ssh_util.sync_dir_from(
            instance.ip,
            instance.ops_dir,
            Path(instance.ops_dir_local),
            Path(instance.login_key_path),
        )
    except HcodeError as exc:
        click.echo(f"  warning: couldn't pull ops dir back: {exc}")


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
        risks = _git_risks(instance)
        prompt = (
            f"Destroy '{instance.name}' ({instance.ip}, {repos})? "
            "This deletes the box and every deploy key on it."
        )
        if risks:
            prompt += "\n  UNSAVED WORK WILL BE LOST:\n" + "\n".join(
                f"    - {r}" for r in risks
            )

        if not yes:
            click.confirm(prompt, abort=True)
        elif risks:
            click.echo(f"warning: destroying '{instance.name}' with unsaved work:")
            for r in risks:
                click.echo(f"  - {r}")

        _pull_ops_dir(instance)

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
