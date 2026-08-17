"""Everything that talks to the Hetzner API runs here, via the `hcloud`
CLI (assumed already authenticated — `hcloud context list`)."""

from __future__ import annotations

from pathlib import Path

from hetzner_code.config import DEFAULT_IMAGE, OWNER_LABEL
from hetzner_code.run import HcodeError, run, run_json


def pick_login_key() -> str:
    """The SSH key hcode injects for the human's own access — the first
    key in the account if the caller didn't name one with --login-key.
    This is separate from the per-repo deploy keys: it's for *you*
    logging in, not for git operations."""
    keys = run_json(["hcloud", "ssh-key", "list", "-o", "json"])
    if not keys:
        raise HcodeError(
            "no SSH key registered with Hetzner yet. Run:\n"
            "  hcloud ssh-key create --name laptop --public-key-from-file ~/.ssh/id_ed25519.pub\n"
            "then retry, or pass --login-key explicitly."
        )
    return keys[0]["name"]


def verify_login_key(login_key: str, login_key_path: Path) -> None:
    """Fail fast, before spending money on a server, if --login-key-path
    doesn't actually match what's registered as --login-key on Hetzner —
    otherwise the mismatch only surfaces as a mysterious SSH hang after
    the box is already up and billing."""
    public_key_path = login_key_path.with_suffix(login_key_path.suffix + ".pub")
    if not public_key_path.exists():
        raise HcodeError(
            f"--login-key-path {login_key_path} has no matching {public_key_path.name} "
            "next to it — pass the private key path, not the public one"
        )
    local_material = _key_material(public_key_path.read_text())

    info = run_json(["hcloud", "ssh-key", "describe", login_key, "-o", "json"])
    remote_material = _key_material(info["public_key"])

    if local_material != remote_material:
        raise HcodeError(
            f"{login_key_path} doesn't match the key registered as '{login_key}' on Hetzner "
            f"— SSH would hang after the box comes up. Pass the right --login-key-path, or "
            f"register this one: hcloud ssh-key create --name <n> "
            f"--public-key-from-file {public_key_path}"
        )


def _key_material(pub_key_text: str) -> str:
    """type + base64 blob only — drop the trailing comment, which
    differs between the local file and what Hetzner echoes back."""
    parts = pub_key_text.strip().split()
    return " ".join(parts[:2])


def create_server(
    *,
    name: str,
    server_type: str,
    location: str,
    login_key: str,
    user_data_file: Path,
    labels: dict[str, str],
) -> tuple[str, str]:
    """Create the server, return (server_id, ipv4)."""
    label_flags = []
    for k, v in {**OWNER_LABEL, **labels}.items():
        label_flags += ["--label", f"{k}={v}"]

    run(
        [
            "hcloud",
            "server",
            "create",
            "--name",
            name,
            "--type",
            server_type,
            "--image",
            DEFAULT_IMAGE,
            "--location",
            location,
            "--ssh-key",
            login_key,
            "--user-data-from-file",
            str(user_data_file),
            *label_flags,
        ]
    )
    info = run_json(["hcloud", "server", "describe", name, "-o", "json"])
    server_id = str(info["id"])
    ipv4 = info["public_net"]["ipv4"]["ip"]
    return server_id, ipv4


def delete_server(server_id: str) -> None:
    run(["hcloud", "server", "delete", server_id])


def describe_server(server_id: str):
    """Live server info, or None if it no longer exists (deleted out of
    band — the case `status --reconcile` needs to catch)."""
    result = run(["hcloud", "server", "describe", server_id, "-o", "json"], check=False)
    if result.returncode != 0:
        return None
    import json

    return json.loads(result.stdout)


def list_managed_servers():
    """Every server on the account carrying hcode's owner label —
    the universe `status --reconcile` compares local state against."""
    label_selector = ",".join(f"{k}={v}" for k, v in OWNER_LABEL.items())
    return run_json(["hcloud", "server", "list", "-l", label_selector, "-o", "json"])
