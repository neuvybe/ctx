# Fill-context workflow

How an agent produces accurate `context/*.md` for a target repo. The value of
`.ctx/` is the *analysis*, not the file structure — a blank template is useless
until an agent reads the repo and writes honest, verified docs. This is the
workflow for doing that (derived from a real repo analysis).

## Principles for the agent doing the filling

- **Read the actual code; don't infer from commit messages or READMEs alone.**
  Verify every claim against source before writing it.
- **Reconcile against upstream before relying on a detail.** If the repo has
  moved since a doc was written, the doc is stale — re-verify.
- **Note what you did NOT verify** rather than papering over it.
- **Keep it scannable.** Each `context/*.md` should be loadable in one screen
  for the relevant task.

## Order of operations

1. **Orient.** Read the root `README.md`, any owner agent-instructions (root
   `CLAUDE.md` / `AGENTS.md` / `.claude/skills/`), `Makefile`/build files,
   `go.mod`/`package.json`/etc., and the top-level tree. Note the project's
   purpose, status, and the owner's rules (the owner's rules govern — see
   `principles.md` §3).
2. **`context/overview.md` first** — the thesis, the mechanism in one pass,
   what's *not* here yet. This forces you to understand the whole before parts.
3. **`context/architecture.md`** — end-to-end data flow, package
   responsibilities, concurrency model, integration paths. Read the entrypoints
   (`cmd/`, `main`) and trace one full flow.
4. **`context/format.md`** — any on-disk formats, persistence, caching, hashing
   (identity/freshness). Read the relevant store/serialize code.
5. **`context/extending.md`** — the seams: how to add a language/provider/
   plugin/query/tool/etc. Map the interfaces and the "where to wire a new X"
   per kind. If the owner already has an authoritative skill doc for one of
   these, defer to it and slim this to a pointer.
6. **`context/known-issues.md`** — races, rough edges, build/tooling gotchas,
   test gaps. Run the tests / build / vet yourself; record what actually fails
   and why (distinguish env-dependent failures from real bugs).
7. **`context/glossary.md`** — every domain term. Write it last; it's the
   glossary of what you just learned.
8. **`INDEX.md`** — repo orientation + load order + hard-won facts that will
   trip up the next agent. Distill the above into the "read this first" file.
9. **`OPERATING.md` + `CONTINUE.md` + `REVIEW.md`** — these come from the
   template largely as-is; customize `OPERATING.md` to the project's
   support-functions reality (what automated gates exist vs not), and seed
   `CONTINUE.md` with the first "last completed = context generated" entry.

## Per-category prompts

Each `template/context/*.md` ships with prompts/questions at the top — answer
them against the actual code, then delete the prompts and keep the answers.
Don't ship the prompts in a filled `.ctx/`.

## After filling

- Update `INDEX.md`'s staleness banner to "✅ reconciled to `<commit>`".
- Seed `CONTINUE.md`: `Last completed` = "Generated `.ctx/` context from
  `<commit>`"; `In flight` = None; `Proposed next` = awaiting direction.
- Commit nothing — `.ctx/` is private (`.git/info/exclude`). It lives on your
  machine and (optionally) a backup; it does not enter the project's history.

## When the repo moves

Re-run the relevant category reads against the new `HEAD`; update the changed
docs; stamp the `INDEX.md` banner with the new commit. Treat it as small
ordered increments (cheapest/most-stale first), verifying against source each
time — exactly the reconciliation discipline used in the source repo this was
extracted from.