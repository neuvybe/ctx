# Behavior — settings

<!-- ctx:doc {"status":"verified","verifiedAt":"{{SOURCE_COMMIT}} @ {{VERIFIED_DATE}}","sources":["settings/store.go","settings/store_test.go","docs/requirements.md"]} -->

Keys are nonempty ASCII letters/digits/underscores/hyphens. Writes accept
nonnegative integers and replace the selected key. Invalid keys or values fail
before writing. Missing valid keys return `ErrNotFound`. These are implemented
in `Set`/`Get` and required in the canonical requirements.
Evidence source: `settings/store.go` — input validation and storage calls
Evidence source: `docs/requirements.md` — product rules

`TestWriteReplaceAndInvalidInputs` directly compares every stored key/value
immediately after each rejected write, proving all rejected writes leave the
entire store unchanged. It also exercises replacement and a missing-key lookup.
Evidence source: `settings/store_test.go` — invalid-write and replacement test

`TestStoredValueSurvivesWriterProcessExit` directly establishes both persistence
across a process boundary and recovery of atomic writes after forced power loss.
Evidence source: `settings/store_test.go` — persistence test
