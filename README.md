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
# from this repo (dev)
make build                 # → bin/ctx
./bin/ctx --version
./bin/ctx init /path/to/target-repo            # default folder .ctx/
./bin/ctx init /path/to/target-repo -f .agent   # custom folder name
./bin/ctx init                 # target defaults to the current directory

# or install (Go users)
go install github.com/donmclean/ctx/cmd/ctx@latest
```

`ctx init` creates `<target>/<.ctx>/` from the embedded templates, substitutes
`{{PROJECT}}`/`{{DATE}}`, writes a `.ctx-version` stamp, and adds the folder to
the target's `.git/info/exclude`. (`{{FOUNDER}}`/`{{COLLABORATOR}}`/
`{{OWNER_INSTRUCTIONS_PATH}}` are intentional user-fill placeholders — an agent
fills them per `docs/fill-context-workflow.md`.) Then point an agent at
`.ctx/INDEX.md`. Upcoming commands: `ctx update` (refresh a repo's `.ctx/`),
`ctx upgrade` (upgrade the CLI), `ctx doctor` (validate).

## Layout of this repo

```
ctx/
├── README.md                       # this file
├── Makefile                        # build / test / install / smoke
├── go.mod / go.sum                 # Go module (github.com/donmclean/ctx)
├── cmd/ctx/main.go                 # CLI entrypoint
├── pkg/ctx/                        # importable library + CLI commands
│   ├── root.go  init.go  git.go  embed.go  version.go
│   └── template/                   # the .ctx/ scaffold (embedded via go:embed)
├── docs/
│   ├── principles.md               # the 5 reusable principles
│   └── fill-context-workflow.md    # how an agent fills context/*.md for a repo
└── bin/                            # build output (gitignored)
```

## Status

Go CLI MVP (increment 1): `ctx init` + `ctx --version`, templates embedded via
`go:embed`, `.ctx-version` stamp, `.git/info/exclude` wiring. Upcoming:
`ctx update` (managed-block refresh via markers), `ctx upgrade` (CLI self-update),
`ctx doctor`, then goreleaser + GitHub Releases + npm launcher distribution.
Not a scripted auto-analyzer — an agent does the per-project analysis (guided by
the workflow doc), because the value is the *analysis*, not the file structure.