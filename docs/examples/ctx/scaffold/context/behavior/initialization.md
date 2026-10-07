# Initialization and fresh-clone hydration — ctx

<!-- ctx:doc {"status":"verified","verifiedAt":"57c719617a570a0f66cc560d380d1b480ab51b74 @ 2026-10-07","sources":["pkg/ctx/init.go","pkg/ctx/config.go","pkg/ctx/ctx_test.go","pkg/ctx/mode_test.go","pkg/ctx/behavior_test.go","pkg/ctx/folder_contract_test.go"]} -->

## Scope

This child owns decisions and outcomes for the current options-based init
flow. Cross-command sharing and readiness rules remain in
[the behavior parent](../behavior.md); implementation mechanics belong in
[architecture](../architecture.md). The deprecated Go `Init` wrapper uses the
legacy path, not this new-layout creation flow.

## New scaffold decisions

| Condition | Outcome | Evidence |
|---|---|---|
| No explicit mode or add-on selection | Team scaffold with behavior and glossary | `pkg/ctx/config.go`, `pkg/ctx/ctx_test.go` |
| Explicit local mode | Whole selected folder excluded from ordinary Git tracking | `pkg/ctx/init.go`, `pkg/ctx/mode_test.go` |
| Missing Git repository, invalid new folder name, or unsafe destination | Refuse creation; do not overwrite user content | `pkg/ctx/init.go`, `pkg/ctx/config.go`, `pkg/ctx/folder_contract_test.go` |
| Absent destination still has tracked index entries | Refuse; user must restore or resolve those entries first | `pkg/ctx/init.go` |
| Existing ignore rules hide required durable team files | Refuse team creation; do not claim shareability that Git rules prevent | `pkg/ctx/init.go`, `pkg/ctx/mode_test.go` |

The new folder name is one top-level segment using letters, digits, dots,
underscores, or hyphens; `.` and `..` are invalid. A nested path such as
`docs/ctx` is not accepted for new-layout initialization.

Successful creation publishes generated templates and configuration. Facts
begin draft. Init does not infer domain rules or create a hierarchy of topics.

## Existing scaffold decisions

A fresh clone of committed team context normally lacks
`local/CONTINUE.md`. Running init can recreate that file when:

- the existing scaffold is configured for team mode and the request is team mode;
- required durable outputs are present and effective Git boundaries are valid;
- local continuation is absent and no ambiguous root continuation exists;
- an explicit add-on selection, if supplied, matches the stored selection.

Shared facts, configuration, and selected add-ons remain unchanged. A scaffold
created before behavior became a default therefore stays without behavior
until the user explicitly adds it. Hydration uses the stored project identity.

If local continuation already exists, init refuses to overwrite it. An existing
local-mode or legacy scaffold is not converted by passing team mode.
An incomplete or conflicting scaffold needs explicit repair, not silent
replacement.

Evidence: `hydrateTeamLocalState` in `pkg/ctx/init.go`, sharing checks in
`pkg/ctx/mode_test.go`, and prior-default hydration coverage in
`pkg/ctx/behavior_test.go`.

## Practical outcome for a teammate

The teammate receives shared project knowledge through Git, then records
their own objective and checkpoint in newly hydrated local state.
They do not inherit another developer's live task or get newly introduced
default add-ons implicitly.
