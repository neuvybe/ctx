# INDEX.md — context router for ctx

Canonical owner instructions: follow those supplied with the task; no
checked-in agent instruction file was found at the reference revision.

<!-- ctx:managed begin index-routing -->
## Session load order

1. Read the owner's canonical instructions.
2. Read `OPERATING.md` when that optional, owner-ratified add-on is present.
3. Read `local/CONTINUE.md` for current local state.
4. Use this index to load the smallest relevant fact set.

## Core routing

| Need | Read |
|---|---|
| First orientation, purpose, scope, or non-goals | `context/overview.md` |
| Components, entrypoints, implementation flows, technical invariants, or dependencies | `context/architecture.md` |
| Known limitations, operational gotchas, or environment constraints | `context/caveats.md` |

## Optional routing

| Need | Read |
|---|---|
| Business/domain rules, decisions, or state transitions | `context/behavior.md` |
| A project-specific or ambiguous term | `context/glossary.md` |

Prefer parent summaries before specialized documents. Follow links to deeper
owner-authored docs instead of copying their contents here. Treat `draft` facts
as leads to verify, `verified` facts as current only at their recorded commit,
and `not-applicable` documents as intentionally empty. Keep this router near 250
words; split detailed material into a focused child document and link it here.
<!-- ctx:managed end index-routing -->

## Project-owned routing

For fresh-clone initialization questions, start at
[behavior](context/behavior.md) and follow its initialization route. For changes
to lifecycle code, also read [architecture](context/architecture.md).
