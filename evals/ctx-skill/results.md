# ctx skill forward-test results

Run date: 2026-10-07. Follow-up runs should use the
[evaluation protocol](README.md), not treat these outcomes as expected prose.

## Setup

Five fresh Codex subagents ran with no inherited conversation history and no
model override. They inherited the current session's model/runtime; the exact
model identifier was not captured. This is a small same-runtime forward test,
not a cross-model or statistical reliability study.

Each population trial started from the same
[booking fixture](../../testdata/booking-demo/README.md), materialized as a
separate Git repository by
[the preparation helper](../../tools/prepare-skill-demo.mjs). Application,
requirements, tests, and owner-instruction bytes were independently checked
against the fixture afterward. Skill-assisted repositories additionally had
the complete skill bundle under `.agents/skills/ctx/`.

The CLI was built from the ctx working tree and reported `ctx 0.4.0`.
Scaffolds used layout v2, template revision `2.0.1`, team mode, the custom
folder `.agent`, and default behavior/glossary add-ons. Local tools reported
Go 1.21.0 on macOS and Node 24.0.1.

| Population trial | Initial source commit |
|---|---|
| No task-specific skill | `b8a958ada03924e64d24b8dfcf931060677fe0a3` |
| Explicit skill invocation | `94cc106cc9c00c74ec8f5db42780df0debf034c2` |
| Skill description supplied for selection | `599dd78a2b4d6c9fec36b9c02ffb0cddced1e5ab` |

These are disposable local commits, not upstream revisions to copy into other
facts. The helper creates a new initial commit for each reproduction.

## Outcomes

| Trial | Observed result |
|---|---|
| No task-specific skill | All five default facts populated accurately; doctor and status passed |
| Explicit skill invocation | All five default facts populated accurately; doctor and status passed |
| Skill description supplied for selection | Agent reported loading the skill; all five default facts populated accurately; doctor and status passed |
| Read-only cancellation explanation | Correctly explained immediate seat release, repeat cancellation, retained identifiers, and missing-booking errors; no workspace changes |
| Uncommitted business-rule reconciliation | Described the new two-seat limit, recorded a local handoff, and left all five facts draft pending an immutable source commit; doctor passed and status failed as expected |

For the three population outputs, independent checks established:

- The facts cite actual initial commits and tracked repository-relative
  evidence whose bytes match those commits.
- Source, tests, requirements, and owner instructions were unchanged; HEAD
  and staged contents remained unchanged.
- Nine durable scaffold files were Git-visible, local continuation was
  ignored, and no output appeared outside the requested context folder.
- Prose correctly described immediate confirmation, confirmed-only capacity,
  cancellation idempotence, prohibition of cancelled identifier reuse,
  email normalization, and process-memory state. It distinguished undefined
  payment/access/refund policies from implemented behavior.

The read-only trial used a separate local clone plus the explicit trial's
durable context, without copying its session checkpoint. File contents,
directory entries, modes, HEAD, and the Git index hash matched before/after.
The agent did not hydrate missing continuation as a side effect of explaining
behavior. Its reported evidence route included INDEX, behavior, caveats, and
relevant source/tests rather than every fact document.

The maintenance trial used another clone, copied durable facts, and newly
hydrated local state. The evaluator changed only `MaxSeatsPerBooking` from
four to two without committing it. Afterward, that remained the only tracked
diff and nothing was staged. The agent updated the three documents that named
the numeric limit, cleared verification metadata on all five facts citing the
changed source, and explained the pending commit in continuation. Independent
doctor/status checks, race tests, and vet agreed with the reported result.

## Interpretation and limits

The baseline already succeeded. This run supports the skill's basic usability
and scope/evidence boundaries; it does not demonstrate that the skill improves
quality over an unassisted capable agent.

Skill-assisted agents reported reading the skill and conditional CLI reference.
Those read lists are self-reports, not complete audited tool traces.
Description-based selection was tested by supplying metadata to a subagent;
native Codex discovery and selector visibility were not tested. No other
provider's host integration was exercised.

The fixture is intentionally small. These trials do not establish behavior on
large repositories, authored child hierarchies, legacy layouts, ambiguous
multi-scaffold scopes, or local-mode migrations. Ordinary CI runs deterministic
fixture/helper tests, not model evaluations. A green readiness result still
does not certify prose truth.
