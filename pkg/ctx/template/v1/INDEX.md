# INDEX.md — {{PROJECT}} agent context index

> **Where this lives:** `{{FOLDER}}/` holds durable agent context plus
> machine-local continuation state. Active visibility mode: **{{MODE}}**. The owner's
> canonical instruction namespace ({{OWNER_INSTRUCTIONS_PATH}}) governs and
> takes precedence.
>
> **⚠️ Staleness:** `context/*.md` describe the repo *as of the last update*.
> Verify against current source before relying on a detail in a build step.
> Reconcile when the repo moves; stamp the banner below with the commit.
> Reconciliation status: **[not yet filled / ✅ reconciled to `<commit>`]**.
>
> Load order: **`OPERATING.md`** → **`{{CONTINUE_PATH}}`** → this file → the
> `context/*.md` docs your next step needs.

## What {{PROJECT}} is (one paragraph)

> **Fill in:** a one-paragraph plain-language description of what the project is
> and the problem it solves. Force yourself to understand the whole before the
> parts.

## Status

- **Stage:** [early prototype / mature / …]
- **Project owner / maintainers:** {{FOUNDER}}.
- **Key capabilities:** [bullets]
- **What's *not* here yet:** [bullets — mirror the project's TODO surface]

## Layout at a glance

```
> Fill in a short tree of the repo's top-level packages/entrypoints with
> one-line roles. This is the "where is everything" map.
```

## Governing + state files (load before context docs)

| File | Role | When |
|---|---|---|
| `OPERATING.md` | Binding operating mode (constitution) | Every session, first |
| `{{CONTINUE_PATH}}` | Local state + resume protocol (living) | Every session, second |
| `REVIEW.md` | Pre-PR `codex review` pass | Before opening any PR |

## Context docs (read the ones relevant to your task)

| File | When to read |
|---|---|
| `context/overview.md` | Always — thesis, value prop, what problem it solves |
| `context/architecture.md` | Before touching any flow: end-to-end data path, packages, integration paths |
| `context/format.md` | Before changing data formats, storage, or compatibility rules |
| `context/extending.md` | Before adding or integrating a new capability |
| `context/known-issues.md` | Before concurrency work, or to avoid known rough edges |
| `context/glossary.md` | When a domain-specific term is unclear |

## Hard-won facts that will trip you up

> **Fill in:** the gotchas a fresh agent must know — build prerequisites,
> separate modules / tools that `go build ./...` won't reach, CGO/dependency
> quirks, env-dependent tests, committed state files, side effects in query
> paths, auth/env defaults. One bullet each, concrete.

## Working in this repo

> **Fill in:** the conventions the project enforces (custom analyzers/linters,
> constructor patterns, hashing rules, AST constraints, …). Respect these when
> writing new code. If the owner's canonical instructions ({{OWNER_INSTRUCTIONS_PATH}})
> cover this, defer to them and slim this to a pointer.

## TODO surface (from the project's README / roadmap)

> **Fill in:** the open TODO list, so proposed next steps can align with the
> owner's roadmap rather than freelancing.
