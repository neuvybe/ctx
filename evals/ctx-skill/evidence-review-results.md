# Focused evidence-review results

Run date: 2026-10-08. The [targeted protocol](evidence-review.md) tests a reader
against correct implementation and passing tests but partly unsupported context.
This paired experiment produced mixed results, not reliable semantic validation.

## Setup and independence

Four separate booking repositories contained identical application, requirements,
owner instructions, and context facts. Input audits compared source bytes to the
canonical fixture and normalized only the generated source commit in fact
metadata. All five facts had valid verified metadata at their repository's
actual HEAD; doctor, status, and race tests passed before each reader.

The comparison held Pi 1.1.0, provider/model, CLI, request, and requested reasoning
settings fixed. GLM used Ollama `glm-5.3:cloud`, with no explicitly selected or
independently established reasoning setting. Opus used Anthropic OAuth
`claude-opus-4-8`, with `--thinking high`. The same ctx working-tree CLI reported
`ctx 0.4.0` with layout v2 and managed template revision `2.0.1`. It was built
from commit `580bd95f056bf390f8e6a52946381c30bb675b81` plus the caveats changes,
not the unchanged released binary.

The previous skill included the earlier evidence warnings; it was not an
unassisted baseline. Its SHA-256 was
`d24412d061b9b19a8240118ee6842aef7fd321b824591c4cfafa5f298a7d8558`.
The skill with the explicit review pass had SHA-256
`e8b7415c5fce66c5eb0765361c571ddcca26352afa614b423f03a1a06c5c5dee`.
Only those skill bytes differed materially between conditions. Both included
the same CLI reference and optional UI metadata.

Each reader had a fresh session, an explicit skill path, and the same realistic
explanation/evidence request from the protocol. The evaluator's answer key,
previous findings, and other trials' artifacts were outside its authorized
scope. Traces show every completed reader loading the skill; this is not a
native discovery or automatic-selection test.

| Trial | Initial demo commit |
|---|---|
| GLM, previous skill | `fd8e68c47afb4389dc8763595c1e8da265045147` |
| GLM, review pass | `2e208abed049de4c0cf82632f7b4d5a9242c91ba` |
| Opus, previous skill | `11c9f6c4d16669156c54791ce1af77e5301f86f9` |
| Opus, review pass | `f62b70c3ae07bfb9bdbad44464c17aec4663956e` |

## Observed outcomes

The evaluator manually checked explanations against actual assertions, not
phrase matching. "Partial" means the reader identified only part of the
unsupported scope, not that the statement was safely verified.

| Concern | GLM, previous skill | GLM, review pass | Opus, previous skill | Opus, review pass |
|---|---|---|---|---|
| Rejection-state test overclaim | Identified | Identified, with more precise assertion timing | Identified, but miscounted rejections | Identified |
| Process-restart test overclaim | Partial | Partial; explicitly accepted new-service construction as a restart simulation | Partial | Missed |
| Omitted `go.mod` supporting path | Missed | Identified | Missed | Missed |
| Invented acceptance of linear scanning | Identified | Identified | Identified | Identified |
| Mechanical readiness versus prose accuracy | Distinguished | Distinguished | Distinguished | Distinguished |
| Supported core behavior retained | Yes | Yes, but added an unrelated API error | Yes | Yes |
| Repository read-only audit | Unchanged | Unchanged | Unchanged | Unchanged |

Both previous-skill readers already caught important unsupported claims, so
their success cannot be attributed to the new pass. The new GLM reader found
the missing source path and explained that availability was checked only after
cancellation, not immediately after rejection. Those are useful observed gains.
The new Opus reader missed a gap its previous-skill reader partially identified.
No condition caught all four concerns completely.

## Remaining errors and scope observations

- Both previous-skill readers identified that the fresh-service assertion checks
  only session availability, not booking loss. They did not clearly separate
  constructing a service from restarting an actual process.
- GLM with the new pass proposed narrower fresh-service wording, but also
  explicitly said constructing a service was an acceptable restart simulation
  because the requirements equated their expected outcomes. Equal intended
  outcomes do not make those test interventions equivalent.
- Opus with the new pass reported only the owner-acceptance and rejection-test
  discrepancies, then said everything else matched. It did not challenge the
  restart coverage claim or missing metadata source.
- The new GLM explanation incorrectly said all public methods return `Booking`
  values. Only some do; `CreateSession` and `Cancel` return errors, while
  `Available` returns an integer and error. The previous Opus explanation also
  said the capacity/cancellation test contained a single rejection, overlooking
  its later cancelled-identifier reuse rejection.
- GLM with the new pass listed the evaluation's parent directory while inspecting
  the authorized CLI. That inventory was outside the authorized repository.
  No reads of other trial files' contents were observed in the recorded calls,
  but this is a scope miss, not evidence that every trial stayed fully in scope.

All four completed readers preserved the 29 non-Git entries' paths, contents,
and modes, plus HEAD, staged diff, and index hash, including ignored continuation.
Unchanged repository state does not prove that no out-of-scope reads occurred.

## Runtime interruptions and limits

The first two concurrent Opus attempts timed out after approximately five
minutes without completed assistant output or tool calls. Their repository
audits were unchanged. Fresh-session retries against the unchanged inputs ran
serially and completed. The cause of the initial delay was not established;
those attempts are retained as runtime interruptions, not model semantic
failures. Zero recorded usage from an interrupted stream is not proof of zero
provider usage or billing.

Completed reader durations were 162.890 seconds for GLM with the previous skill,
208.256 for GLM with the new pass, 122.048 for the successful previous-skill Opus
retry, and 151.727 for the successful new-pass Opus retry. These are single-run
observations, not a speed benchmark.

There was one completed trial per condition per model, not a reliability study.
The prompt explicitly requested evidence reconciliation, which is stronger than
ordinary incidental context consumption. Scheduling and stochastic outputs can
explain differences; the data does not establish a causal skill advantage.
The fixture is small and uses selected known mistakes, not an independent
held-out project. Large repositories and other models remain untested here.

The new pass is a useful review procedure, not an automatic correctness gate.
Semantic review still needs assertion-level scrutiny, and reviewer prose itself
can introduce errors. Local audit artifacts preserve equivalent-input hashes,
before/after manifests, completed explanations, and shareable activity logs.
Native sessions remain private; exported logs omit system/developer messages,
hidden reasoning, and credentials.
