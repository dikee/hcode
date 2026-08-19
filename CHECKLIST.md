# `hcode run` build checklist

Tick `[x]` only when the item is actually done and verified, in the commit
that does it — never ahead of `main`, never batch-ticked at the end.

See the full design in the session that produced this file for context on
each item: what it does, and why it is built in this order.

## Port to Go (step 0 — finish before anything below starts)

- [ ] `internal/config` — ported (`config.py`)
- [ ] `internal/run` — subprocess wrapper (`run.py`)
- [ ] `internal/state` — `Instance`/`Repo` structs, JSON round-trip (`state.py`)
- [ ] `internal/github` — repo URL parsing, deploy-key add/delete (`github.py`)
- [ ] `internal/keys` — keypair generation (`keys.py`)
- [ ] `internal/hetzner` — server create/delete/describe, login-key verify (`hetzner.py`)
- [ ] `internal/sshutil` — every remote call, `copy_dir_to`'s trailing-`/.` merge semantics preserved exactly (`ssh_util.py`)
- [ ] `internal/provision` — base script, worktree-add command (`provision.py`)
- [ ] `internal/forwards` — `-L` shorthand parsing (`forwards.py`)
- [ ] `internal/naming` — instance/key-title generation (`naming.py`)
- [ ] `internal/attach` — repo-attach shared logic (`attach.py`)
- [ ] `cmd/hcode` — `cobra` root + all 8 existing subcommands (`cli.py` + `commands/*.py`)
- [ ] `workflow-help` text ported verbatim (`workflow_help.py`)
- [ ] Full parity check: every flag, every error message, every behavior from
      the README re-verified against the Go build, not assumed equivalent
- [ ] Live re-run of the worktrees/ops-dir end-to-end test (create with
      `--worktrees`/`--ops-dir`/`--post-clone`/`--post-worktree`, `pull`,
      `destroy` with the merge-not-overwrite check) against a real box, Go
      binary this time
- [ ] Old Python package removed from the repo once the Go build passes the
      above — no dual-maintenance period

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
