# Layout v2 and add-ons

Layout v2 minimizes always-loaded context, gives each concern one owner, and
makes fact readiness machine-readable without pretending metadata proves truth.

## Core catalog

Every new v2 scaffold contains:

| Path | Ownership |
|---|---|
| `README.md` | ctx-maintained explanation of the scaffold and lifecycle |
| `INDEX.md` | ctx-maintained hierarchical router plus the owner-instruction pointer |
| `context/overview.md` | Project purpose, users, capabilities, boundaries, and direction |
| `context/architecture.md` | Components, entrypoints, flows, invariants, and runtime integrations |
| `context/caveats.md` | Confirmed limitations, gotchas, and environment constraints |
| `local/CONTINUE.md` | Machine-local objective, repository position, work state, and next action |

Both modes also write `.gitignore` and `config.json`. The config records the
schema, layout, template revision, stable project name, visibility mode, and
installed add-ons. In team mode, `.gitignore` ignores `local/` plus ctx's
short-lived atomic transaction filenames; durable documents remain visible.
Local mode additionally writes a whole-folder rule to the repository's common
`.git/info/exclude`.

The glossary is not part of the fixed core: it remains a catalog add-on so a
project can omit it. New scaffolds select it by default, however, so a normal
`ctx init` also writes `context/glossary.md` and records `glossary` in the
configuration. Create a core-only scaffold with:

```bash
ctx init --without glossary
```

## Add-on catalog

List available add-ons with `ctx add --list`. Select non-default add-ons at
initialization with repeatable/comma-friendly `--with`:

```bash
ctx init --with operating,contracts
```

Add one to an existing v2 scaffold with:

```bash
ctx add [target] <addon...> [--folder .ctx]
```

| ID | Output | Default | Concern |
|---|---|---:|---|
| `glossary` | `context/glossary.md` | Yes | Ambiguous project-specific terminology |
| `operating` | `OPERATING.md` | No | Optional owner-ratified working agreement |
| `contracts` | `context/contracts.md` | No | Representation, interface, and compatibility boundaries |
| `extending` | `context/extending.md` | No | Supported extension points and their end-to-end addition path |
| `review` | `workflows/review.md` | No | Shared tool-neutral review workflow/profile |

`ctx add` updates the named INDEX routing block so it names only installed
add-ons. It does not overwrite an existing add-on output.

## Project-fact metadata

Core project facts and the `contracts`, `extending`, and `glossary` add-ons begin
with exactly one metadata line near the top:

```html
<!-- ctx:doc {"status":"draft","verifiedAt":"","sources":[]} -->
```

- `status` is `draft`, `verified`, or `not-applicable`.
- `verifiedAt` is empty until verification; use
  `<commit-hash> @ YYYY-MM-DD` (`git rev-parse HEAD`, never a mutable ref).
- `sources` is a JSON array of repo-relative source, test, or canonical-document
  paths that support the claims.

Keep the JSON on one line. `not-applicable` documents retain a short human reason.
README, INDEX, CONTINUE, OPERATING, and review are mechanics, routing, state, or
policy rather than project-fact documents, so they do not carry this metadata.

## Named managed blocks

V2 ctx-owned prose uses stable named markers:

```html
<!-- ctx:managed begin readme-platform -->
...
<!-- ctx:managed end readme-platform -->
```

Marker IDs use lowercase kebab case. A marker occupies its own line; IDs are
unique within a file; blocks do not nest; begin/end IDs match. V2 updates match
blocks by ID rather than position and preserve all bytes outside them. Current
IDs are `readme-platform`, `index-routing`, and `review-workflow` when that add-on
is installed.

Malformed, duplicate, mixed, or mismatched markers fail closed instead of risking
user content. V1 retains its separate unnamed ordinal marker behavior.

## Routing and size guidance

The normal session hierarchy is:

1. canonical owner instructions;
2. optional owner-ratified `OPERATING.md`;
3. local `CONTINUE.md`;
4. `INDEX.md`;
5. the smallest relevant branch of project-fact/add-on documents.

Do not copy child detail into INDEX or overview. Add routes for custom nested
documents to INDEX's project-owned routing section, which lifecycle updates
preserve. Give each nested project-fact Markdown document the same `ctx:doc`
metadata line so `ctx status` includes it. Prefer links to canonical
owner-authored material. Guidance targets are 250 words for INDEX, 300 for
CONTINUE, 500 for overview, and 800 for other project-fact documents under
`context/`.

These are not validity limits. `ctx status` gives non-failing size warnings only
at approximately twice the targets: 500, 600, 1,000, and 1,600 words. The warning
means “reconsider routing,” not “the document is invalid.”

## Doctor versus status

`ctx doctor` checks structural integrity and effective Git boundaries: expected
configuration/files, layout and template compatibility, marker grammar,
tracked state, and ignore behavior.

`ctx status` reads project-fact metadata, confirms listed sources existed at the
recorded commit, flags later source changes, and reports non-failing size
warnings. It cannot determine whether a claim is correct, whether listed sources
are sufficient, or whether an owner has ratified policy.

## V1 compatibility

Schema-v1 scaffolds keep their original core-plus-all-documents layout,
`context/format.md`, `context/known-issues.md`, unnamed managed markers, and v1
update source. Config-less legacy scaffolds additionally retain root
`CONTINUE.md` and whole-folder local visibility.

Neither `ctx update` nor `ctx add` is an implicit layout converter. Existing
scaffolds must remain on their compatible lifecycle until an explicit conversion
flow is supported.
