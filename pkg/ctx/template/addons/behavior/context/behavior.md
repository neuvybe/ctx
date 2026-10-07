# Behavior — {{PROJECT}}

<!-- ctx:doc {"status":"draft","verifiedAt":"","sources":[]} -->

> **Explain the core business/domain logic from evidence, then delete this
> note.** Keep this parent near 800 words and route deeper explanations to
> focused children under `context/behavior/`. For `verified`, record
> `<commit-hash> @ YYYY-MM-DD` and supporting repo-relative source, test, or
> canonical requirement paths. Label current behavior separately from intended
> changes; do not present a requirement as implemented without checking the code.

## Domain concepts and relationships

> Identify the entities, actors, or concepts needed to understand the rules and
> their relationships. Link to `glossary.md` when installed for detailed term
> definitions.

| Concept | Role and relationships | Evidence |
|---|---|---|
| | | |

## Core workflows and decisions

> Explain what happens for the main inputs or actions. Include preconditions,
> decision rules, expected outcomes, and externally meaningful side effects.
> Link to `architecture.md` for how components implement these workflows.

| Trigger or input | Preconditions and decision | Outcome and side effects | Evidence |
|---|---|---|---|
| | | | |

## Rules, state transitions, and exceptions

> Record domain rules that must hold, valid and rejected state transitions,
> edge cases, and failure outcomes. Cite the tests or requirements that establish
> each rule. Keep component ownership and technical invariants in architecture;
> link to `contracts.md` when installed for interface or data representations.

## Intended behavior and implementation gaps

> When owner-approved requirements differ from current implementation, state
> the requirement, observed behavior, and supporting evidence separately. Link
> to the canonical specification or issue. Leave unknowns explicit and remove
> this section when there is no relevant gap.

## Focused behavior routes

> Split a coherent topic into `context/behavior/<topic>.md` when detail outgrows
> this parent. Give each child its own `ctx:doc` metadata, explain its scope,
> and link only the relevant children here. Keep the shared model and cross-topic
> rules in this parent; INDEX routes behavior tasks here first.

| Topic or decision | Read |
|---|---|
| [focused behavior] | `context/behavior/<topic>.md` |
