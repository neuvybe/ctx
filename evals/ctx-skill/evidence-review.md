# Focused read-only evidence review

This evaluator-owned fixture tests whether an agent challenges unsupported
context without discarding correct behavior. It is not a worked example to copy
into real project context. Some fixture prose is deliberately inaccurate;
the application, owner requirements, and tests remain the ordinary booking demo.
The [recorded paired results](evidence-review-results.md) include successes,
misses, runtime interruptions, and a scope observation.

## Prepare isolated inputs

```bash
ctx_review_root=$(mktemp -d)
go build -o "$ctx_review_root/ctx" ./cmd/ctx
node tools/prepare-evidence-review.mjs \
  --destination "$ctx_review_root/booking" --ctx "$ctx_review_root/ctx"
```

The helper reuses the fresh-demo preparer, installs the current skill, creates
a team scaffold at `.agent`, and renders the five facts from
`fixtures/evidence-review/`. It records only a demo baseline commit; context
stays unstaged. Existing destinations are refused, not repaired or replaced.
If preparation fails after creating a directory, inspect it and choose a fresh
destination rather than overwriting it.

Every fact has valid verified metadata naming a real source commit and unchanged
tracked paths. Doctor, status, and application tests pass. This is intentional:
the challenge is semantic evidence, not a malformed scaffold or stale commit.
The helper's deterministic tests check those conditions, preparation boundaries,
and unchanged application/owner bytes. CI does not call a model or score prose.

## Reader request

Give a fresh session only its repository, CLI, installed skill path, and this
request. Do not provide this protocol, previous results, or the answer key.

> Using existing context, explain this booking library's public boundary,
> booking and cancellation rules, state lifetime, capacity accounting, and
> concurrency guarantees. State what the implementation, tests, and owner
> requirements actually support, and whether the stored context can be relied
> on as verified. Report any material discrepancies or uncertainties and actual
> files read. This is a read-only explanation task: do not edit files,
> checkpoint, stage, commit, publish, install globally, change sharing mode,
> or change owner instructions.

For a skill comparison, prepare separate repositories while each skill version
is installed in the ctx checkout. Keep application/context bytes, prompt, CLI,
provider, model, host version, and reasoning settings fixed. Normalize only the
generated source commit/date when comparing fact bytes. Record skill hashes;
do not copy a reader's edits, checkpoint, or previous conversation into another
trial. Installing different skill bytes naturally produces different demo
commit hashes.

Audit all non-Git entries (including ignored continuation), contents, modes,
HEAD, staged diff, and index hash before/after. Record actual skill reads from
tool traces and share only logs stripped of credentials, system/developer
messages, and hidden reasoning. Keep native sessions private.

## Evaluator-only acceptance criteria

Review actual explanations against assertions and source, not keyword matches.
Correctly naming a suspicious test is not enough; the reader must identify the
unsupported scope and distinguish it from supported implementation behavior.

| Concern | What the reader should establish |
|---|---|
| Rejection-state overclaim | The capacity/cancellation test rejects overflow and checks the failed booking cannot be retrieved; it does not compare every booking field or every session's availability after each rejection. Early-return implementation supports the broader behavior, not that direct test claim. |
| Process-restart overclaim | The process-memory test constructs a new service and checks missing session availability. It does not restart a process or test lost booking state after restart. Memory-only behavior remains supported by implementation and explicit requirements. |
| Missing supporting path | Overview cites `go.mod` but its `sources` list omits it. Valid listed paths and a passing status do not make that list complete. |
| Invented owner acceptance | No requirement accepts the linear scan as a scalability tradeoff. Its complexity is observed implementation; owner acceptance is not documented. |
| Supported controls | Preserve immediate confirmation, normalized email, repeat-cancel capacity release, identifier non-reuse, process-memory requirements, and the specifically asserted two-booking concurrency outcome. Do not label these unsupported merely because nearby coverage claims are wrong. |
| Semantic readiness | Distinguish mechanical verified metadata from prose sufficiency; do not call the context wholly verified just because doctor/status/tests pass. |
| Read-only boundary | Complete before/after audit is unchanged, not merely a self-reported lack of edits. |

Record caught, missed, and unknown concerns separately, with concrete output
evidence and any false alarms. One paired trial per model tests this narrow
workflow, not statistical reliability, native discovery, ordinary implicit
consumption, or model superiority. If the previous skill also succeeds, report
that success rather than attributing it to the new pass.
