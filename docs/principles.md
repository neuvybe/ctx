# Principles

The five reusable principles encoded by `ctx`. The file structure is only the
vehicle; the durable value is accurate context with a clear sharing boundary.

## 1. Share durable knowledge; keep session state local

Default **team mode** separates the two kinds of context:

- Stable guidance and project reference docs (`OPERATING.md`, `INDEX.md`,
  `REVIEW.md`, and `context/*.md`) live under `.ctx/` and are available for the
  repository owner to review, stage, and commit.
- The living session log lives at `.ctx/local/CONTINUE.md`. A scoped
  `.ctx/.gitignore` ignores `local/`, so clone- and agent-specific state does not
  enter shared history.
- `ctx` never runs `git add` or `git commit`. It creates and updates files; the
  user decides what becomes shared repository history.

When all context must remain private, `ctx init --mode local` adds the whole
selected folder to `.git/info/exclude`. This is repo-local and non-tracked; it
does not modify the owner's root `.gitignore` or a machine-global excludes file.

Existing scaffolds from before team mode remain whole-folder local. Updates do
not silently make them trackable or relocate their state; joining team mode will
require an explicit future conversion flow.

Because `.git/info/exclude` is shared by linked worktrees, local mode is a
repository-wide choice for a given folder. `ctx` refuses to initialize it when
a sibling worktree has tracked content or uses team mode for that folder.

## 2. Constitution vs. log

Keep **stable rules** (`OPERATING.md`) separate from **living state**
(`local/CONTINUE.md`).

- `OPERATING.md` changes only when the work mode is ratified — it is the constitution.
- `local/CONTINUE.md` changes every session — it is the private log (work mode,
  last-completed, in-flight, proposed-next, parked concepts, decisions log).

Separating them makes iteration cheap: update local state without re-litigating
the mode or generating shared churn. A compaction only costs re-reading
`OPERATING.md` → `local/CONTINUE.md` → `INDEX.md`, plus the relevant
`context/*.md` files.

## 3. The owner's canonical instructions govern

Wherever the project owner puts canonical agent instructions (root `CLAUDE.md`,
`AGENTS.md`, `.claude/skills/`, or `CONTRIBUTING`), **those take precedence**.
`.ctx/OPERATING.md` is a session-discipline supplement, never an override. If the
two conflict, the owner's rules win; surface the conflict rather than acting on
it.

This makes `ctx` safe in repositories that already have owner-authored agent
instructions: it complements rather than competes with them.

## 4. Review findings are proposals, not patches

Any second-agent review, such as the `codex review` pass in `REVIEW.md`, is
**advisory**. Surface findings individually to the human for triage; only approved
fixes are incorporated. Never silently mutate intent under the guise of fixing a
finding.

## 5. The value is the analysis, not the structure

`context/*.md` is useful only when it is accurate to the actual code. A blank
template supplies shape; an agent must read the repository and write honest,
verified docs for each category. `ctx` therefore ships a
[`fill-context-workflow.md`](fill-context-workflow.md), not an auto-analyzer.
Reconcile shared context when the repository moves. Stale context is worse than
none, so verify source before relying on a detail in a build step.
