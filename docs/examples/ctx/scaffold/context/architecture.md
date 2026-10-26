# Architecture — ctx

<!-- ctx:doc {"status":"verified","verifiedAt":"57c719617a570a0f66cc560d380d1b480ab51b74 @ 2026-10-07","sources":["cmd/ctx/main.go","pkg/ctx/root.go","pkg/ctx/embed.go","pkg/ctx/catalog.go","pkg/ctx/config.go","pkg/ctx/init.go","pkg/ctx/update.go","pkg/ctx/add.go","pkg/ctx/git.go","pkg/ctx/status.go","pkg/ctx/markers.go","pkg/ctx/filelock_unix.go","pkg/ctx/filelock_windows.go","pkg/ctx/filelock_other.go","go.mod"]} -->

## System map

| Component | Responsibility | Source |
|---|---|---|
| CLI entrypoint and Cobra commands | Parse arguments, invoke operations, render checks, and return errors | `cmd/ctx/main.go`, `pkg/ctx/root.go` |
| Catalog and config | Define versioned document sets, default add-ons, routing, and persisted scaffold identity | `pkg/ctx/catalog.go`, `pkg/ctx/config.go` |
| Embedded templates | Supply frozen legacy, current core, and separate add-on templates without a runtime template download | `pkg/ctx/embed.go` |
| Initialization | Stage creation and hydrate missing team-local state | `pkg/ctx/init.go` |
| Managed maintenance | Parse markers, preflight outputs, publish changes, and preserve project-owned content | `pkg/ctx/update.go`, `pkg/ctx/add.go`, `pkg/ctx/markers.go` |
| Git boundary helpers | Resolve repository/common Git paths and effective ignore/tracking state | `pkg/ctx/git.go` |
| Readiness | Discover nested fact documents and compare recorded source evidence with Git/worktree state | `pkg/ctx/status.go` |

## Entrypoints and external boundaries

The binary exposes init, add, update, doctor, status, upgrade, and version
commands. The lifecycle API lives in `pkg/ctx`.
Lifecycle operations use the local filesystem and invoke Git; Git is needed
for tracking, ignore semantics, worktrees, and source freshness.
The core CLI parser uses Cobra, declared in `go.mod`.

Upgrade is a separate CLI operation. It is not part of context generation,
and running update does not download a new binary.

## Implementation flows

New initialization normalizes options, acquires a repository lifecycle lock,
checks destination/tracking/visibility preconditions, renders selected embedded
documents into a sibling staging tree, and publishes by rename. It verifies
the resulting Git boundary and attempts safe rollback if publication fails
its postcondition. Local-mode exclusion changes participate in rollback.

Hydration loads existing config, checks the durable baseline and local privacy,
then stages and publishes only the missing continuation. It uses the stored
project name rather than the clone directory name.

Update loads the persisted layout/add-ons, validates template compatibility and
managed-marker grammar, and builds replacement plans before writing.
Add also preflights requested outputs, then publishes new documents together
with config and INDEX changes as a rollback-capable transaction.

Status discovers Markdown documents under `context/`, includes required
catalog facts even when missing, parses metadata, and checks evidence against
the recorded commit and current Git/worktree state.

## Technical invariants and state ownership

Config owns schema, layout, template revision, project identity, visibility
mode, and selected add-ons. Those versions are not interchangeable with the
CLI release version.

Named marker boundaries determine managed ownership in layout v2. Bytes
outside those blocks, fact documents, and authored children remain
project-owned. Invalid marker sets fail maintenance preflight.

Lifecycle locks use the common Git directory so linked worktrees coordinate
operations that share ignore state. Platform-specific locking uses flock on
supported Unix systems, a Windows lock implementation, or exclusive-file
locking on other targets. These coordinate ctx operations; they do not stop
unrelated processes from editing files.

See [behavior](behavior.md) for sharing decisions and expected outcomes, and
[caveats](caveats.md) for limitations that affect working safely.
