from __future__ import annotations

import sys

import click

from hetzner_code.config import DEFAULT_LOCATION, DEFAULT_TYPE
from hetzner_code.run import HcodeError


@click.group()
def cli() -> None:
    """Disposable Hetzner boxes for Claude Code, wired to a git repo via
    a per-repo GitHub deploy key. Runs `hcloud` and `gh` under the hood —
    both must already be authenticated."""


@cli.command()
@click.argument("repo_url")
@click.option("--name", default=None, help="Instance name. Default: <repo>-<random>.")
@click.option(
    "--branch",
    default=None,
    help="Branch to check out. Default: repo's default branch.",
)
@click.option(
    "--type",
    "server_type",
    default=DEFAULT_TYPE,
    show_default=True,
    help="hcloud server type.",
)
@click.option(
    "--location", default=DEFAULT_LOCATION, show_default=True, help="hcloud location."
)
@click.option(
    "--login-key",
    default=None,
    help="hcloud SSH key name for your own access. Default: first on the account.",
)
@click.option(
    "--login-key-path",
    default="~/.ssh/id_ed25519",
    show_default=True,
    help="Local private key matching --login-key, used to actually SSH in.",
)
@click.option(
    "--env-file",
    "env_files",
    multiple=True,
    help="Local .env to copy up, repeatable. Path is relative to the repo root on both ends.",
)
@click.option(
    "--worktrees",
    default=0,
    show_default=True,
    help="Add N extra git worktrees (cc2, cc3, ...) for parallel lanes on their own branches.",
)
@click.option(
    "--ops-dir",
    default=None,
    help="Local directory to copy up as a sibling of the repo/worktrees (e.g. an "
    "orchestration mailbox folder) — never inside any single worktree, so every "
    "lane can see it.",
)
@click.option(
    "--post-clone",
    default=None,
    help="A script inside the repo (path relative to its root, e.g. bin/bootstrap.sh) "
    "to run once, after cloning — for repo-specific setup (installing a database "
    "server, running migrations) that's the repo's business, not hcode's. Runs "
    "with output streamed live, since these can take a while.",
)
@click.option(
    "--forward",
    "-L",
    "forwards",
    multiple=True,
    help="Forward a local port to the box on the post-create SSH session, repeatable. "
    "Accepts ssh's own PORT:HOST:PORT syntax, or shorthand: 8000 (same port both "
    "ends), 8000:9000 (different remote port). No effect with --no-attach.",
)
@click.option(
    "--no-attach",
    is_flag=True,
    help="Don't SSH in after creation; just print connection info.",
)
def create(
    repo_url,
    name,
    branch,
    server_type,
    location,
    login_key,
    login_key_path,
    env_files,
    worktrees,
    ops_dir,
    post_clone,
    forwards,
    no_attach,
):
    """Create a box and clone REPO_URL onto it."""
    from hetzner_code.commands.create import create as _create

    _create(
        repo_url=repo_url,
        name=name,
        branch=branch,
        server_type=server_type,
        location=location,
        login_key=login_key,
        login_key_path=login_key_path,
        env_files=env_files,
        worktrees=worktrees,
        ops_dir=ops_dir,
        post_clone=post_clone,
        forwards=forwards,
        no_attach=no_attach,
    )


@cli.command()
@click.argument("repo_url")
@click.option(
    "--instance",
    "instance_name",
    required=True,
    help="Existing instance to add this repo to.",
)
@click.option(
    "--branch",
    default=None,
    help="Branch to check out. Default: repo's default branch.",
)
@click.option(
    "--env-file", "env_files", multiple=True, help="Local .env to copy up, repeatable."
)
def add(repo_url, instance_name, branch, env_files):
    """Clone a second (third, ...) repo onto an already-running instance."""
    from hetzner_code.commands.add import add as _add

    _add(
        repo_url=repo_url,
        instance_name=instance_name,
        branch=branch,
        env_files=env_files,
    )


@cli.command()
@click.argument("repo_name")
@click.option(
    "--instance", "instance_name", required=True, help="Instance the repo is on."
)
@click.option("--yes", is_flag=True, help="Skip the confirmation prompt.")
def remove(repo_name, instance_name, yes):
    """Remove one codebase from an instance — deletes its deploy key and
    its clone directory, leaves the box and every other codebase up."""
    from hetzner_code.commands.remove import remove as _remove

    _remove(repo_name=repo_name, instance_name=instance_name, yes=yes)


@cli.command()
@click.argument("name", required=False)
@click.option("--all", "all_", is_flag=True, help="Destroy every tracked instance.")
@click.option(
    "--keep-key",
    is_flag=True,
    help="Delete the box but leave every deploy key on GitHub.",
)
@click.option("--yes", is_flag=True, help="Skip the confirmation prompt.")
def destroy(name, all_, keep_key, yes):
    """Destroy an instance: the box, and every deploy key it holds."""
    from hetzner_code.commands.destroy import destroy as _destroy

    _destroy(name=name, all_=all_, keep_key=keep_key, yes=yes)


@cli.command()
@click.argument("name", required=False)
@click.option("--json", "json_output", is_flag=True, help="Machine-readable output.")
@click.option(
    "--reconcile", is_flag=True, help="Cross-check against hcloud/GitHub for orphans."
)
def status(name, json_output, reconcile):
    """List tracked instances, or show detail for one."""
    from hetzner_code.commands.status import status as _status

    _status(name=name, json_output=json_output, reconcile=reconcile)


@cli.command()
@click.argument("name")
@click.option(
    "--repo",
    "repo_name",
    default=None,
    help="cd into this repo's directory on connect.",
)
@click.option(
    "--worktree",
    default=None,
    help="cd into this worktree (e.g. cc2) instead of the repo's main clone.",
)
@click.option(
    "--forward",
    "-L",
    "forwards",
    multiple=True,
    help="Forward a local port to the box, repeatable. Accepts ssh's own "
    "PORT:HOST:PORT syntax, or shorthand: 8000 (same port both ends), "
    "8000:9000 (different remote port).",
)
def ssh(name, repo_name, worktree, forwards):
    """SSH into a tracked instance."""
    from hetzner_code.commands.ssh import ssh as _ssh

    sys.exit(
        _ssh(
            instance_name=name,
            repo_name=repo_name,
            worktree=worktree,
            forwards=forwards,
        )
    )


@cli.command()
@click.argument("name")
@click.argument("remote_path")
@click.argument("local_path", required=False)
@click.option(
    "--repo",
    "repo_name",
    default=None,
    help="Resolve REMOTE_PATH relative to this repo's clone dir instead of an absolute path.",
)
def pull(name, remote_path, local_path, repo_name):
    """Copy a file or directory back from an instance. LOCAL_PATH
    defaults to REMOTE_PATH's basename in the current directory."""
    from hetzner_code.commands.pull import pull as _pull

    _pull(
        instance_name=name,
        remote_path=remote_path,
        local_path=local_path,
        repo_name=repo_name,
    )


@cli.command()
def workflow_help():
    """Print the multi-lane orchestrator + worker-lane workflow."""
    from hetzner_code.workflow_help import WORKFLOW_HELP

    click.echo(WORKFLOW_HELP)


def main() -> None:
    try:
        cli()
    except HcodeError as exc:
        click.echo(f"error: {exc}", err=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
