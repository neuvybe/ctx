# On-disk format & caching — {{PROJECT}}

> **Prompts — answer against the actual code, then delete this block.**
> - What does the project persist to disk? Where? (file path(s))
> - What's the wire/record format? (a sample record + field-by-field)
> - How is access guarded? (file locks, mutexes, lock ordering)
> - How is serialization made deterministic? (ordering, stamping)
> - What's the hashing / identity model? (hash type, what goes into it, what's
>   portable vs absolute)
> - What's the cache key / freshness model? (what invalidates a cache entry;
>   is it content/position/anything-else derived?)
> - Read path and write path (concise).
> - How to modify the format safely? (forward-compat, version field?, migration)

## File location & locking

> Fill in.

## Wire / record format

> Fill in (sample record + field-by-field).

## Serialization determinism

> Fill in.

## Hashing & identity

> Fill in.

## Cache key / freshness

> Fill in (what invalidates; what guarantees freshness).

## Read path

> Fill in.

## Write path

> Fill in.

## Modifying the format safely

> Fill in (forward-compat rules; version field presence; migration notes).