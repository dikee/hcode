from __future__ import annotations

import secrets


def generate_instance_name(repo_name: str) -> str:
    return f"{repo_name}-{secrets.token_hex(3)}"


def generate_key_title(instance_name: str, repo_name: str) -> str:
    return f"hcode-{instance_name}-{repo_name}-{secrets.token_hex(3)}"
