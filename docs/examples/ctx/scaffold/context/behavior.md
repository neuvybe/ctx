# Behavior — ctx

<!-- ctx:doc {"status":"verified","verifiedAt":"57c719617a570a0f66cc560d380d1b480ab51b74 @ 2026-10-07","sources":["pkg/ctx/root.go","pkg/ctx/catalog.go","pkg/ctx/config.go","pkg/ctx/init.go","pkg/ctx/update.go","pkg/ctx/add.go","pkg/ctx/status.go","pkg/ctx/mode_test.go","pkg/ctx/catalog_lifecycle_test.go","pkg/ctx/behavior_test.go"]} -->

## Domain concepts and relationships

The **repository owner** supplies authoritative instructions and decides what
to share. A **scaffold** holds durable facts, a router, configuration, and
local continuation. A **layout** defines its structure; selected **add-ons**
extend that structure. The **visibility mode** determines what ordinary Git
tracking can see, not who can read the filesystem.

Facts describe the project. Readiness metadata records whether they were
checked and against which evidence. Managed guidance explains platform use;
it does not own or rewrite those facts. See [glossary](glossary.md) for exact
term distinctions.

## Core workflows and decisions

| Trigger | Decision | Expected outcome | Evidence |
|---|---|---|---|
| New `init` | Default to team mode, behavior, and glossary unless explicitly changed | Durable files are shareable; local continuation is ignored | `pkg/ctx/root.go`, `pkg/ctx/catalog.go`, `pkg/ctx/mode_test.go` |
| `init --mode local` | Keep the whole scaffold out of ordinary Git tracking | A repository-local exclusion covers the folder | `pkg/ctx/init.go`, `pkg/ctx/mode_test.go` |
| Fresh-clone `init` | Hydrate missing local state only for a valid existing team scaffold | Shared documents and selected add-ons remain unchanged | `pkg/ctx/init.go`, `pkg/ctx/behavior_test.go` |
| `add` | Install explicitly requested, supported add-ons | New templates plus updated config and INDEX routing; no existing output overwritten | `pkg/ctx/add.go` |
| `update` | Respect the persisted layout and add-on selection | Managed guidance is refreshed; facts and project-owned routes survive | `pkg/ctx/update.go`, `pkg/ctx/catalog_lifecycle_test.go` |
| `status` | Distinguish readiness from structural health | Draft, invalid, or changed evidence fails readiness; size guidance alone does not | `pkg/ctx/status.go` |

## Rules and exceptions

ctx never stages or commits files. The human owns that decision in both modes.
Ignored files are not secret storage.

Existing scaffolds do not receive newly introduced default add-ons through
update or hydration. Adopting one requires an explicit `add`.
Neither init nor update converts an existing scaffold's mode or layout.
Legacy scaffolds retain their compatibility path.

Every project-fact Markdown child under `context/` has its own metadata.
Verification is scoped to a recorded immutable source commit and listed
paths; a later source change requires review, not automatic prose rewriting.

## Focused behavior routes

| Topic or decision | Read |
|---|---|
| New initialization, refused inputs, and fresh-clone hydration | [Initialization](behavior/initialization.md) |

Shared concepts and cross-command rules stay here. The initialization child
was authored after repository inspection; ctx does not generate topic files.
See [architecture](architecture.md) when tracing implementation.
