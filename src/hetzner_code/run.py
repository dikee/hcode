"""Thin subprocess wrapper. Every external call (hcloud, gh, ssh, scp)
goes through here so failures come back as one clear error type instead
of a bare CalledProcessError with no context about which step broke.
"""

from __future__ import annotations

import json
import subprocess


class HcodeError(RuntimeError):
    pass


def run(
    cmd: list[str], *, input: str | None = None, check: bool = True
) -> subprocess.CompletedProcess:
    """Run `cmd`, capturing stdout/stderr. Raises HcodeError with the
    command and stderr on failure unless check=False."""
    result = subprocess.run(
        cmd,
        input=input,
        capture_output=True,
        text=True,
        check=False,
    )
    if check and result.returncode != 0:
        raise HcodeError(
            f"command failed ({result.returncode}): {' '.join(cmd)}\n{result.stderr.strip()}"
        )
    return result


def run_json(cmd: list[str]):
    """Run `cmd` and parse its stdout as JSON. hcloud/gh both support
    -o json / --json on the calls this tool needs."""
    result = run(cmd)
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        raise HcodeError(
            f"expected JSON from: {' '.join(cmd)}\ngot: {result.stdout[:500]}"
        ) from exc
