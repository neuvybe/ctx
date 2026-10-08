# Glossary — settings

<!-- ctx:doc {"status":"verified","verifiedAt":"{{SOURCE_COMMIT}} @ {{VERIFIED_DATE}}","sources":["settings/store.go","docs/storage.md"]} -->

- Key: a validated setting name, not an encryption key.
- Value: the nonnegative integer accepted by `Set`.
- Store: a directory-backed accessor, not an in-memory database.
- Persistence: a later process can read a successful write; not a crash-safety
  or power-loss durability promise.
Evidence source: `settings/store.go` — accepted keys and values
Evidence source: `docs/storage.md` — representation and durability boundary
