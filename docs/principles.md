# Principles

`ctx` is useful when it reduces re-derivation without replacing source truth or
the repository owner's authority. Layout v2 follows these principles.

## 1. Share durable evidence; keep current state local

Project facts, routing, and deliberately selected team workflows are durable
context. In team mode they are available for normal review and version control.
The continuation file is different: it records one clone's current objective,
working-tree position, verification, and next action, so it remains ignored.

Local mode provides a stronger visibility choice by excluding the whole context
folder through the repository's common `.git/info/exclude`. Ignored means absent
from ordinary Git tracking; it does not mean encrypted or access-controlled.

`ctx` creates and updates files but never stages or commits them. A human owns
the sharing decision.

## 2. Keep the core small; choose specialized context deliberately

Every project benefits from a concise overview, architecture map, confirmed
caveats, router, and continuation state. Project-specific terms are common
enough that new scaffolds select the glossary add-on by default, but it remains
outside the fixed core and can be omitted when ordinary language is sufficient.
Not every project has a formal extension API, compatibility-sensitive
representations, a shared operating policy, or a review workflow.

Layout v2 keeps those concerns in the add-on catalog. Apart from the
default-selected glossary, add-ons are opt-in. Installed documents should earn
their routing and maintenance cost rather than exist to complete an empty
taxonomy.

## 3. Give each concern one owner and route hierarchically

The index routes; it does not duplicate facts. Overview owns purpose and scope.
Architecture owns components, flows, and invariants. Caveats owns confirmed
limitations and operational gotchas. Optional fact documents own their narrower
contracts, extension points, or terminology. Continuation owns local state only.

Agents should read parent summaries before specialized children and load only
the branch relevant to the task. When detail outgrows a scannable document,
split a coherent child and link it instead of expanding the mandatory load path.

## 4. Make freshness and evidence visible

A polished document can still be wrong. Every v2 project-fact document therefore
states whether it is `draft`, `verified`, or `not-applicable`, the commit/date at
which it was checked, and the source paths that support it.

`verified` is scoped evidence, not permanent authority. When relevant source
moves, return the affected document to `draft`, reconcile it, and record the new
verification point. Unknowns should remain explicit rather than being filled by
plausible inference.

## 5. Owner instructions govern; generated policy is optional

Canonical owner instructions such as `AGENTS.md`, `CLAUDE.md`,
`CONTRIBUTING.md`, or owner-authored skills take precedence. The optional
`operating` add-on begins as a draft supplement and should be ratified, trimmed,
or removed. It must not manufacture approval gates or override an explicit
higher-priority instruction.

The same applies to review: ctx can supply a tool-neutral workflow shape, but the
project owns when review is required, its base, its tools, and its acceptance
criteria.

## 6. Separate structural health from content readiness

`ctx doctor` answers structural questions: are the expected files/configuration
coherent, are managed markers valid, and do effective Git boundaries match the
selected mode?

`ctx status` answers metadata questions: which fact documents remain draft,
which were verified, which are intentionally not applicable, and which have
grown large enough to reconsider their routing. These checks cannot establish
factual truth; an agent must inspect source and tests.

## 7. Preserve compatibility deliberately

Existing v1 and config-less legacy scaffolds keep their original layout,
continuation path, managed-marker grammar, and frozen update source. New design
improvements do not justify silently moving private state, exposing files, or
rewriting a user's established context contract.
