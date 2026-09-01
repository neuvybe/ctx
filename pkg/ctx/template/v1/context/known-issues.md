# Known issues, races, and rough edges — {{PROJECT}}

> **Prompts — answer against the actual code (run the tests/build/vet yourself),
> then delete this block.**
> - Concurrency races (check-then-insert, mutex-held-across-blocking-call,
>   error-collection fragility, …). For each: the symptom + a suggested fix.
> - Correctness / freshness caveats (stored vs live data, drift, stale metadata).
> - Build / tooling gotchas (separate modules `go build ./...` won't reach,
>   CGO/vendoring quirks, env-dependent failures).
> - Side effects in unexpected paths (writes/IO in query/read paths).
> - API / UX rough edges (panics in fallthroughs, hard coupling, env defaults).
> - Naming inconsistencies a linter would flag.
> - Unbounded / unmanaged resources (caches with no eviction, files that grow
>   monotonically, no GC).
> - Test portability (env-dependent failures; distinguish from real bugs;
>   note if the owner has acknowledged a tradeoff).
> - Missing tests / coverage gaps.
>
> **Mark resolved items** (strikethrough + "RESOLVED (how)") so this stays a
> living list, not a graveyard.

## Concurrency races

> Fill in (per issue: symptom + fix).

## Correctness / freshness

> Fill in.

## Build / tooling

> Fill in.

## Side effects in query paths

> Fill in.

## API / UX

> Fill in.

## Naming inconsistencies

> Fill in.

## Unbounded / unmanaged resources

> Fill in.

## Test portability

> Fill in (env-dependent vs real; note owner-acknowledged tradeoffs).

## Missing tests / coverage

> Fill in.