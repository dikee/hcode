# `hcode run` build checklist

Tick `[x]` only when the item is actually done and verified, in the commit
that does it — never ahead of `main`, never batch-ticked at the end.

See the full design in the session that produced this file for context on
each item: what it does, and why it is built in this order.

## Port to Go (step 0 — finish before anything below starts)

- [x] `internal/config` — ported (`config.py`)
- [x] `internal/run` — subprocess wrapper (`run.py`)
- [x] `internal/state` — `Instance`/`Repo` structs, JSON round-trip (`state.py`)
- [x] `internal/github` — repo URL parsing, deploy-key add/delete (`github.py`) —
      `ParseRepoURL` unit-tested against the Python regex's exact case set; the
      live test below caught a real bug here (deploy-key id decoded as
      `float64` and mangled into scientific notation, e.g.
      `160723208` -> `"1.60723208e+08"`, which 404'd on delete) — fixed to
      decode as `int64`, regression-tested
- [x] `internal/keys` — keypair generation (`keys.py`)
- [x] `internal/hetzner` — server create/delete/describe, login-key verify (`hetzner.py`)
- [x] `internal/sshutil` — every remote call, `copy_dir_to`'s trailing-`/.` merge semantics preserved exactly (`ssh_util.py`)
- [x] `internal/provision` — base script, worktree-add command (`provision.py`)
- [x] `internal/forwards` — `-L` shorthand parsing (`forwards.py`) — unit-tested
      against the Python original's exact case set, including its `""` edge case
- [x] `internal/naming` — instance/key-title generation (`naming.py`)
- [x] `internal/attach` — repo-attach shared logic (`attach.py`)
- [x] `cmd/hcode` — `cobra` root + all 8 existing subcommands (`cli.py` + `commands/*.py`)
- [x] `workflow-help` text ported verbatim (`workflow_help.py`) — diffed
      byte-for-byte against the Python CLI's own output
- [x] Full parity check: every flag and help string on every subcommand
      diff-checked against the Python CLI's own `--help` output; error paths
      (`status` on an untracked name, a malformed repo URL, a missing
      required flag) spot-checked side by side. `--post-clone`/
      `--post-worktree` weren't separately live-tested — they run a plain
      shell command over the same `RunRemoteStreaming` path the worktree
      step below already exercised
- [x] Live re-run of the worktrees/ops-dir end-to-end test against a real
      box, Go binary this time: `create --worktrees 2 --ops-dir` against
      `dikee/inzu` (ccx23, nbg1) — main clone, both worktrees, and the
      copied-up ops dir all confirmed present via `pull`; `destroy` pulled
      the ops dir back with no nesting bug (files landed flat, not under a
      spurious `inzu_ops/` subdirectory — the case the trailing `/.` in
      `SyncDirFrom` exists to prevent) and cleaned up the server and deploy
      key. Caught the deploy-key id bug above; re-run confirmed clean after
      the fix
- [x] Old Python package removed from the repo now that the Go build has
      passed the above — `src/`, `pyproject.toml`, `uv.lock`,
      `.python-version` deleted, `.gitignore` and the README's install
      instructions updated for `go install`, no dual-maintenance period

## `hcode run` core pipeline

- [ ] `internal/tickets` — schema, dependency graph, lock graph, ready-set
- [ ] Unit tests: ready-set respects `depends_on`==`merged`; cycles rejected
- [ ] Unit tests: lock-sharing tickets never scheduled concurrently
- [ ] `internal/runstate` — `state.json` read/write/reconcile, resume path
- [ ] Resume re-runs the git-safety check on every `running` worktree
- [ ] `internal/workerdispatch` — single-ticket `claude -p` round trip, both
      auth modes, `--max-budget-usd`, `--ticket-timeout`
- [ ] Infra-failure vs genuine-difficulty classified correctly on retry
- [ ] `internal/scopecheck` — mechanical `allowed_paths`/protected-paths check
- [ ] `internal/review` — structured verdict, single-pass
- [ ] Multi-pass (majority-of-N) review wired to `--auto-merge`
- [ ] `--specs-dir` copy-up + `--add-dir` wiring; standing decision policy in
      `--append-system-prompt`; `judgment_calls` parsed from results
- [ ] Concurrency: goroutines + channels, `--workers` respected, merges
      serialized on the scheduler goroutine
- [ ] `--max-run-budget-usd` stops new dispatches, leaves in-flight alone
- [ ] Merge step: `full_gate` on the combined branch; `--auto-merge` vs
      one-pause-per-batch; `gh`/CI-verification decision explicit either way
- [ ] `--playwright` gate wired for `touches_ui: true` tickets

## Escape hatch (v1)

- [ ] `internal/notify` — localhost POST to `--notify-port`, generic body only
- [ ] `awaiting_human` status; worker slot frees immediately, no idle wait
- [ ] `hcode answer` — requeues with the answer in context
- [ ] `hcode pause` / `hcode cancel`
- [ ] Batch-ready-to-merge checkpoint fires the same notify path
- [ ] Local tray app: native notification, click-through panel, text answer
      (separate small project — Tauri, per the desktop-first decision)
- [ ] Live test: a deliberately consequential, spec-silent question — notify
      arrives, body has no question content, `hcode status --awaiting-human`
      shows the real question, `hcode answer` unblocks without disturbing
      other workers

## v2 — Flutter companion app (not started; separate project, own checklist when it starts)

- [ ] Relay server: hosting decision, outbound-websocket protocol from
      `hcode run`, persisted run/ticket state
- [ ] Device pairing flow (QR/token) — no unpaired device talks to anything
- [ ] APNs setup (own Apple Developer account/credentials)
- [ ] FCM setup (own Google/Firebase project)
- [ ] Flutter app: fleet view, run board, rich answer screen, Q&A history,
      batch-merge checkpoint, cost view (API + Hetzner combined), notification
      preferences
- [ ] Merge-approval requires deliberate confirmation, never a single tap
- [ ] Device-revocation path (lost phone ≠ incident)
