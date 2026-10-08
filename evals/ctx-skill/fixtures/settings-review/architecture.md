# Architecture — settings

<!-- ctx:doc {"status":"verified","verifiedAt":"{{SOURCE_COMMIT}} @ {{VERIFIED_DATE}}","sources":["settings/store.go","settings/store_test.go","docs/storage.md"]} -->

`Open` creates a directory and retains its path. `Set` validates before encoding
`value` as JSON and calling `os.WriteFile` on the key's file. `Get` validates,
reads the file, maps absence to `ErrNotFound`, and decodes the integer. There is
no in-memory value cache or synchronization between writers.
Evidence source: `settings/store.go` — exported operations

Writes use neither a temporary-file rename nor a sync call. The storage note
records that this is not a crash-safety guarantee.
Evidence source: `docs/storage.md` — representation and write boundary

The persistence test launches the test executable as a child, writes `durable`
with value 42 there, waits for successful child exit, then reads 42 in the parent.
That is direct evidence across two processes for one successful write, not
forced interruption, power-loss recovery, or every possible value.
Evidence source: `settings/store_test.go` — process-boundary assertion
