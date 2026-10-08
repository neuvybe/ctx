# Settings demo

A disposable Go library storing integer settings as individual JSON files.
It is an evidence-review fixture, not a production configuration service.
There is no server, background work, encryption, or transaction across keys.

Run `go test ./...`. Product requirements live in `docs/requirements.md`;
`docs/storage.md` records the representation, not a durability promise.
