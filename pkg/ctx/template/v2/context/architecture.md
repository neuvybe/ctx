# Architecture — {{PROJECT}}

<!-- ctx:doc {"status":"draft","verifiedAt":"","sources":[]} -->

> **Fill from source, then delete this note.** Keep this document near 800 words.
> For `verified`, set `verifiedAt` to `<commit-hash> @ YYYY-MM-DD` and list the
> repo-relative sources used. Delete inapplicable sections and distinguish
> current paths from planned or latent ones.

## System map

| Component or boundary | Responsibility | Primary source |
|---|---|---|
| | | |

## Entrypoints and external boundaries

> Identify user/runtime entrypoints and exchanges with external systems. Omit
> build or delivery tooling unless it materially affects runtime behavior.

## Implementation flows

> Trace the important control/data flows through components as short numbered
> sequences. Cite source paths. When installed, link to `behavior.md` for domain
> decisions and expected outcomes, and `contracts.md` for representation-level
> compatibility details.

## Technical invariants and state ownership

> Record which component owns mutable state and the technical invariants callers
> rely on. Put business/domain rules in `behavior.md` when that add-on is installed.

## Concurrency and lifecycle

> Describe concurrency, ordering, cancellation, cleanup, or process lifecycle
> only where they matter. State explicitly when the project is single-threaded
> or has no special concurrency contract.

## Runtime dependencies and integration paths

> Separate active integrations from optional, generated, or currently unused
> paths. Point to canonical operational documentation rather than duplicating it.
