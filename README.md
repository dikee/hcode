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
hcode ssh <name> [--repo NAME] [--worktree LABEL] [-L PORT ...]
hcode pull <name> <remote-path> [local-path] [--repo NAME]
hcode workflow-help                      print the workflow below
```

`create` additionally takes `--worktrees N`, `--ops-dir <local-path>`,
`--post-clone <script>`, `--post-worktree <script>`, and `-L`/`--forward`
(repeatable) — see Workflow below. Run `hcode <command> --help` for the
full flag list on any of these.

## Workflow

The multi-lane shape this was built for: one orchestrator lane plus N
worker lanes, each in its own git worktree, sharing one ops folder for
coordination. (`hcode workflow-help` prints this same walkthrough.)

**1. Create the box** — main clone, worktrees, ops folder, secrets, repo
setup, a tunnel, one shot:

```sh
hcode create git@github.com:OWNER/REPO.git \
  --login-key-path ~/.ssh/hetzner_ed25519 \
  --env-file backend/.env \
  --worktrees 3 \
  --ops-dir ~/code/REPO_ops \
  --post-clone bin/bootstrap_cloud_box.sh \
  --post-worktree bin/bootstrap_cloud_box.sh \
  -L 8000 -L 5173
```

Clones `REPO` into `/root/code/REPO`, adds `REPO-cc2`/`REPO-cc3`/`REPO-cc4`
worktrees on their own `cc2/base`, `cc3/base`, `cc4/base` branches — each
also gets its own copy of `--env-file` (worktrees don't inherit untracked
files just by existing), copies the ops folder up, runs `REPO`'s own
`bin/bootstrap_cloud_box.sh` once for the main clone and once per
worktree (whatever a fresh box or a fresh lane needs — installing a
database server, provisioning a *per-lane* database, running migrations
— that's the repo's business, not `hcode`'s, so it's a script the repo
owns; `HCODE_WORKTREE_LABEL` in its environment tells it which case it's
in), then SSHes you into the main clone with `localhost:8000`/`:5173`
already tunneled. That first terminal is your orchestrator lane.
`--post-clone`, `--post-worktree`, and `-L` are all optional — skip
whichever the repo doesn't need.

**2. Log into Claude Code** in that terminal: `claude`

**3. Open one more terminal per worker lane**, landing directly in its
own worktree — separate branch, separate working directory, no lane can
clobber another's files:

```sh
hcode ssh <instance> --worktree cc2
hcode ssh <instance> --worktree cc3
hcode ssh <instance> --worktree cc4
```

`claude` in each.

**4. Ask the orchestrator lane to write task prompts** — it has the ops
folder to read and write (`ORCHESTRATOR_RULES.md`, `ORCH.md`, `task.md`/
`results.md` mailboxes), since `--ops-dir` put it outside every worktree,
visible to all of them at once. Copy the prompts into each worker
terminal.

**Along the way:**

- **See the UI/API yourself** — already tunneled if you passed `-L` at
  `create`. Otherwise, one dedicated terminal: `hcode ssh <instance> -L
  8000 -L 5173`, then hit `localhost:8000`/`:5173` on your own laptop —
  never the box's public IP.
- **Visual QA** — Claude Code on the box drives Playwright headlessly
  and reads its own screenshots with `Read` — no tunnel needed, it's
  local to the box. Pull one down yourself with `hcode pull <instance>
  <path> --repo REPO`.
- **Check on things** — `hcode status`, or `hcode status --reconcile`
  to also flag anything orphaned.

**When you're done:** `hcode destroy <instance>` — pulls the ops folder
back down to its original local path, warns you by name if any repo or
worktree has uncommitted or unpushed work before you confirm, then
deletes the box and every deploy key it held.

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

**The ops folder lives outside every worktree, on purpose.** A git
worktree only ever shows you its own branch's checked-out files — a
shared mailbox (`task.md`/`results.md` coordination) committed into the
repo would be invisible across worktrees the moment more than one
exists, silently breaking multi-lane coordination. `--ops-dir` copies a
local directory up as a sibling of the repo and its worktrees instead,
so every lane can see it. `destroy` pulls it back to its original local
path before deleting anything — it's the only copy of everything the
orchestrator wrote during the session, and nothing else syncs it back.

**`destroy` warns before it's too late to matter.** Before the confirm
prompt, it checks every repo and worktree for uncommitted changes or
commits that were never pushed, and names them explicitly instead of a
generic "destroy this box?" — `--yes` still skips the prompt, but the
warning still prints either way.

**`--post-clone`/`--post-worktree` run a script the repo owns, not one
`hcode` writes.** `hcode` doesn't know whether a given repo needs
Postgres, a specific `make` target, or nothing at all — that's
project-specific knowledge that belongs in the repo itself (e.g.
`bin/bootstrap_cloud_box.sh`), the same way `Makefile` targets already
are. `hcode` just runs whatever's named — once after cloning, and once
more per worktree with `HCODE_WORKTREE_LABEL` set, since a worktree
often needs something the main clone doesn't (its own database, most
commonly — four lanes sharing one database is exactly the collision a
multi-lane setup exists to avoid).

## Sizing

`create --type` defaults to `ccx33` (8 dedicated vCPUs, 32GB) —
`--type ccx23` (4 vCPU/16GB) is a reasonable floor for lighter or
single-lane sessions. Both are dedicated-vCPU plans; avoid the shared
`cpx*` line for anything running concurrent test suites — noisy
neighbors are exactly wrong for that workload.

## What create actually does

1. Verifies `--login-key-path` actually matches `--login-key` on
   Hetzner — fails before spending any money if it doesn't.
2. Parses `owner/repo` out of the SSH URL.
3. `hcloud server create`s the box from a boot script that installs
   git, uv, Node, Docker, and the Claude Code CLI — no secrets in it.
4. Waits for SSH, then for cloud-init to actually finish.
5. Generates a fresh ed25519 keypair, registers the public half as a
   read-write deploy key on the repo (`gh repo deploy-key add`), copies
   the private key up, clones the repo with it, and points the clone's
   `core.sshCommand` at that key so future `git push`/`pull` on the box
   keep working without extra setup. Copies any `--env-file`s to the
   same relative path in the clone, then forgets the local private key
   copy.
6. If `--worktrees N` was given, adds `N` more worktrees on fresh
   `cc<n>/base` branches, sharing the main clone's `.git` (and therefore
   its `core.sshCommand` — no separate key needed per worktree). Copies
   every `--env-file` into each one too — worktrees share tracked git
   content, never untracked files, so a `.env` in the main clone doesn't
   just appear in a new worktree on its own. If `--post-worktree` was
   given, runs it once per worktree afterward, with `HCODE_WORKTREE_LABEL`
   (`cc2`, `cc3`, ...) set so the script can do lane-specific setup.
7. If `--ops-dir` was given, copies it up as a sibling of the repo and
   its worktrees — never inside any single one of them.
8. If `--post-clone` was given, runs that script (path relative to the
   repo root) once against the main clone, with output streamed live
   since it can take a while.
9. Saves instance state, then SSHes you in with any `-L` forwards
   already active (unless `--no-attach`).
