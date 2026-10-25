# Flat versus folder-first retrieval results

Run date: 2026-10-25. See the [protocol](retrieval-comparison.md) and
[retained public record](retrieval-public-results.json).

Outcome: **no retrieval-speed or accuracy advantage was established**. The
folder-first implementation is supported by lifecycle/compatibility tests, but
the model comparison is mixed and incomplete. Keep the layout for predictable
organization and topic-level ownership; do not advertise a measured agent
performance improvement from this experiment.

## Inputs and runtime

Eight fresh Pi 1.1.0 sessions were planned and launched: GLM 5.3 through Ollama
(`glm-5.3:cloud`, no explicit thinking override) and Claude Opus 4.8 through
Anthropic (`claude-opus-4-8`, high thinking), each on two tasks and both layouts.
Three sessions returned final answers. Four GLM sessions failed to connect
before any tool call; one Claude session reached the requested tool budget.
No evaluator-started retry or correction was supplied. Pi internally emitted
four provider-error messages per failed GLM invocation; provider usage/billing
cannot be inferred from the absence of tool activity.

Model names are configured and Pi-runtime-reported labels, not independently
verified provider response identifiers. The selected Claude thinking setting is
recorded; no effective GLM reasoning setting is inferred.

The evaluator authored fifteen topic payloads about ctx, combining them into
five flat category documents or splitting them beneath five category indexes.
Both variants passed doctor and status before the runs. Topic payloads, copied
source bytes, and the complete installed skill bundle matched across trials.
Semantic payload SHA-256:
`41a6bb995d0d90d54fe9b0f0332b7427b76010c5964582a37f45a29e9e5c07ba`.
The evaluated skill entrypoint SHA-256 is
`99988e6037b1ffe849e60de3564a7ecaf16d13cca2f35dcc2245d21b4c96bed4`.
The public record includes source, reference, helper, CLI, and manifest hashes.

Each trial had a separate repository and fresh session, the same read-only
task wrapper, and explicit installed-skill loading. Tools were restricted to
read, grep, find, and ls, with a path guard excluding parent paths, symlink
escapes, and Git internals. Automatic discovery and saved native sessions were
disabled. Claude ran folder-first before flat for each task; GLM ran the reverse
order. This is not fully randomized or a counterbalanced repetition series.

## Observed Claude results

Bytes below are actual UTF-8 text returned by tools, including repeated output,
not model-input tokens, unique file sizes, or a provider-cost measurement.
Context output includes routing/config/continuation and directory listings,
not only fact prose. Interrupted output is not a comparable completed result.

| Task and layout | Tool starts | Returned tool bytes | Context bytes | Outcome |
|---|---:|---:|---:|---|
| Focused hydration question, flat | 6 | 15,216 | 634 | Final answer; false source-absence claim and no implementation check |
| Focused hydration question, folder-first | 10 | 45,488 | 664 | Final answer; checks implementation, but bypasses category facts |
| Cross-category question, flat | 19 | 69,728 | 10,747 | Final answer; main compatibility/evidence conclusions supported |
| Cross-category question, folder-first | 26 | 69,899 | 6,200 | Budget interruption; 24 tool results, no final answer |

The focused flat response gives the documented hydration contract, but claims
that no source implementation or applicable config exists. Both are present.
Its trace reads the skill/CLI reference and root context mechanics, not the
category facts, source, or config. Lower returned bytes here do not represent
a successful equally grounded answer.

The focused folder-first response reads `pkg/ctx/init.go`, correctly describing
the narrow missing-continuation action, preserved durable files/add-ons,
explicit-selection conflicts, and refusal boundaries. It distinguishes code
inspection from running the CLI. However, it too never reads a category index
or fact topic, so this does not demonstrate successful selective fact retrieval.
Its project-identity fallback description needs the distinction between
configured schema-one team scaffolds and config-less legacy scaffolds, which
hydration refuses.

The completed flat cross-category response reads architecture, behavior, and
caveats, then uses targeted grep against update, add, catalog, doctor, and
status source. Its main conclusions match source: update does not convert
layout or adopt new defaults; contracts remains flat in an existing v2 scaffold;
doctor/status are mechanical checks; category verification does not certify
children. Its generalized statement about indexes should be read as **fact
category indexes**, not the root INDEX router, which has no fact metadata.
It did not inspect every fact in the fixture and is not an exhaustive audit.

The folder-first cross-category response reads an architecture index and
focused topics, plus source. It also spends calls listing category directories.
The budget terminates it before a final answer. Twenty-six calls start because
parallel calls can arrive while best-effort termination is pending; twenty-four
results are recorded. This is an interrupted trial, not a factual pass or an
equal-budget completed retrieval comparison. No six-minute time limit was hit.

## Scope and records

All eight complete before/after workspace manifests, including ignored local
state and installed skills, match. HEAD, staged diff, and index hashes match.
Every recorded model tool path stays in the authorized repository; no blocked
path or tool error is recorded. These observations do not audit all filesystem
activity performed by the host/provider runtime itself.

The public JSON preserves prompts with neutral repository placeholders, exact
final text with that path normalization, configured/runtime-reported model
labels, tool arguments and returned-byte counts, runtime outcomes, input
hashes, and before/after manifest/Git checks. Full local manifests and frozen
source snapshots remain under `/private/tmp/ctx-retrieval-eval.ZjrhJH/`; those
temporary copies are not a durable archive. Hidden reasoning, native sessions,
system/developer messages, and credentials are not retained in the public record.
Missing tool results and absent final answers remain explicit rather than
being silently scored as successful zero-cost runs.

## Interpretation and limitations

The experiment exposes navigation overhead and a source-checking miss, not a
winner. A smaller document can support targeted reads, but splitting also adds
navigation choices. In this run, lower context output in the interrupted
folder trial cannot establish an efficiency gain. Provider/model variation
and unequal completion make wall-time comparisons particularly unsuitable.

Important design limits:

- The subject is ctx itself. The installed skill's CLI reference already
  contains answers overlapping the fact corpus, allowing agents to bypass it.
  This especially weakens the focused retrieval comparison.
- The cross-category question describes a hypothetical v2 scaffold while the
  reader's own context configuration varies by layout. That can introduce
  extra interpretation/config checks rather than pure retrieval work.
- The root router uses category labels; the folder category indexes supply
  richer topic descriptions. This compares navigation packages, not filenames
  or one-file-per-item granularity alone.
- Fifteen authored topics in a partial source snapshot are not a large,
  independently sourced business application. There is one run per condition,
  only one model with completed answers, and no randomized repeat series.
- Guarded read-only tools do not validate authoring quality, unrestricted-tool
  safety, native skill discovery, or implementation performance.

A stronger follow-up would use an unrelated application whose business answers
are absent from the ctx skill, unambiguous identical scenario state, richer
root task routes, and a tool budget sufficient for both completed arms. That
would be a separately bounded experiment, not a retry to obtain a favorable
result. No further model calls were made to replace these outcomes.

## Implementation verification

The repository race suite, Go vet/build, both demo vet/race suites, lifecycle
smoke test, thirteen Node tests, skill validation, and local documentation link
checks pass. Existing v1/v2 template bytes and the pinned v2 worked-example
scaffold are unchanged. New tests cover folder-only defaults, index/child
readiness independence, layout-aware add-ons, old-scaffold update/hydration,
and unsafe/orphan add-on parents.

On this Mac, Go 1.21.0's internally linked test binaries were rejected by the
current loader with `missing LC_UUID`. Local Go checks used
`GOFLAGS=-ldflags=-linkmode=external`; no toolchain upgrade or global setting
change was made. The Linux CI commands remain unchanged apart from including
the new deterministic retrieval-fixture test; model calls are not in CI.
