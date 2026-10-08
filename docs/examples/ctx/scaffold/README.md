# `.ctx/` — agent context for ctx

<!-- ctx:managed begin readme-platform -->
This folder keeps durable project facts with explicit readiness separate from
machine-local working state. The active visibility mode is **`team`**. In
team mode, durable files are available for review and sharing while
`local/CONTINUE.md` stays ignored. In local mode, Git ignores the whole folder.
`ctx` never stages or commits files.

## Contents

- `INDEX.md` routes agents to the smallest relevant context set.
- `context/overview.md`, `architecture.md`, and `caveats.md` hold core project
  facts. New scaffolds select behavior and glossary by default: behavior owns
  business/domain rules and outcomes; glossary clarifies project-specific terms.
  Other add-ons contribute focused documents when selected.
- `local/CONTINUE.md` holds current clone/session state and is never a source of
  durable project truth.

Read the project owner's canonical instructions first, then any optional
`OPERATING.md`, `local/CONTINUE.md`, `INDEX.md`, and only the fact documents the
task needs.

## Maintenance

- Keep each fact document's `ctx:doc` metadata honest: `draft`, `verified`, or
  `not-applicable`; verification records a commit/date and source paths.
- Run `ctx doctor --folder .ctx` for scaffold and Git-boundary checks.
  Run `ctx status --folder .ctx` for content-readiness and size guidance.
- Run `ctx update --folder .ctx` after upgrading ctx, then review changes.
- Treat ignored context as private from Git, not as secret storage.
<!-- ctx:managed end readme-platform -->

## Project-owned pointer

No checked-in canonical agent instruction file was found at the reference
revision. Follow owner instructions supplied with the task; this context does
not create or override them. Repository `README.md` documents usage and
`docs/principles.md` records the platform's design principles.

Facts are verified at the commit/date in each document, not at every later
checkout. Start with [INDEX](INDEX.md) and follow the relevant route.
