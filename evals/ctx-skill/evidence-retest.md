# Evidence-precision retest

Run date: 2026-10-08. This is a small forward test of the revised skill and
caveats prompt, not a model ranking or proof of causality. Follow the
[protocol](README.md); do not supply these findings as expected answers.

## Changes under test

- Draft facts clear `verifiedAt` but retain known relevant source paths.
- Observed limitations need an explicit requirement/decision before being
  called accepted tradeoffs; unknown owner decisions are valid.
- Authors and readers distinguish implementation reasoning, direct test
  assertions, and owner-approved requirements. Supporting files cited in prose
  belong in metadata, and passing tests do not certify exhaustive coverage.

The human-facing workflow was aligned with the skill. The caveats template is
project-owned initial content, not a managed update block, so managed template
revision remains `2.0.1`; update does not replace existing authored caveats.

## Setup

Two fresh repositories were prepared with `--with-skill`, with identical
application, tests, requirements, and owner-instruction bytes from the booking
fixture. Neither received an earlier model's context or implementation. Each
model used fresh sessions for population, rescheduling, and read-only handoff.
The explicit requests followed the cross-model protocol, without the evaluator's
known failure cases or desired prose. Tool traces show both loading the skill
and its CLI reference in each phase.

| Runtime | Model | Reasoning setting | Initial demo commit |
|---|---|---|---|
| Pi 1.1.0, Ollama | `glm-5.3:cloud` | Not explicitly selected or independently established | `95ff4714815a503b952088def748398efcd513d9` |
| Pi 1.1.0, Anthropic OAuth | `claude-opus-4-8` | `high` | `cd4fe1409593f31656dc0691440f74e935b9b624` |

The CLI was built from ctx commit
`580bd95f056bf390f8e6a52946381c30bb675b81` plus these uncommitted changes. It
reports `ctx 0.4.0`, but is not the unchanged released binary. The tested skill's
SHA-256 is `d24412d061b9b19a8240118ee6842aef7fd321b824591c4cfafa5f298a7d8558`;
the caveats template's is
`9bbe26192a7b27c3093d29cb8de4cce399d95ebdd5148ad73d5335fc47b7845e`.
Scaffolds used team mode, `.agent`, layout v2, and default behavior/glossary.
Local tools were Node 24.0.1 and Go 1.21.0 on macOS.

## Mechanical and implementation outcomes

Both models completed all three phases. Independent checks established:

- Population changed only context, with empty staging and unchanged HEAD.
- Both implementations satisfy the rescheduling contract: confirmed-only,
  whole-booking destination capacity, failed moves unchanged, and confirmed
  same-session no-op before the capacity check. Their error APIs differ:
  GLM added `ErrNotConfirmed`; Opus reused `ErrInvalid` for cancelled bookings.
- Authored tests, race tests, vet, doctor, and diff checks passed. A separate
  evaluator-only copy passed the same held-out contract suite with 100
  race-enabled repetitions, including move-versus-booking,
  move-versus-cancellation, cancelled same-session, and multi-seat insufficient
  capacity. Those tests were not visible to the agents.
- All five affected facts became draft, with empty verification stamps and
  every population source path retained. Status exited non-zero as expected
  for uncommitted evidence; doctor passed. Both recorded local handoffs.
- Durable team facts remained visible and local continuation ignored. Owner
  instructions, installed skill, and module manifest were unchanged.
- Both read-only handoffs preserved all 29 non-Git entries' contents, paths,
  and modes, plus HEAD, staged diff, and the Git index hash.

Neither model's authored tests directly exercise move-versus-booking,
move-versus-cancellation, cancelled same-session, or insufficient space in a
partly occupied destination for a multi-seat booking. The stronger held-out
suite passing does not retroactively make those cases part of authored coverage.

## Semantic outcomes and remaining misses

Source retention improved over the previous GLM run, which emptied all source
lists on draft conversion. Both runs now explicitly recorded missing owner
decisions for observed implementation limitations. Detailed prose more often
separated test assertions from implementation reasoning, including a fresh
service instance versus a real process restart.

However, evidence precision did not fully pass:

- Opus behavior and its fresh reader described every failed move's unchanged
  booking and availability as directly asserted. The authored test runs a
  sequence of failures, then checks one booking's session and aggregate
  availability. It never inspects the cancelled booking after its rejected
  move or compares complete records after each error.
- GLM's writer said each rejection asserts unchanged booking and availability;
  the reader repeated broad before/after support. The test instead groups three
  errors before one state check. Its cancelled case checks only status,
  session, and destination availability, not the complete invariant.
- GLM's behavior summary says tests assert every owner requirement, despite no
  process-restart test. Opus's overview says all behavior is exercised despite
  recording untested cases elsewhere. Baseline statement coverage was 98.0%;
  the changed suites measured 100.0%, which still is not exhaustive behavioral
  coverage.
- GLM caveats explicitly cites `go.mod` without listing it in metadata. Both
  overviews state the module/import path recorded there without listing that
  supporting file.

Both fresh readers correctly explained core behavior and draft readiness, but
did not identify the overstated failure-state evidence. Opus concluded the only
remaining verification gap was a source commit; that misses the prose issue.

## Interpretation and limits

The revised guidance showed gains in retained evidence and owner authority,
not reliable enforcement of assertion-level accuracy. This demonstrates why
semantic review remains necessary even when metadata and implementation tests
pass. Known source paths being retained does not mean the source lists are
complete or that every claim is verified.

The earlier runs used Pi 0.85.1; these used 1.1.0. The host changed alongside the
skill/template, there was one trial per model, and no new unassisted baseline or
GPT trial. Outcomes cannot be attributed solely to the wording changes.
Activation was explicit, not native discovery. Large repositories, authored
hierarchies, legacy layouts, and ambiguous scopes remain untested here.

Model phase durations were 269.931/260.252/83.257 seconds for GLM and
178.765/255.536/73.813 for Opus (population/change/handoff). These are observed
single-run durations, not a speed benchmark.

Local audit artifacts retain setup hashes, metadata snapshots, separate contract
results, before/after manifests, and sanitized tool/activity logs. Raw native
sessions are not shared because they can contain hidden reasoning. Disposable
demo commits are not upstream revisions to copy into other projects' facts.
