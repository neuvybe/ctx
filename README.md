# ctx — a reusable agent-context platform

`ctx` gives coding agents durable, project-specific context that survives
compaction and makes resuming work cheap. Its default **team mode** keeps stable
project knowledge available to share while isolating per-clone session state:

- durable guidance and reference docs live under `.ctx/` and can be tracked;
- the living session log lives at `.ctx/local/CONTINUE.md` and is ignored by
  `.ctx/.gitignore`;
- `ctx` creates and updates files, but never stages or commits them.

For work that must remain entirely private, `ctx init --mode local` keeps the
whole context folder out of Git through `.git/info/exclude`.

## What team mode creates

```text
.ctx/
├── .gitignore          # ignores local/
├── .ctx-version        # scaffold version
├── config.json         # schema version + team/local mode
├── README.md           # what this folder is + health rules
├── OPERATING.md        # stable operating mode (constitution)
├── INDEX.md            # repo orientation + load order
├── REVIEW.md           # pre-PR adversarial review workflow
├── local/
│   └── CONTINUE.md     # private living session state
└── context/
    ├── overview.md     # what the project is + what's not here yet
    ├── architecture.md # end-to-end flow + package responsibilities
    ├── format.md       # data formats, storage, and compatibility
    ├── extending.md    # project extension points and conventions
    ├── known-issues.md # races, rough edges, gotchas + fixes
    └── glossary.md     # domain-specific terms
```

Everything except `.ctx/local/` is available for the repository owner to review,
stage, and commit in team mode. The CLI deliberately leaves those Git decisions
to the user.

## Why these choices

- **Durable knowledge can be shared.** `OPERATING.md`, `INDEX.md`, `REVIEW.md`,
  and `context/*.md` describe the project and its working conventions, so teams
  can review and improve them like other documentation.
- **Session state stays local.** `.ctx/local/CONTINUE.md` changes frequently and
  may contain machine- or session-specific state. `.ctx/.gitignore` keeps it out
  of commits without hiding the durable docs.
- **Fully private mode remains available.** `--mode local` adds the whole context
  folder to the target repository's `.git/info/exclude`; it does not edit the
  repository's root `.gitignore`.
- **Constitution and log stay separate.** `OPERATING.md` holds stable rules;
  `local/CONTINUE.md` holds living state, so routine updates do not re-litigate
  the operating mode.
- **The owner's instructions govern.** Canonical owner instructions such as
  `AGENTS.md` or `CLAUDE.md` take precedence. `.ctx/OPERATING.md` supplements
  them; it never overrides them.
- **Resume after compaction.** A fresh agent loads `OPERATING.md` →
  `local/CONTINUE.md` → `INDEX.md` → the relevant `context/*.md`.

## Install and initialize

```bash
# npm launcher
npm install -g @neuvybe/ctx
ctx --version

# default: team mode
ctx init /path/to/target-repo

# opt in to a wholly private, repo-local scaffold
ctx init /path/to/target-repo --mode local

# custom folder; target defaults to the current directory
ctx init /path/to/target-repo --folder .agent
ctx init
```

For `ctx init`, `--folder` must be one top-level directory name containing only
letters, digits, `.`, `_`, or `-`. Nested paths such as `docs/ctx` and names
containing spaces are not supported.

On a fresh clone of a team scaffold, run `ctx init` again (with the same
`--folder`, if customized). It recognizes the committed team configuration and
creates only the ignored `local/CONTINUE.md`; it does not rewrite durable files.

For development from this repository:

```bash
make build                 # → bin/ctx
./bin/ctx --version
./bin/ctx init /path/to/target-repo
```

Go users can also install with:

```bash
go install github.com/neuvybe/ctx/cmd/ctx@latest
```

For Go embedders, `InitWithOptions` makes the mode explicit. The older exported
`Init(repo, folder)` wrapper preserves its original whole-folder-local behavior
and root `CONTINUE.md` layout so existing integrations do not become shareable
or change paths silently.

`ctx init` substitutes `{{PROJECT}}` and `{{DATE}}`, writes a `.ctx-version`
stamp, and leaves `{{FOUNDER}}`, `{{COLLABORATOR}}`, and
`{{OWNER_INSTRUCTIONS_PATH}}` for the fill workflow. In team mode it writes
`.ctx/.gitignore` for `local/`; in local mode it adds the whole selected folder
to `.git/info/exclude`. It never runs `git add` or `git commit`.

After filling a team scaffold, review the durable files and decide explicitly
whether to share them:

```bash
git status --short .ctx
git add .ctx               # optional; local/ remains ignored
git commit                 # always user-controlled
```

The example uses the default `.ctx` folder. Replace it with the selected custom
folder everywhere, and repeat `--folder <name>` on `update` and `doctor`.

See [`docs/fill-context-workflow.md`](docs/fill-context-workflow.md) for the
agent-guided analysis process.

## Commands

- `ctx init [target] [--folder .ctx] [--mode team|local]` — create a scaffold;
  team mode is the default.
- `ctx update [target] [--folder .ctx]` — refresh managed blocks in `README.md` and `REVIEW.md`,
  preserve user-owned content, and bump `.ctx-version`. It preserves the
  scaffold's sharing mode and never touches `local/CONTINUE.md`.
- `ctx doctor [target] [--folder .ctx]` — validate the scaffold, version stamp, ignore policy,
  placeholders, managed markers, and expected files.
- `ctx upgrade` — self-update a direct binary from the latest GitHub release or
  print the appropriate package-manager command.
- `ctx --version` — print the installed version.

## Existing local scaffolds

Scaffolds created before team mode remain local: the whole folder stays excluded
through `.git/info/exclude`, and their continuation remains at
`<folder>/CONTINUE.md`. New team and local scaffolds use
`<folder>/local/CONTINUE.md`. `ctx update` does not silently move or expose a
legacy scaffold; converting one to team mode will require an explicit future
conversion flow.

See the [`0.2` migration guide](docs/migrations/0.2-team-mode.md) for the exact
default change, compatibility behavior, and opt-out command.

Git's repository-local exclude is shared by linked worktrees. To prevent one
worktree from hiding another's tracked or team context, `ctx` rejects local
initialization when a sibling worktree has tracked content or a team scaffold
for the same folder.

## Layout of this repository

```text
ctx/
├── .github/workflows/               # PR checks + releases
├── cmd/ctx/main.go                   # CLI entrypoint
├── docs/                             # principles + fill workflow
├── npm/                              # published npm launcher
├── pkg/ctx/                          # library, commands, tests, templates
├── tools/                            # release/version/binary tooling
├── Makefile
├── go.mod / go.sum
└── package.json / package-lock.json  # commit + release tooling
```

## Status

The CLI ships `init`, `update`, `doctor`, `upgrade`, embedded templates, GitHub
release binaries, and the `@neuvybe/ctx` npm launcher. It is intentionally not a
scripted auto-analyzer: an agent performs the per-project analysis because the
value is the accuracy of the context, not merely the file structure.
