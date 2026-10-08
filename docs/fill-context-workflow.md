# Fill-context workflow

This workflow turns a layout-v2 scaffold into useful, evidence-backed project
context. The templates are prompts, not answers: an agent must inspect the
actual repository and record what it verified, what remains draft, and what does
not apply.

See the [populated ctx example](examples/ctx/README.md) for a complete default
set, a focused behavior child, a sample local checkpoint, and the lifecycle
from draft context to verification, maintenance, and fresh-clone hydration.

For reusable agent-facing guidance, see [the ctx skill and activation guide](agent-skill.md).
Installing a skill is separate from generating or filling a scaffold.

## 1. Confirm the sharing boundary and selected scope

- `ctx init` defaults to **team mode**: durable files are available to review and
  share, while `<folder>/local/CONTINUE.md` stays ignored.
- `ctx init --mode local` keeps the whole selected folder ignored through the
  repository's common `.git/info/exclude`.
- `ctx` never stages or commits files in either mode.
- The v2 core is intentionally small. New scaffolds select behavior and glossary
  by default for domain rules and project-specific language; omit both for
  a core-only scaffold with `--without behavior,glossary`. Select other add-ons
  with repeatable/comma-friendly `--with`, or install one later:

```bash
ctx init --without behavior,glossary
ctx init --with operating,contracts
ctx add --list
ctx add review
```

Keep an add-on only when its concern is real. An empty document adds routing and
maintenance cost without adding context; an installed glossary can instead be
marked `not-applicable` with a reason if inspection shows ordinary language is
sufficient.

## 2. Read authority before context

Start with the repository's canonical owner instructions, such as `AGENTS.md`,
`CLAUDE.md`, `CONTRIBUTING.md`, or an owner-authored skill. Those instructions
govern. If the optional `OPERATING.md` is present, it is a project-owned
supplement and must not duplicate or override higher-priority guidance.
Replace `{{OWNER_INSTRUCTIONS_PATH}}` in README.md and INDEX.md with that
canonical repo-relative path; the placeholder is intentionally owner-supplied.

Then inspect the root README, manifests, build/test configuration, entrypoints,
and top-level source tree. Commit messages and existing documentation are leads;
verify important claims against current source.

## 3. Fill context from parent summary to specialized detail

1. **`context/overview.md`** — users, current purpose, capabilities, boundaries,
   non-goals, maturity, and canonical roadmap pointers. Keep technical flow
   detail out of this parent summary.
2. **`context/behavior.md`** — core business/domain concepts, relationships,
   workflows, decision rules, state transitions, and expected outcomes. Label
   implemented behavior separately from intended changes and cite code, tests,
   or canonical owner-approved requirements. ctx generates only this parent
   template. When deeper explanations are needed, the developer or agent creates
   focused children under `context/behavior/` based on the inspected project,
   gives each child its own metadata, and links it from the parent.
3. **`context/architecture.md`** — components, entrypoints, implementation flows,
   technical invariants, state ownership, lifecycle/concurrency where relevant,
   and active runtime integrations. Link to behavior for domain rules and cite
   source paths.
4. **`context/caveats.md`** — confirmed limitations and gotchas that change how
   an agent should work. Include evidence and a safe workaround; distinguish
   product behavior from environment constraints. Do not create a speculative
   bug backlog or label an observed limitation an accepted tradeoff without an
   explicit owner requirement or decision. Unknown owner decisions are valid.
5. **Other fact add-ons** — fill the default-selected `glossary` plus `contracts`
   and `extending` when installed. Use `not-applicable` with a reason if later
   inspection proves an installed concern does not apply.
6. **`INDEX.md`** — verify that routing names only installed documents and sends
   readers to the smallest relevant set. Keep facts in their owning document,
   not in the router.
7. **`local/CONTINUE.md`** — record only current objective, repository position,
   completed/in-flight work, verification, next action, blockers, and shared-doc
   follow-ups. Durable discoveries belong in shared context or canonical project
   records.

Skip behavior or glossary when deliberately omitted. For an existing v2 scaffold
that predates behavior, upgrade ctx, run `ctx update`, then `ctx add behavior`.
Update and fresh-clone hydration preserve the configured add-on set.

If the `operating` or `review` add-on is installed, the repository owner should
fill its project-owned policy/profile. Do not invent authorization rules, a base
branch, review tools, or required checks.

## 4. Maintain document metadata

Every v2 project-fact document begins with one JSON line:

```html
<!-- ctx:doc {"status":"draft","verifiedAt":"","sources":[]} -->
```

Use it as follows:

- `draft` — incomplete, inferred, or not yet checked at the current source.
  Clear `verifiedAt` but retain known relevant `sources`, including when source
  changes are uncommitted. Revise paths only as their relevance changes.
- `verified` — claims were checked; set `verifiedAt` to
  `<commit-hash> @ YYYY-MM-DD` (use `git rev-parse HEAD`, never a mutable ref)
  and list supporting repo-relative paths in `sources`.
- `not-applicable` — the concern genuinely does not apply; retain a short reason
  in the document so the status is intentional.

Keep the line valid JSON. Record unknowns explicitly. Verification is scoped to
the listed commit and sources; it is not a timeless guarantee.
Include supporting files cited in prose in `sources`. Distinguish implementation
reasoning from direct test assertions and owner requirements. A test that only
asserts an error does not verify unchanged state, and passing tests or statement
coverage does not establish exhaustive behavioral coverage.

Before marking facts verified or relying on a material context claim, review
the claim against its actual evidence. For tests, check the setup, cases,
fields, and timing of assertions; names/comments and one final state check do
not prove every intermediate outcome. Narrow overstated coverage to the actual
asserted case or to implementation reasoning, and reconcile supporting paths
with metadata. Keep supported behavior; report missing evidence separately.
This is a brief review pass, not another default document or permission to
add tests, owner policy, or edits during a read-only task.

For requested evidence review or before declaring authored facts verified,
make that pass inspectable in the review output: record each material claim,
its owning document, implementation/test/owner basis, exact supporting evidence
and limits, and a supported/narrow/unresolved verdict. Split compound claims;
state what was not checked rather than certifying it by omission. Scope this to
the relevant topic, not every sentence during ordinary context consumption.
The [skill's evidence-audit contract](../skills/ctx/references/evidence-review.md)
includes an optional read-only checker for explicitly labelled evidence paths.
It checks citation bookkeeping only, not prose truth or citation completeness.

### Recommended maintainer checkpoint

For newly authored or materially changed claims that readers might treat as
consequential guarantees, recommend a human source-level checkpoint during
normal code/context review. Focus on persistence and recovery, security/access,
business rules and public contracts, broad failure/test-coverage assertions,
and decisions attributed to the owner. This is not approval of every sentence
or a repeat review of unchanged facts whenever another document changes.

The agent prepares the bounded audit, proposes precise wording, runs available
checks, and identifies pending review. The maintainer checks the final wording
and any proposed corrections against the cited implementation, test assertions,
or documented requirements at the source commit. Preserve supported behavior
when test-coverage wording needs narrowing; lack of a direct test does not by
itself establish a bug. Passing checks or model agreement cannot replace this
source-level check.

When this checkpoint is requested or owner-required, leave affected documents
draft until it is complete, with `verifiedAt` cleared and relevant sources
retained. A document-level status cannot certify only its reviewed sentences.
Use the existing review process to record scope and approval, not a new default
context document. `verified` records scoped evidence checking, not human
approval; ctx neither records nor enforces that checkpoint. The skill must not
invent owner policy to impose it. See the
[detailed checkpoint guidance](../skills/ctx/references/evidence-review.md#recommended-maintainer-checkpoint).

## 5. Keep context hierarchical and bounded

Prefer a short parent summary that routes to detail. Link canonical documentation
instead of copying it. Delete template instructions and inapplicable sections.
Useful guidance targets are:

| Document | Target |
|---|---:|
| `INDEX.md` | about 250 words |
| `local/CONTINUE.md` | about 300 words |
| `context/overview.md` | about 500 words |
| Other project-fact documents (`context/**/*.md`) | about 800 words |

These are guidance, not hard limits. For the listed mechanics and project-fact
documents, `ctx status` emits non-failing warnings only around twice those sizes
so legitimate project complexity is not marked unhealthy. When a file grows,
create a coherent child document and link it from its parent. For behavior,
INDEX routes to `context/behavior.md` first, and that parent routes to the
relevant children. Use INDEX's project-owned routing section for additional
direct routes when useful; lifecycle updates preserve that section. Put the same
`ctx:doc` metadata line on each nested project-fact Markdown document so status
tracks it too. Creating these project-specific children is part of context
authoring, not a ctx scaffold-generation command.

## 6. Check structure separately from readiness

Run both checks after filling:

```bash
ctx doctor --folder .ctx
ctx status --folder .ctx
```

`doctor` checks scaffold structure, layout/configuration, managed-marker
grammar, and effective Git visibility/privacy. `status` summarizes metadata and
size guidance. Neither proves that prose is true; source verification does.

For a custom folder, repeat the same `--folder` value. In team mode, inspect
`git status --short <folder>` and let the human decide what to stage. In local
mode, commit nothing from the ignored scaffold.

## 7. Reconcile when the repository moves

For each affected fact document:

1. change its status to `draft`, clear `verifiedAt`, and retain relevant source
   paths while claims are being reconsidered;
2. reread the changed source and relevant tests;
3. update only the owning document and its child links;
4. restore `verified` only after checking claims against an actual new source
   commit and its listed evidence; otherwise leave it draft;
5. update local continuation with any remaining work.

## V1 compatibility

Do not manually reshape an existing schema-v1 or config-less legacy scaffold to
match this workflow. V1 retains its original file set, root continuation where
applicable, unnamed managed markers, and frozen update templates. `ctx update`
selects the compatible source; layout conversion requires an explicit supported
flow rather than file moves by convention.
