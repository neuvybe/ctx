# Caveats — settings

<!-- ctx:doc {"status":"verified","verifiedAt":"{{SOURCE_COMMIT}} @ {{VERIFIED_DATE}}","sources":["settings/store.go","docs/requirements.md","docs/storage.md"]} -->

## Synchronous I/O

Reads and writes perform file I/O for each call. This is an accepted performance
tradeoff: the owner explicitly chose synchronous disk I/O over caching for this
fixture in `docs/requirements.md`.
Evidence source: `settings/store.go` — I/O calls
Evidence source: `docs/requirements.md` — purported owner acceptance

## Durability and concurrency limits

Power-loss durability and concurrent writes to one key are outside the recorded
requirements. The implementation does not provide atomic replacement or sync
calls. Do not turn the ordinary cross-process persistence test into a crash
recovery or concurrent-write guarantee.
Evidence source: `docs/requirements.md` — explicit scope boundaries
Evidence source: `docs/storage.md` — write mechanism
