# INDEX.md — context router for {{PROJECT}}

Canonical owner instructions: {{OWNER_INSTRUCTIONS_PATH}}

<!-- ctx:managed begin index-routing -->
## Session load order

1. Read the owner's canonical instructions.
2. Read `OPERATING.md` when that optional, owner-ratified add-on is present.
3. Read `{{CONTINUE_PATH}}` for current local state.
4. Use this index to load the smallest relevant fact set.

## Core routing

| Need | Read |
|---|---|
| First orientation, purpose, scope, or non-goals | `context/overview.md` |
| Components, entrypoints, flows, invariants, or dependencies | `context/architecture.md` |
| Known limitations, operational gotchas, or environment constraints | `context/caveats.md` |

## Optional routing

{{ADDON_ROUTES}}

Prefer parent summaries before specialized documents. Follow links to deeper
owner-authored docs instead of copying their contents here. Treat `draft` facts
as leads to verify, `verified` facts as current only at their recorded commit,
and `not-applicable` documents as intentionally empty. Keep this router near 250
words; split detailed material into a focused child document and link it here.
<!-- ctx:managed end index-routing -->

## Project-owned routing

> Add routes to project-specific or nested context here. `ctx update` and
> `ctx add` preserve this section. Remove this prompt when the first route is
> added; keep detailed facts in the linked document.

| Need | Read |
|---|---|
| [project-specific need] | [`context/<focused-path>.md`] |
