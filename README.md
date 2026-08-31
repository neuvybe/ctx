# ctx — durable context for coding agents

`ctx` creates a small, reviewable context system inside a Git repository. Its
default **team mode** keeps durable project facts available to share while
isolating clone/session state:

- core fact documents live under `.ctx/context/`;
- `.ctx/local/CONTINUE.md` records current local state and stays ignored;
- `INDEX.md` routes an agent to only the documents a task needs;
- the glossary add-on is selected by default; other add-ons stay opt-in;
- `ctx` never stages or commits files.

Use `--mode local` when the entire context folder should remain private through
the repository's common `.git/info/exclude` file.

## Layout v2

A default new initialization creates the fixed core plus glossary:

```text
.ctx/
├── .gitignore
├── config.json
├── README.md
├── INDEX.md
├── context/
│   ├── overview.md
│   ├── architecture.md
│   ├── caveats.md
│   └── glossary.md      # default-selected add-on
└── local/
    └── CONTINUE.md
```

The core has one owner per concern: overview owns purpose and scope;
architecture owns components and flows; caveats owns confirmed limitations and
operational gotchas; continuation owns current local state only.
Large projects can add focused nested documents under `context/` and route to
them from INDEX's project-owned section without expanding the always-read core.

The core stays fixed and lean. Glossary remains an add-on, but new scaffolds
select it by default because project-specific language is broadly useful and
the index loads it only when terminology matters. Use `--without glossary` for
a core-only scaffold. The other add-ons remain opt-in.

| Add-on ID | Installed file | Default | Use when |
|---|---|---:|---|
| `glossary` | `context/glossary.md` | Yes | Project-specific terminology needs disambiguation |
| `operating` | `OPERATING.md` | No | The owner wants a shared project-specific working agreement |
| `contracts` | `context/contracts.md` | No | Data, API, storage, or message compatibility is a real boundary |
| `extending` | `context/extending.md` | No | The project has supported extension points |
| `review` | `workflows/review.md` | No | The project wants a shared, tool-neutral review profile |

See [Layout v2 and add-ons](docs/layout-v2.md) for metadata, routing, managed
markers, and size guidance.

## Install

```bash
npm install -g @neuvybe/ctx
ctx --version
```

Or install with Go:

```bash
go install github.com/neuvybe/ctx/cmd/ctx@latest
```

## Initialize and extend

```bash
# Default: team mode with the v2 core and glossary add-on
ctx init /path/to/repo

# Core only: omit the default-selected glossary
ctx init /path/to/repo --without glossary

# Entire scaffold private to the repository
ctx init /path/to/repo --mode local

# Select other add-ons at creation; --with is repeatable and comma-friendly
ctx init /path/to/repo --with operating,contracts

# Inspect the catalog, then add one capability later
ctx add --list
ctx add /path/to/repo review

# One custom top-level folder name
ctx init /path/to/repo --folder .agent
```

For a new initialization, `--folder` accepts one top-level directory name made
of letters, digits, `.`, `_`, or `-`. Nested paths and spaces are not part of
the new-layout CLI contract.

On a fresh clone containing committed team context, run `ctx init` again and
repeat `--folder` when customized. It hydrates the missing ignored continuation
without rewriting durable files.

## Fill and maintain context

Project-fact documents carry one machine-readable metadata line:

```html
<!-- ctx:doc {"status":"draft","verifiedAt":"","sources":[]} -->
```

Set status to `verified` only after checking the recorded source paths, using
`<commit-hash> @ YYYY-MM-DD` (`git rev-parse HEAD`) as the immutable verification
point. Use `not-applicable` deliberately rather than filling an
irrelevant document with boilerplate. `draft` facts are leads to verify, not
authority.

Use the checks for their distinct purposes:

- `ctx doctor [target] [--folder .ctx]` validates scaffold structure, layout
  compatibility,
  managed-marker grammar, and Git visibility/privacy boundaries.
- `ctx status [target] [--folder .ctx]` summarizes document readiness, flags
  listed sources changed since their recorded commit, and emits non-failing size
  guidance. It does not verify factual truth or source sufficiency.

The normal maintenance flow is:

```bash
ctx update /path/to/repo
ctx doctor /path/to/repo
ctx status /path/to/repo
git status --short .ctx
```

In team mode, a human reviews and chooses whether to stage durable changes;
`local/` remains ignored. In local mode, the whole selected folder remains
ignored. See the [fill-context workflow](docs/fill-context-workflow.md).

## Commands

- `ctx init [target] [--folder .ctx] [--mode team|local] [--with <addons>] [--without <addons>]`
- `ctx add [target] <addon...> [--folder .ctx]`
- `ctx add --list`
- `ctx update [target] [--folder .ctx]`
- `ctx doctor [target] [--folder .ctx]`
- `ctx status [target] [--folder .ctx]`
- `ctx upgrade`
- `ctx --version`

## V1 compatibility

Existing schema-v1 and config-less legacy scaffolds are not silently converted
to layout v2. Their file names, unnamed ordinal managed blocks, update source,
and continuation location remain on the v1 compatibility path. Config-less
legacy scaffolds stay whole-folder local with root `CONTINUE.md`.

The deprecated Go `Init(repo, folder)` compatibility wrapper likewise emits the
legacy v1 layout and preserves safe repository-contained custom paths accepted
by ctx 0.1. New Go integrations should use the current options-based API.

See the [0.2 migration guide](docs/migrations/0.2-team-mode.md) for the team-mode
default and legacy visibility behavior. See the
[layout-v2 migration guide](docs/migrations/0.3-layout-v2.md) for the lean core,
add-ons, readiness metadata, and Go API notes.

## Design principles

The short version is: share durable evidence, keep state local, load context
hierarchically, defer to owner instructions, and distinguish structural health
from factual readiness. The full rationale is in [Principles](docs/principles.md).

## Development

```bash
make build
make test
make smoke
```

Release automation ships GitHub binaries and the `@neuvybe/ctx` npm launcher.
The platform supplies structure and lifecycle checks; an agent still has to read
the actual repository and write accurate context.
