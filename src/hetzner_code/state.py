"""Local state: one JSON file per instance under ~/.hetzner-code/instances/.

This is the only source of truth hcode trusts for "what did I create."
`status --reconcile` cross-checks it against hcloud/GitHub directly,
since a crash between steps can leave this file out of sync with the
real world — see hetzner_code.commands.status.
"""

from __future__ import annotations

import json
from dataclasses import asdict, dataclass, field

from hetzner_code.config import INSTANCES_DIR
from hetzner_code.run import HcodeError


@dataclass
class Repo:
    url: str  # the original git@github.com:owner/repo.git the user passed
    owner: str
    name: str  # short repo name, also the directory name on the box
    branch: str | None
    deploy_key_id: str  # GitHub's numeric id for this deploy key, for delete
    deploy_key_title: str


@dataclass
class Instance:
    name: str
    server_id: str
    ip: str
    type: str
    location: str
    login_key: str  # the hcloud SSH key *name* injected into the box
    login_key_path: (
        str  # local private key path used to actually SSH in as that identity
    )
    created_at: (
        str  # ISO 8601, set by the caller (Date.now() equivalents forbidden here)
    )
    repos: list[Repo] = field(default_factory=list)

    def to_json(self) -> str:
        return json.dumps(asdict(self), indent=2)

    @classmethod
    def from_json(cls, text: str) -> Instance:
        data = json.loads(text)
        data["repos"] = [Repo(**r) for r in data.get("repos", [])]
        return cls(**data)


def _instance_dir(name: str):
    return INSTANCES_DIR / name


def _meta_path(name: str):
    return _instance_dir(name) / "meta.json"


def save(instance: Instance) -> None:
    _instance_dir(instance.name).mkdir(parents=True, exist_ok=True)
    _meta_path(instance.name).write_text(instance.to_json())


def load(name: str) -> Instance:
    path = _meta_path(name)
    if not path.exists():
        raise HcodeError(f"no tracked instance named '{name}' (see `hcode status`)")
    return Instance.from_json(path.read_text())


def exists(name: str) -> bool:
    return _meta_path(name).exists()


def list_all() -> list[Instance]:
    if not INSTANCES_DIR.exists():
        return []
    out = []
    for child in sorted(INSTANCES_DIR.iterdir()):
        meta = child / "meta.json"
        if meta.exists():
            out.append(Instance.from_json(meta.read_text()))
    return out


def delete(name: str) -> None:
    import shutil

    d = _instance_dir(name)
    if d.exists():
        shutil.rmtree(d)


def key_dir(instance_name: str, repo_name: str):
    """Where a repo's local keypair lives before/while it's being
    uploaded. The private half is deleted after a successful scp —
    see hetzner_code.keys.forget_private_key."""
    return _instance_dir(instance_name) / "keys" / repo_name
