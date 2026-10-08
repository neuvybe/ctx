# Caveats — ctx

<!-- ctx:doc {"status":"verified","verifiedAt":"57c719617a570a0f66cc560d380d1b480ab51b74 @ 2026-10-07","sources":["pkg/ctx/config.go","pkg/ctx/init.go","pkg/ctx/add.go","pkg/ctx/status.go","pkg/ctx/filelock_other.go","pkg/ctx/folder_contract_test.go","pkg/ctx/status_test.go"]} -->

## Active caveats

### Readiness is not factual truth

- **Status:** Current checker boundary.
- **Impact:** A document can have valid, unchanged evidence while its prose is
  incomplete or wrong.
- **Evidence:** `pkg/ctx/status.go` validates metadata, source presence,
  freshness, and size; `pkg/ctx/status_test.go` exercises those checks.
- **Safe response:** Inspect source and tests before marking verified. Do not
  interpret successful status as independent review of every claim.

### New folder names are top-level only

- **Status:** Current new-layout input contract.
- **Impact:** `ctx init --folder docs/ctx` is rejected even though maintenance
  accepts safe existing nested paths for compatibility.
- **Evidence:** `pkg/ctx/config.go`, `pkg/ctx/folder_contract_test.go`.
- **Safe response:** Use a top-level name such as `.ctx` or `.agent` and repeat
  it on later commands. Do not infer support for new nested scaffolds from
  the compatibility validator.

### Init is not a mode-conversion or reset command

- **Status:** Current lifecycle boundary.
- **Impact:** An existing continuation is not overwritten; existing local or
  legacy scaffolds are not silently converted to team mode.
- **Evidence:** `hydrateTeamLocalState` in `pkg/ctx/init.go`.
- **Safe response:** Preserve the scaffold and ask for an explicit reviewed
  migration plan. Do not delete existing context merely to make init succeed.

### Add-ons require explicit adoption and compatible guidance

- **Status:** Current installation contract.
- **Impact:** Updating old context does not install new defaults. Adding an
  already-installed add-on fails; older managed template revisions need update.
- **Evidence:** `pkg/ctx/add.go`, `pkg/ctx/init.go`.
- **Safe response:** Inspect config/catalog, upgrade ctx if needed, run update,
  then add only a missing concern. Fill and verify its new draft document.

## Environment and tooling constraints

On targets using exclusive-file locking, an interrupted lifecycle operation
can leave a lock file behind. The error names the stale path. Inspect it and
confirm no live operation owns it before removing that specific lock.
Do not broadly remove Git metadata or context to clear a lock.
Evidence: `pkg/ctx/filelock_other.go`.
