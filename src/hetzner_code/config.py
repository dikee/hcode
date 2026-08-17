"""Paths and defaults. Nothing here talks to a network."""

from __future__ import annotations

from pathlib import Path

STATE_DIR = Path.home() / ".hetzner-code"
INSTANCES_DIR = STATE_DIR / "instances"

DEFAULT_TYPE = "ccx33"
DEFAULT_LOCATION = "nbg1"
DEFAULT_IMAGE = "ubuntu-24.04"

# Every hcode-created server carries this label so `status --reconcile`
# can tell "ours, untracked locally" apart from "not ours at all" on an
# account that runs other things too.
OWNER_LABEL = {"managed-by": "hcode"}

REMOTE_CODE_DIR = "/root/code"
REMOTE_KEY_DIR = "/root/.ssh/hcode"
