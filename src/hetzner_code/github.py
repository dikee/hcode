"""Everything that talks to GitHub runs here, and only ever on the local
machine — never shelled out to from the box itself. `gh` must already be
authenticated (`gh auth status`); hcode doesn't manage that credential.
"""

from __future__ import annotations

import re
from dataclasses import dataclass

from hetzner_code.run import HcodeError, run, run_json

_SSH_URL_RE = re.compile(
    r"^(?:git@|ssh://git@)github\.com[:/](?P<owner>[^/]+)/(?P<repo>[^/.]+?)(?:\.git)?/?$"
)


@dataclass
class RepoRef:
    owner: str
    name: str

    @property
    def slug(self) -> str:
        return f"{self.owner}/{self.name}"


def parse_repo_url(url: str) -> RepoRef:
    match = _SSH_URL_RE.match(url.strip())
    if not match:
        raise HcodeError(
            f"'{url}' doesn't look like a GitHub SSH URL "
            "(expected git@github.com:owner/repo.git)"
        )
    return RepoRef(owner=match.group("owner"), name=match.group("repo"))


def add_deploy_key(
    repo: RepoRef, *, public_key_path, title: str, write: bool = True
) -> str:
    """Register `public_key_path` as a deploy key on `repo`, return its id.

    `gh repo deploy-key add` doesn't print the id, so this looks it back
    up by the (unique, caller-supplied) title immediately after adding.
    """
    cmd = [
        "gh",
        "repo",
        "deploy-key",
        "add",
        str(public_key_path),
        "-R",
        repo.slug,
        "-t",
        title,
    ]
    if write:
        cmd.append("-w")
    run(cmd)

    keys = run_json(
        ["gh", "repo", "deploy-key", "list", "-R", repo.slug, "--json", "id,title"]
    )
    matches = [k for k in keys if k["title"] == title]
    if not matches:
        raise HcodeError(
            f"added deploy key '{title}' to {repo.slug} but couldn't find it back by title "
            "— check `gh repo deploy-key list` by hand"
        )
    return str(matches[0]["id"])


def delete_deploy_key(repo: RepoRef, key_id: str) -> None:
    run(["gh", "repo", "deploy-key", "delete", key_id, "-R", repo.slug])


def repo_default_branch(repo: RepoRef) -> str:
    data = run_json(["gh", "repo", "view", repo.slug, "--json", "defaultBranchRef"])
    ref = data.get("defaultBranchRef") or {}
    return ref.get("name", "main")
