"""Per-repo SSH keypairs. One keypair per (instance, repo) pair, never
reused across repos, so removing one repo's access can never affect
another's — see the design note in README.md.
"""

from __future__ import annotations

from pathlib import Path

from hetzner_code.run import run


def generate(key_dir: Path, *, comment: str) -> tuple[Path, Path]:
    """Generate a passwordless ed25519 keypair at key_dir/id_ed25519{,.pub}."""
    key_dir.mkdir(parents=True, exist_ok=True)
    private = key_dir / "id_ed25519"
    public = key_dir / "id_ed25519.pub"
    run(
        [
            "ssh-keygen",
            "-t",
            "ed25519",
            "-N",
            "",
            "-C",
            comment,
            "-f",
            str(private),
        ]
    )
    return private, public


def forget_private_key(private_key_path: Path) -> None:
    """Delete the local private key copy once it's safely on the box.
    The public key file is left in place — it's public, and handy for
    'which key did I register' debugging without another gh round trip."""
    if private_key_path.exists():
        private_key_path.unlink()
