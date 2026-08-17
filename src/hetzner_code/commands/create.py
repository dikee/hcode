from __future__ import annotations

import tempfile
from datetime import UTC, datetime
from pathlib import Path

import click

from hetzner_code import attach, hetzner, ssh_util, state
from hetzner_code.config import REMOTE_CODE_DIR
from hetzner_code.naming import generate_instance_name
from hetzner_code.provision import BASE_PROVISION_SCRIPT, add_worktree_command
from hetzner_code.run import HcodeError


def create(
    *,
    repo_url: str,
    name: str | None,
    branch: str | None,
    server_type: str,
    location: str,
    login_key: str | None,
    login_key_path: str,
    env_files: tuple[str, ...],
    worktrees: int,
    ops_dir: str | None,
    no_attach: bool,
) -> None:
    from hetzner_code.github import parse_repo_url

    repo = parse_repo_url(repo_url)
    instance_name = name or generate_instance_name(repo.name)
    if state.exists(instance_name):
        raise HcodeError(
            f"instance '{instance_name}' already exists — pick another --name"
        )

    resolved_login_key = login_key or hetzner.pick_login_key()
    identity_path = Path(login_key_path).expanduser()
    if not identity_path.exists():
        raise HcodeError(f"--login-key-path {identity_path} does not exist")

    resolved_ops_dir = None
    if ops_dir:
        resolved_ops_dir = Path(ops_dir).expanduser()
        if not resolved_ops_dir.is_dir():
            raise HcodeError(f"--ops-dir {resolved_ops_dir} isn't a directory")

    click.echo(
        f"[1/6] checking {identity_path} matches '{resolved_login_key}' on Hetzner ..."
    )
    hetzner.verify_login_key(resolved_login_key, identity_path)

    click.echo(f"[2/6] creating {server_type} in {location} ({instance_name}) ...")
    with tempfile.NamedTemporaryFile("w", suffix=".sh", delete=False) as f:
        f.write(BASE_PROVISION_SCRIPT)
        script_path = Path(f.name)
    try:
        server_id, ip = hetzner.create_server(
            name=instance_name,
            server_type=server_type,
            location=location,
            login_key=resolved_login_key,
            user_data_file=script_path,
            labels={"hcode-instance": instance_name},
        )
    finally:
        script_path.unlink(missing_ok=True)

    click.echo(
        f"[3/6] waiting for {ip} to finish booting (this installs Docker/Node/uv, ~1-2 min) ..."
    )
    ssh_util.wait_for_ssh(ip, identity_path)

    click.echo("[4/6] attaching the repo ...")
    repo_state = attach.attach_repo(
        ip=ip,
        identity_path=identity_path,
        instance_name=instance_name,
        repo_url=repo_url,
        branch=branch,
        env_files=env_files,
    )

    if worktrees:
        click.echo(f"[5/6] adding {worktrees} worktree(s) for parallel lanes ...")
        main_dest = f"{REMOTE_CODE_DIR}/{repo.name}"
        for n in range(2, worktrees + 2):
            label = f"cc{n}"
            worktree_dest = f"{REMOTE_CODE_DIR}/{repo.name}-{label}"
            ssh_util.run_remote(
                ip,
                add_worktree_command(
                    main_dest=main_dest,
                    worktree_dest=worktree_dest,
                    branch=f"{label}/base",
                ),
                identity_path,
            )
            repo_state.worktrees.append(label)
            click.echo(f"  {worktree_dest}  (branch {label}/base)")
    else:
        click.echo("[5/6] no --worktrees requested, skipping")

    if resolved_ops_dir:
        remote_ops_dir = f"{REMOTE_CODE_DIR}/{repo.name}_ops"
        click.echo(f"[6/6] copying {resolved_ops_dir} -> {remote_ops_dir} ...")
        ssh_util.copy_dir_to(ip, resolved_ops_dir, remote_ops_dir, identity_path)
    else:
        remote_ops_dir = None
        click.echo("[6/6] no --ops-dir given, skipping")

    instance = state.Instance(
        name=instance_name,
        server_id=server_id,
        ip=ip,
        type=server_type,
        location=location,
        login_key=resolved_login_key,
        login_key_path=str(identity_path),
        created_at=datetime.now(UTC).isoformat(),
        repos=[repo_state],
        ops_dir=remote_ops_dir,
    )
    state.save(instance)

    click.echo(f"\n{instance_name} is up at {ip}")
    click.echo(f"  ssh:    hcode ssh {instance_name}")
    if worktrees:
        click.echo(f"  ssh:    hcode ssh {instance_name} --worktree cc2  (etc.)")
    click.echo(f"  add:    hcode add <repo-url> --instance {instance_name}")
    click.echo(f"  kill:   hcode destroy {instance_name}")

    if not no_attach:
        ssh_util.interactive(ip, identity_path, cwd=f"{REMOTE_CODE_DIR}/{repo.name}")
