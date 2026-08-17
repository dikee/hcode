"""SSH/SCP to a box, as root, using the login key that was injected into
Hetzner at create time (never a repo's deploy key — those live only on
the box, and are only ever used by git there).

Every call here takes `identity_path` explicitly and passes
`-i <path> -o IdentitiesOnly=yes` — the box only trusts the one public
key registered with Hetzner for it, and that key almost never lives at
one of ssh's default-tried filenames (id_rsa, id_ed25519, ...), so
letting ssh guess would hang or silently try the wrong key.

Hetzner reuses IPv4 addresses across different servers over time, so a
freshly created box's IP may already sit in ~/.ssh/known_hosts from some
unrelated past host. hcode boxes are throwaway by design, so every call
here skips host-key pinning rather than risk a spurious "REMOTE HOST
IDENTIFICATION HAS CHANGED" failure — and never touches the user's real
known_hosts file to do it.
"""

from __future__ import annotations

import subprocess
import time
from pathlib import Path

from hetzner_code.run import HcodeError


def _ssh_opts(identity_path: Path) -> list[str]:
    return [
        "-i",
        str(identity_path),
        "-o",
        "IdentitiesOnly=yes",
        "-o",
        "StrictHostKeyChecking=no",
        "-o",
        "UserKnownHostsFile=/dev/null",
        "-o",
        "LogLevel=ERROR",
    ]


def wait_for_ssh(ip: str, identity_path: Path, *, timeout: int = 180) -> None:
    """Poll until the box accepts an SSH connection, then until
    cloud-init's boot script has actually finished — the connection
    coming up doesn't mean provisioning has."""
    opts = _ssh_opts(identity_path)
    deadline = time.monotonic() + timeout
    last_error = "connection never succeeded"
    while time.monotonic() < deadline:
        result = subprocess.run(
            ["ssh", *opts, "-o", "ConnectTimeout=5", f"root@{ip}", "true"],
            capture_output=True,
            text=True,
            check=False,
        )
        if result.returncode == 0:
            break
        last_error = result.stderr.strip()
        time.sleep(3)
    else:
        raise HcodeError(f"timed out waiting for SSH on {ip}: {last_error}")

    result = subprocess.run(
        ["ssh", *opts, f"root@{ip}", "cloud-init", "status", "--wait"],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        raise HcodeError(
            f"cloud-init failed on {ip} — SSH in and check `cloud-init status --long` "
            f"or /var/log/cloud-init-output.log\n{result.stderr.strip()}"
        )


def run_remote(
    ip: str, command: str, identity_path: Path
) -> subprocess.CompletedProcess:
    result = subprocess.run(
        ["ssh", *_ssh_opts(identity_path), f"root@{ip}", command],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        raise HcodeError(
            f"remote command failed on {ip}: {command}\n{result.stderr.strip()}"
        )
    return result


def copy_to(ip: str, local_path: Path, remote_path: str, identity_path: Path) -> None:
    run_remote(ip, f"mkdir -p {_shell_quote(_dirname(remote_path))}", identity_path)
    result = subprocess.run(
        ["scp", *_ssh_opts(identity_path), str(local_path), f"root@{ip}:{remote_path}"],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        raise HcodeError(f"scp to {ip}:{remote_path} failed\n{result.stderr.strip()}")


def copy_dir_to(ip: str, local_dir: Path, remote_dir: str, identity_path: Path) -> None:
    """Like copy_to but recursive — for a whole directory (the ops
    folder), not a single file."""
    run_remote(ip, f"mkdir -p {_shell_quote(_dirname(remote_dir))}", identity_path)
    result = subprocess.run(
        [
            "scp",
            "-r",
            *_ssh_opts(identity_path),
            str(local_dir),
            f"root@{ip}:{remote_dir}",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        raise HcodeError(f"scp -r to {ip}:{remote_dir} failed\n{result.stderr.strip()}")


def copy_from(ip: str, remote_path: str, local_path: Path, identity_path: Path) -> None:
    """Pull a file or directory back down. -r is harmless on a plain
    file, so one code path covers both without an extra round trip to
    ask the box which kind remote_path is."""
    local_path.parent.mkdir(parents=True, exist_ok=True)
    result = subprocess.run(
        [
            "scp",
            "-r",
            *_ssh_opts(identity_path),
            f"root@{ip}:{remote_path}",
            str(local_path),
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        raise HcodeError(f"scp from {ip}:{remote_path} failed\n{result.stderr.strip()}")


def interactive(
    ip: str,
    identity_path: Path,
    *,
    cwd: str | None = None,
    forwards: list[str] | None = None,
) -> int:
    """Interactive session — inherits this process's tty."""
    command = ["ssh", *_ssh_opts(identity_path)]
    for spec in forwards or []:
        command += ["-L", spec]
    command += ["-t", f"root@{ip}"]
    if cwd:
        command.append(f"cd {_shell_quote(cwd)} && exec bash -l")
    return subprocess.call(command)


def _dirname(remote_path: str) -> str:
    return remote_path.rsplit("/", 1)[0] if "/" in remote_path else "."


def _shell_quote(s: str) -> str:
    return "'" + s.replace("'", "'\\''") + "'"
