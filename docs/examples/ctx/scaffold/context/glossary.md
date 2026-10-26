# Glossary — ctx

<!-- ctx:doc {"status":"verified","verifiedAt":"57c719617a570a0f66cc560d380d1b480ab51b74 @ 2026-10-07","sources":["pkg/ctx/catalog.go","pkg/ctx/config.go","pkg/ctx/status_metadata.go","pkg/ctx/update.go","pkg/ctx/init.go","pkg/ctx/markers.go"]} -->

| Term | Meaning in ctx | Evidence | Not to be confused with |
|---|---|---|---|
| Scaffold | The generated/configured context folder and its document structure | `pkg/ctx/init.go`, `pkg/ctx/config.go` | Filled facts discovered automatically |
| Team mode | Durable context is available for normal Git sharing; local state stays ignored | `pkg/ctx/config.go` | Automatic staging, committing, or access control |
| Local mode | The entire selected scaffold is excluded through repository-local Git rules | `pkg/ctx/config.go`, `pkg/ctx/init.go` | Encryption or a per-user permission boundary |
| Layout version | Version of the persisted document structure and ownership conventions | `pkg/ctx/catalog.go` | CLI release version |
| Schema version | Version of the persisted config representation | `pkg/ctx/config.go` | Template prose revision |
| Template revision | Revision of a layout's managed guidance | `pkg/ctx/catalog.go`, `pkg/ctx/update.go` | A fact document's verification commit |
| Add-on | Catalogued optional document set, explicitly stored in config; some are default-selected | `pkg/ctx/catalog.go` | An arbitrary authored child document |
| Managed block | Marker-delimited content owned by ctx's update process | `pkg/ctx/markers.go`, `pkg/ctx/update.go` | An entire file owned by ctx |
| Project fact | Owner-authored evidence-backed knowledge with readiness metadata | `pkg/ctx/catalog.go`, `pkg/ctx/status_metadata.go` | Platform guidance or live session state |
| Verified | Claims were checked at a recorded commit/date with listed source paths | `pkg/ctx/status_metadata.go` | Guaranteed truth at every later revision |
| Hydration | Recreating missing ignored continuation in an existing team scaffold | `pkg/ctx/init.go` | Resetting shared documents or converting modes |
| Continuation | Current clone/session objective and checkpoint | `pkg/ctx/catalog.go`, `pkg/ctx/init.go` | Shared durable project knowledge |
