# Overview — ctx

<!-- ctx:doc {"status":"verified","verifiedAt":"57c719617a570a0f66cc560d380d1b480ab51b74 @ 2026-10-07","sources":["cmd/ctx/main.go","pkg/ctx/root.go","pkg/ctx/catalog.go","pkg/ctx/version.go"]} -->

## Purpose and users

ctx gives developers and coding agents a small, reviewable context system
inside a Git repository. It reduces repeated discovery by providing places
for durable project facts, task routing, and local continuation state.
It provides structure and lifecycle checks; the developer or agent still
inspects source and writes accurate facts.

## Capabilities and boundaries

| Capability | What it does | Boundary | Evidence |
|---|---|---|---|
| Initialize | Creates embedded templates in team or local visibility mode | Requires a Git repository; does not generate project facts | `pkg/ctx/root.go` |
| Extend | Lists and installs selected add-ons | Installation is explicit in existing scaffolds | `pkg/ctx/root.go`, `pkg/ctx/catalog.go` |
| Maintain | Refreshes managed guidance and checks scaffold health | Does not silently convert layout or sharing mode | `pkg/ctx/root.go` |
| Assess readiness | Checks fact metadata, listed-source freshness, and size guidance | Does not prove factual truth | `pkg/ctx/root.go` |
| Upgrade CLI | Provides the supported binary/package-manager upgrade path | Separate from refreshing a repository's scaffold | `pkg/ctx/root.go` |

## High-level mechanism

Initialize a scaffold, fill its facts from evidence, and use INDEX to load
only what a task needs. Keep discoveries in the owning shared document and
current session state in local continuation. Reconcile affected facts when
supporting source changes; refresh managed guidance after upgrading ctx.

New scaffolds use layout v2 and select behavior and glossary by default.
Other concerns remain optional. See [behavior](behavior.md) for decisions
and [architecture](architecture.md) for implementation.

## Maturity and direction

The reference release is 0.4.0: a pre-1.0 CLI with a frozen layout-v1
compatibility path and a configured layout-v2 path. This context describes
shipped capabilities, not an approved future roadmap. Owner decisions about
future capabilities belong in canonical project records, not inferred here.
