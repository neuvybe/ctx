# `{{FOLDER}}/` — agent context for {{PROJECT}}

<!-- ctx:managed begin readme-platform -->
This folder keeps durable project facts with explicit readiness separate from
machine-local working state. The active visibility mode is **`{{MODE}}`**. In
team mode, durable files are available for review and sharing while
`{{CONTINUE_PATH}}` stays ignored. In local mode, Git ignores the whole folder.
`ctx` never stages or commits files.

## Contents

- `INDEX.md` routes agents to the smallest relevant context set.
- `context/overview.md`, `architecture.md`, and `caveats.md` hold core project
  facts. New scaffolds select the glossary add-on by default; other add-ons
  contribute focused documents only when selected.
- `{{CONTINUE_PATH}}` holds current clone/session state and is never a source of
  durable project truth.

Read the project owner's canonical instructions first, then any optional
`OPERATING.md`, `{{CONTINUE_PATH}}`, `INDEX.md`, and only the fact documents the
task needs.

## Maintenance

- Keep each fact document's `ctx:doc` metadata honest: `draft`, `verified`, or
  `not-applicable`; verification records a commit/date and source paths.
- Run `ctx doctor --folder {{FOLDER}}` for scaffold and Git-boundary checks.
  Run `ctx status --folder {{FOLDER}}` for content-readiness and size guidance.
- Run `ctx update --folder {{FOLDER}}` after upgrading ctx, then review changes.
- Treat ignored context as private from Git, not as secret storage.
<!-- ctx:managed end readme-platform -->

## Project-owned pointer

Canonical owner instructions: {{OWNER_INSTRUCTIONS_PATH}}
