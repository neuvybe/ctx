# ctx — a reusable agent-context platform

`ctx` is a small platform for giving a coding agent **durable, project-specific
context** that survives context compaction and makes resuming a session cheap.
It's the extraction of a working system proven on a real repo: a private
`.ctx/` folder (gitignored via `.git/info/exclude`, not the repo's
`.gitignore`) holding a **constitution-vs-log** split of governing files plus a
**reference-context schema** for the project.

## What you get in a target repo

```
.ctx/
├── README.md          # what this folder is + health rules
├── OPERATING.md       # the binding operating mode (constitution; stable)
├── CONTINUE.md        # living session state (resume protocol + state)
├── INDEX.md           # repo orientation + load order
├── REVIEW.md          # pre-PR `codex review` adversarial pass
└── context/
    ├── overview.md        # what the project is + what's not here yet
    ├── architecture.md    # end-to-end flow + package responsibilities
    ├── format.md          # on-disk formats / persistence / caching
    ├── extending.md       # how to add a language/provider/query/tool/etc.
    ├── known-issues.md    # races, rough edges, gotchas + fixes
    └── glossary.md        # domain terms
```

## Why these choices

- **Private, not tracked.** `.ctx/` is ignored via `.git/info/exclude` —
  repo-local, non-tracked, survives syncing, and **never touches the project's
  `.gitignore`** or the owner's tracked instruction namespace (root `CLAUDE.md`,
  `AGENTS.md`, `.claude/skills/`, …). It's the collaborator's private working
  context, not shared project knowledge.
- **Constitution vs. log.** `OPERATING.md` (stable rules of engagement) is
  separate from `CONTINUE.md` (living state that changes every session), so
  iteration is cheap — you update state without re-litigating the mode.
- **Owner's instructions govern.** Wherever the project owner puts canonical
  agent instructions, those take precedence; `OPERATING.md` is a *supplement*,
  never an override.
- **Resume after compaction.** A fresh agent loads `OPERATING.md` →
  `CONTINUE.md` → `INDEX.md` → the relevant `context/*.md` and is fully back in
  context for a fraction of the token cost of re-deriving it.

## Quick start

```bash
# from this platform repo
./init /path/to/target-repo            # uses default folder .ctx/
./init /path/to/target-repo .agent     # custom folder name
```

`init` creates `<target>/<.ctx>/` from `template/`, appends the folder to the
target's `.git/info/exclude`, and fills `{{PROJECT}}` / `{{DATE}}` placeholders.
Then point an agent at `.ctx/INDEX.md` (or follow
[`docs/fill-context-workflow.md`](docs/fill-context-workflow.md) to have an
agent produce accurate `context/*.md` by reading the repo).

## Layout of this platform repo

```
ctx/
├── README.md                       # this file
├── init                            # scaffold script
├── docs/
│   ├── principles.md               # the 5 reusable principles
│   └── fill-context-workflow.md    # how an agent fills context/*.md for a repo
└── template/                       # the .ctx/ scaffold (parameterized)
```

## Status

Early MVP: template + init + workflow doc. Not a scripted auto-analyzer — an
agent does the per-project analysis (guided by the workflow doc), because the
value is the *analysis*, not the file structure. Upgradeability across
projects is a later concern (MVP is copy-once-and-own).