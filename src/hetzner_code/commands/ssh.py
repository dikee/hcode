from __future__ import annotations

from pathlib import Path

from hetzner_code import ssh_util, state
from hetzner_code.config import REMOTE_CODE_DIR
from hetzner_code.run import HcodeError


def normalize_forward(spec: str) -> str:
    """Accept ssh's own `-L` syntax, plus shorthand:

    - "8000"               -> "8000:localhost:8000"
    - "8000:9000"           -> "8000:localhost:9000"  (different remote port)
    - "8000:localhost:9000" -> unchanged (full form, any remote host)
    """
    parts = spec.split(":")
    if len(parts) == 1:
        return f"{parts[0]}:localhost:{parts[0]}"
    if len(parts) == 2:
        return f"{parts[0]}:localhost:{parts[1]}"
    if len(parts) == 3:
        return spec
    raise HcodeError(
        f"--forward {spec!r} doesn't look like PORT, PORT:PORT, or PORT:HOST:PORT"
    )


def ssh(*, instance_name: str, repo_name: str | None, forwards: tuple[str, ...]) -> int:
    instance = state.load(instance_name)
    cwd = None
    if repo_name:
        if not any(r.name == repo_name for r in instance.repos):
            available = ", ".join(r.name for r in instance.repos) or "(none)"
            raise HcodeError(
                f"'{repo_name}' isn't on '{instance_name}'. On it: {available}"
            )
        cwd = f"{REMOTE_CODE_DIR}/{repo_name}"
    elif len(instance.repos) == 1:
        cwd = f"{REMOTE_CODE_DIR}/{instance.repos[0].name}"

    normalized = [normalize_forward(f) for f in forwards]
    return ssh_util.interactive(
        instance.ip, Path(instance.login_key_path), cwd=cwd, forwards=normalized
    )
