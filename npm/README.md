# @neuvybe/ctx

`ctx` gives coding agents durable, project-specific context that survives
compaction. New scaffolds use folder-first layout v3: category `index.md` files for
evidence-tracked facts, focused topic files when useful, and an ignored local
continuation file. The
behavior and glossary add-ons are selected by default; other add-ons stay opt-in.

Team mode is the default. Durable context is available to review and commit,
while `.ctx/local/CONTINUE.md` stays ignored. `--mode local` keeps the entire
scaffold private through the repository's common `.git/info/exclude`. `ctx`
never stages or commits files.

## Install

```bash
npm install -g @neuvybe/ctx
ctx --version
```

The postinstall step fetches the matching binary from the package version's
[GitHub release](https://github.com/neuvybe/ctx/releases). If that download is
unavailable, install with:

```bash
go install github.com/neuvybe/ctx/cmd/ctx@latest
```

## Initialize

```bash
# Team-mode v3 core plus behavior and glossary
ctx init /path/to/repo

# Core only
ctx init /path/to/repo --without behavior,glossary

# Whole-folder private
ctx init /path/to/repo --mode local

# Add focused context at creation
ctx init /path/to/repo --with operating,contracts

# Inspect or extend a configured v2/v3 scaffold
ctx add --list
ctx add /path/to/repo review

# Adopt behavior in an existing v2 scaffold after upgrading ctx
ctx update /path/to/repo
ctx add /path/to/repo behavior
```

`behavior` explains core business/domain rules and expected outcomes;
`glossary` explains project-specific terminology. Both remain add-ons so
projects can omit either with `--without behavior` or `--without glossary`,
but new scaffolds select both by default. `--with` and `--without` are repeatable
and comma-friendly. The other available add-ons are `operating`, `contracts`,
`extending`, and `review`.

ctx generates category indexes such as `context/behavior/index.md`, not preset
topics. During initial population, give independently useful explanations
sibling `<topic>.md` files and descriptive routes. Keep short related definitions
together instead of forcing one file per term. Every index and topic has its
own evidence metadata; index verification does not certify children. Existing scaffolds retain their selected add-ons until explicitly
extended.

For new initialization, `--folder` accepts one top-level name containing only
letters, digits, `.`, `_`, or `-`. Repeat a custom `--folder` with later
commands. On a fresh clone with committed team context, rerun `ctx init` to
hydrate only the ignored continuation file.

## Health and readiness

- `ctx doctor` checks structure, layout compatibility, managed markers, and Git
  boundaries.
- `ctx status` summarizes `draft` / `verified` / `not-applicable` metadata,
  flags listed sources changed since verification, and gives non-failing size
  guidance; it cannot verify truth or source sufficiency.
- `ctx update` refreshes compatible named managed blocks without touching local
  continuation state.

Existing v2 scaffolds keep flat `context/<category>.md` parents and their saved
add-on/update catalog. No command silently converts them to v3. Existing
schema-v1 and config-less legacy scaffolds stay on their frozen v1
layout/update path; ctx does not silently relocate their root `CONTINUE.md`,
change unnamed markers, or expose private context.

See the [project README](https://github.com/neuvybe/ctx#readme),
[folder-first layout guide](https://github.com/neuvybe/ctx/blob/main/docs/layout-v3.md),
[fill workflow](https://github.com/neuvybe/ctx/blob/main/docs/fill-context-workflow.md),
[team-mode migration](https://github.com/neuvybe/ctx/blob/main/docs/migrations/0.2-team-mode.md),
and [layout-v2 migration](https://github.com/neuvybe/ctx/blob/main/docs/migrations/0.3-layout-v2.md).
