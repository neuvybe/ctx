# Principles

The five reusable principles encoded by `ctx`. These are the hard-won parts —
the file structure is just the vehicle.

## 1. Private context uses `.git/info/exclude`, never the repo's `.gitignore`

A collaborator's working context (session state, private notes, the operating
mode) is **private and repo-local**. Ignore it via the target repo's
`.git/info/exclude` — **not** the repo's `.gitignore`.

- **Why not `.gitignore`:** that's the project owner's tracked file; editing it
  is a PR the owner reviews, and it pollutes the shared repo with your personal
  tooling.
- **Why not a global `core.excludesFile`:** that's machine-global and affects
  every repo; `.git/info/exclude` is repo-scoped, zero side-effects, and
  survives syncing to `origin/main` (unlike a folder that's only ignored on one
  branch).
- **Never collide with the owner's tracked instruction namespace** (root
  `CLAUDE.md`, `AGENTS.md`, `.claude/skills/`). Pick a folder name the owner
  isn't using (`.ctx` default; configurable).

**Clone-local caveat:** `.git/info/exclude` doesn't follow the repo to a fresh
clone. For personal/session context that's appropriate. On re-clone, re-run
`init` (or keep your `.ctx/` copied elsewhere).

## 2. Constitution vs. log

Keep **stable rules** (`OPERATING.md`) separate from **living state**
(`CONTINUE.md`).

- `OPERATING.md` changes only when the mode is ratified — it's the constitution.
- `CONTINUE.md` changes every session — it's the log (mode, last-completed,
  in-flight, proposed-next, parked-concepts, decisions-log).

Separating them means iteration is cheap: you update state without re-litigating
the mode. A compaction only costs re-reading the three governing files
(`OPERATING` + `CONTINUE` + `INDEX`) plus 1–2 relevant `context/*.md` — not the
whole folder.

## 3. The owner's canonical instructions govern

Wherever the project owner puts canonical agent instructions (root `CLAUDE.md`,
`AGENTS.md`, `.claude/skills/`, a `CONTRIBUTING`), **those take precedence**.
`.ctx/OPERATING.md` is the collaborator's *session-discipline supplement*,
layered on top — never an override. If the two conflict, the owner's rules win;
surface the conflict rather than acting on it.

This is what makes `ctx` safe to drop into a repo that already has owner-authored
agent instructions: it complements, doesn't compete.

## 4. Review findings are proposals, not patches

Any second-agent review (e.g. the `codex review` pre-PR pass in `REVIEW.md`) is
**advisory**: findings are surfaced per-finding to the human, who triages. Only
approved fixes get incorporated. Never let a reviewer silently mutate intent
under the guise of "fixing." (`OPERATING.md` codifies this; `REVIEW.md` is the
concrete pass.)

## 5. The value is the analysis, not the structure

`context/*.md` is only useful when it's **accurate to the actual code**. A blank
template gives you the shape; an agent has to read the repo and write honest,
verified docs per category. So `ctx` ships a **fill-context workflow**
([`fill-context-workflow.md`](fill-context-workflow.md)), not an auto-analyzer.
Reconcile docs against upstream when the repo moves; stale context is worse than
none — verify against source before relying on a detail in a build step.