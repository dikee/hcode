"""The canonical multi-lane workflow, printed by `hcode workflow-help`
and mirrored into README.md's "Workflow" section — keep both in sync
when this changes."""

from __future__ import annotations

WORKFLOW_HELP = """\
A multi-lane Claude Code workflow on one box: one orchestrator lane plus
N worker lanes, each in its own git worktree, sharing one ops folder for
coordination.

1. Create the box — main clone, worktrees, ops folder, secrets, repo
   setup, a tunnel, one shot:

     hcode create git@github.com:OWNER/REPO.git \\
       --login-key-path ~/.ssh/hetzner_ed25519 \\
       --env-file backend/.env \\
       --worktrees 3 \\
       --ops-dir ~/code/REPO_ops \\
       --post-clone bin/bootstrap_cloud_box.sh \\
       -L 8000 -L 5173

   Clones REPO into /root/code/REPO, adds REPO-cc2/REPO-cc3/REPO-cc4
   worktrees on their own cc2/base, cc3/base, cc4/base branches, copies
   backend/.env and the ops folder up, runs REPO's own bin/bootstrap_
   cloud_box.sh (whatever a fresh box needs — installing a database
   server, running migrations — that's the repo's business, not hcode's,
   so it's a script the repo owns), then SSHes you into the main clone
   with localhost:8000/:5173 already tunneled. That first terminal is
   your orchestrator lane. --post-clone and -L are both optional — skip
   either if the repo needs no setup or you'd rather tunnel later.

2. Log into Claude Code in that terminal:

     claude

3. Open one more terminal per worker lane, landing directly in its own
   worktree — separate branch, separate working directory, no lane can
   clobber another's files:

     hcode ssh <instance> --worktree cc2
     hcode ssh <instance> --worktree cc3
     hcode ssh <instance> --worktree cc4

   `claude` in each.

4. Ask the orchestrator lane to write task prompts — it has the ops
   folder to read and write (ORCHESTRATOR_RULES.md, ORCH.md, task.md/
   results.md mailboxes), since --ops-dir put it outside every worktree,
   visible to all of them at once. Copy the prompts into each worker
   terminal.

Along the way:

  See the UI/API yourself       Already tunneled if you passed -L at
                                 create. Otherwise, one dedicated terminal:
                                   hcode ssh <instance> -L 8000 -L 5173
                                 Then hit localhost:8000 / :5173 on your
                                 own laptop — never the box's public IP.

  Visual QA                     Claude Code on the box drives Playwright
                                 headlessly and reads its own screenshots
                                 with Read — no tunnel needed, it's local
                                 to the box. Pull one down yourself with:
                                   hcode pull <instance> <path> --repo REPO

  Check on things                 hcode status
                                   hcode status --reconcile   (flags orphans)

When you're done:

     hcode destroy <instance>

  Pulls the ops folder back down to its original local path, warns you
  by name if any repo or worktree has uncommitted or unpushed work
  before you confirm, then deletes the box and every deploy key it held.
"""
