# Overview — settings

<!-- ctx:doc {"status":"verified","verifiedAt":"{{SOURCE_COMMIT}} @ {{VERIFIED_DATE}}","sources":["README.md","settings/store.go","docs/requirements.md","go.mod"]} -->

A disposable Go library stores nonnegative integer settings by key in a local
directory. Its public boundary is `Open`, `Set`, and `Get`, plus `ErrInvalid`
and `ErrNotFound`. It has no server, background work, encryption, or multi-key
transaction. The module is `example.invalid/settings-demo`.

Each key uses a JSON file containing the `value` field, as specified by the
storage documentation.
Evidence source: `docs/storage.md` — file naming and JSON representation

Read [behavior](behavior.md), [architecture](architecture.md), and
[caveats](caveats.md) for the rules, implementation, and boundaries.
