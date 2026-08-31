# Fill-context workflow

How an agent produces accurate `context/*.md` for a target repository. The value
of `.ctx/` is the analysis, not the file structure: a blank template is useless
until an agent reads the repository and writes honest, verified documentation.

## Know the scaffold mode

- `ctx init` defaults to **team mode**. Durable docs under `.ctx/` are available
  to share, while `.ctx/local/CONTINUE.md` is ignored by `.ctx/.gitignore`.
- `ctx init --mode local` opts into a wholly private scaffold. The entire selected
  folder is ignored through the target repository's `.git/info/exclude`.
- `ctx` never stages or commits files in either mode. A human owns every decision
  to add durable team docs to repository history.
- Legacy pre-team scaffolds stay local. `ctx update` does not silently convert
  their layout or visibility: their continuation remains at
  `<folder>/CONTINUE.md`, while new scaffolds use
  `<folder>/local/CONTINUE.md`. An explicit future conversion flow will be
  needed.

## Principles for the agent doing the filling

- **Read the actual code; don't infer from commit messages or READMEs alone.**
  Verify every claim against source before writing it.
- **Reconcile against upstream before relying on a detail.** If the repository
  has moved since a doc was written, the doc is stale; re-verify it.
- **Note what you did not verify** rather than papering over it.
- **Keep it scannable.** Each `context/*.md` should be loadable in one screen for
  the relevant task.
- **Respect the sharing boundary.** Durable project facts belong in the shared
  docs in team mode; transient, machine-specific, or session state belongs only
  in the configured continuation file (`local/CONTINUE.md` for new scaffolds,
  root `CONTINUE.md` for legacy scaffolds).

## Order of operations

1. **Orient.** Read the root `README.md`, owner agent instructions (for example
   root `CLAUDE.md`, `AGENTS.md`, or `.claude/skills/`), build files, dependency
   manifests, and the top-level tree. Note the project's purpose, status, and the
   owner's rules; those rules govern (see `principles.md` §3).
2. **`context/overview.md` first** — the thesis, core mechanism, and what is not
   here yet. This forces an understanding of the whole before the parts.
3. **`context/architecture.md`** — end-to-end data flow, package responsibilities,
   concurrency model, and integration paths. Read the entrypoints and trace one
   full flow.
4. **`context/format.md`** — data formats, storage boundaries, and compatibility
   constraints. Read the relevant encoding, storage, and migration code.
5. **`context/extending.md`** — project extension points and the conventions for
   adding a new capability. Defer to an owner-authored authoritative guide
   where one exists.
6. **`context/known-issues.md`** — races, rough edges, build/tooling gotchas, and
   test gaps. Run tests, build, and vet; distinguish environment failures from
   product bugs.
7. **`context/glossary.md`** — define every domain-specific term. Write this last.
8. **`INDEX.md`** — distill repository orientation, load order, and hard-won
   facts that will trip up the next agent.
9. **`OPERATING.md`, the continuation file, and `REVIEW.md`** — customize
   `OPERATING.md` to the project's support-function reality and seed the private
   continuation with the initial completed-state entry. Its path is
   `local/CONTINUE.md` in a new scaffold and root `CONTINUE.md` in a legacy one.

## Per-category prompts

Each generated `context/*.md` ships with prompts at the top. Answer them against
the actual code, then delete the prompts and keep the answers. Do not leave the
prompts in a filled context folder.

## After filling

Set `CTX_FOLDER` and `CTX_CONTINUE_PATH` before running the commands below. For
a new default-folder scaffold:

```bash
export CTX_FOLDER=.ctx
export CTX_CONTINUE_PATH="$CTX_FOLDER/local/CONTINUE.md"
```

For a legacy scaffold, set
`CTX_CONTINUE_PATH="$CTX_FOLDER/CONTINUE.md"` instead.

- Update `INDEX.md`'s staleness banner to “✅ reconciled to `<commit>`”.
- Seed `$CTX_CONTINUE_PATH`: `Last completed` = “Generated context
  from `<commit>`”; `In flight` = None; `Proposed next` = awaiting direction.
- Run `ctx doctor --folder "$CTX_FOLDER"` and record any environment-dependent
  verification caveats in `context/known-issues.md`, where `CTX_FOLDER` is the
  selected folder (`.ctx` by default).
- In **team mode**, review `git status --short "$CTX_FOLDER"` and present the
  durable docs to the repository owner. Stage or commit them only when the human
  explicitly chooses to do so. `$CTX_FOLDER/local/` remains ignored.
- In **local mode**, commit nothing from the scaffold; the whole folder remains
  private through `.git/info/exclude`.

## When the repository moves

Re-read the affected code at the new `HEAD`, update the relevant durable docs,
and stamp `INDEX.md` with the reconciled commit. Treat reconciliation as small,
ordered increments and verify every changed claim against source.

In team mode, these updates are ordinary candidate documentation changes, but
`ctx` still never stages or commits them. In local mode and legacy scaffolds,
they remain private unless and until the user explicitly chooses a supported
future conversion path.
