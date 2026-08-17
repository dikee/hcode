# hetzner-code

Disposable Hetzner boxes for running Claude Code, each wired to one or
more git repos through a per-repo GitHub deploy key. Built to move heavy
multi-lane Claude Code work off a laptop and onto cloud compute you spin
up for a session and tear down when you're done.

## Prerequisites

- `hcloud` — authenticated (`hcloud context create ...`)
- `gh` — authenticated with `repo` scope (`gh auth status`)
- An SSH key already registered with Hetzner for *your own* login access —
  separate from the per-repo deploy keys `hcode` generates itself. If you
  don't have one yet:
  ```sh
  ssh-keygen -t ed25519 -N "" -C "hetzner-login" -f ~/.ssh/hetzner_ed25519
  hcloud ssh-key create --name laptop --public-key-from-file ~/.ssh/hetzner_ed25519.pub
  ```
  `create` defaults `--login-key-path` to `~/.ssh/id_ed25519` — if yours
  lives elsewhere (as above), pass `--login-key-path ~/.ssh/hetzner_ed25519`.
  `create` verifies the local key actually matches what's registered
  before it spends any money, so a mismatch here fails fast, not as a
  mysterious hang after the box is already up.

## Install

```sh
cd ~/code/hetzner-code
uv sync
uv tool install --editable .   # puts `hcode` on your PATH
```

## Commands

```
hcode create <git-ssh-url> [options]     create a box, clone the repo onto it
hcode add <git-ssh-url> --instance NAME  clone a second codebase onto an existing box
hcode remove <repo-name> --instance NAME remove one codebase, keep the box up
hcode destroy <name> | --all             tear down a box and every deploy key on it
hcode status [name] [--json] [--reconcile]
hcode ssh <name> [--repo NAME]
```

Run `hcode <command> --help` for the full flag list.

## Design choices, and why

**`gh` and `hcloud` only ever run on your machine, never on the box.**
The box receives a narrow, revocable credential (one deploy key per
repo); it never holds anything that could create or delete infrastructure
or repo access on its own.

**Claude Code auth is interactive, on purpose.** `create` provisions the
box and drops you into an SSH session; you run `claude` yourself to log
in. No API key is minted or copied automatically — one manual step, in
exchange for never having a long-lived credential written to a box that
gets created and destroyed repeatedly.

**One SSH keypair per (instance, repo), never shared.** Removing one
codebase's access (`hcode remove`) can never accidentally affect another
codebase on the same box, because they were never the same key. The
private half is deleted from your laptop immediately after it's copied to
the box — the only place it ends up living is the box itself, which
`destroy` deletes along with everything else.

**Secrets (`.env` files) travel over SSH after boot, never through
Hetzner's user-data.** User-data is stored by Hetzner as plaintext and
readable via their API/console — fine for installing packages, wrong for
a Resend or S3 key. Pass `--env-file <local-path>` (repeatable); each one
is copied to the same relative path inside the cloned repo on the box.
`hcode` never keeps its own copy.

**Local state, not Hetzner labels, is the source of truth for what
`hcode` manages** — `~/.hetzner-code/instances/<name>/meta.json`. Every
box still carries an `hcode` label so `status --reconcile` can catch
drift (a box that exists on Hetzner but isn't tracked locally, or a
tracked instance whose box is already gone) — see that command's output
if a create/destroy is interrupted mid-way.

**Running two codebases on one box is your call, not something `hcode`
second-guesses.** `add`/`remove` let you do it. Nothing stops you from
oversubscribing the CPU — that tradeoff is on you to make per session,
not something worth a flag.

## Sizing

`create --type` defaults to `ccx33` (8 dedicated vCPUs, 32GB) —
`--type ccx23` (4 vCPU/16GB) is a reasonable floor for lighter or
single-lane sessions. Both are dedicated-vCPU plans; avoid the shared
`cpx*` line for anything running concurrent test suites — noisy
neighbors are exactly wrong for that workload.

## What create actually does

1. Parses `owner/repo` out of the SSH URL.
2. Generates a fresh ed25519 keypair, registers the public half as a
   read-write deploy key on the repo (`gh repo deploy-key add`).
3. `hcloud server create`s the box from a boot script that installs
   git, uv, Node, Docker, and the Claude Code CLI — no secrets in it.
4. Waits for SSH, then for cloud-init to actually finish.
5. Copies the private key up, clones the repo with it, and points the
   clone's `core.sshCommand` at that key so future `git push`/`pull` on
   the box keep working without extra setup.
6. Copies any `--env-file`s to the same relative path in the clone.
7. Forgets the local private key copy.
8. Saves instance state, then SSHes you in (unless `--no-attach`).
