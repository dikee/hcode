"""Parsing for -L/--forward specs — shared by `create` and `ssh`, which
both open an interactive session and can both usefully tunnel ports."""

from __future__ import annotations

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
