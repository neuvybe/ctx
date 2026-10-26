# Claim-audit alignment results

Run date: 2026-10-08. This follows the mixed
[focused reader comparison](evidence-review-results.md), using the
[claim-audit protocol](claim-audit.md). The target is correct, observable
evidence reconciliation, not matching headings or agreement between models.

Outcome: the models agree on source-checked corrected statement wording after
explicit human feedback. Independent confirmation and fully correct audit
alignment were **not achieved**. Review explanations still introduced errors;
agreement on corrected wording must not be presented as reliable autonomous
verification.

## What changed

The skill routes requested evidence review and certification to a bounded audit:
claim, owning document, implementation/direct-test/owner basis, actual evidence
and limits, and verdict. Quantified and lifecycle test claims require a compact
operation/assertion inventory before counts or corrections are derived. Public
boundary reviews compare exported declarations and signatures, not just names
of common calls. Supported behavior is retained when its claimed test coverage
is overstated.

The optional read-only citation helper checks explicit `Evidence source:` paths
against metadata. It neither validates prose nor infers arbitrary citations;
zero recognized citations is not evidence of completeness. No default context
document, metadata field, CLI behavior, or owner operating policy was added.

## Setup and isolation

Readers used fresh sessions and separate repositories, explicitly loading the
installed skill. The authorized CLI and skill were inside each repository;
other trials, evaluator materials, answer keys, and private sessions were
outside it. For independent reviews, no expected finding count or prior reader
answer was supplied. Assisted conditions are described separately below.
The same focused independent request asked about public boundaries, business rules, state
lifetime/storage, failure behavior, and concurrency, with supported claims,
discrepancies, proposed corrections, unchecked scope, and actual files read.

Pi was 1.1.0. GLM used Ollama's GLM 5.3 cloud model, with no explicitly selected
or independently established reasoning setting. Claude used Anthropic OAuth's
Claude Opus 4.8 with high thinking. Actual model identifiers and invocations
are retained in sanitized activity records. No runtime upgrade, global
installation, authentication change, or remote publication was performed.
The working-tree ctx CLI was built from the checkout at
`17f1c60243a2bb0fd042459949771e48d84dc96a`; it reported ctx 0.4.0.

The existing booking application and a newly evaluator-authored settings
application have correct implementation and passing tests, but mixed supported
facts and unsupported evidence descriptions. The settings control is a real
child-process write, normal successful exit, and parent read. It must remain
supported while the forced-power-loss/atomic-recovery extrapolation is rejected.
The settings public inventory also omits its exported `Store` type.

Source bytes, owner requirements, normalized fact contents, and CLI hashes were
compared by fixture across trials. Only generated source commits/dates were
normalized in facts; scaffold project names can differ. Before every reader,
doctor, status, race tests, and diff checks passed. Afterward, complete non-Git
path/content/mode manifests, HEAD, staged diff, and Git index hashes were
compared, including ignored continuation and installed skill files.

## Earlier attempts and revisions

The initial audit reference improved useful coverage but did not align all
readers:

| Reader and fixture | Observation |
|---|---|
| GLM, booking | Found the seeded discrepancies, but incorrectly counted only one rejected booking despite the later cancelled-identifier rejection. |
| Claude, booking | Found rejection-scope, citation, and owner-acceptance gaps; did not explicitly distinguish constructing a service from executing a process restart. |
| GLM, settings | Found the evidence and citation gaps and preserved the real process-boundary control; one source reference had a filename typo. |
| Claude, settings | Found the test and owner-acceptance overclaims, but missed the storage-document metadata gap and incorrectly treated the listed callables/errors as the complete public export set. |

The next reference revision emphasized intervention versus outcome, public
exports, and citation reconciliation before closing a verdict. Its first pass
caught the original discrepancies on both fixtures with both models. Repetition
revealed
why a single clean pass is insufficient: Claude's second booking review caught
the discrepancy but again described the test as containing exactly one
rejection, while separately acknowledging the later identifier-reuse rejection.
That inconsistent count prevented declaring the revision aligned.

The final correction requires the observable operation/assertion inventory,
including later cases. It is not a booking-specific answer or an instruction
to report a predetermined count. Its first booking reviews exposed another
important limit: Claude listed both rejected bookings but still labelled the
first as the only rejection. GLM's inventory accounted for both, but its review
also attributed a grouped set of cancellation properties to direct assertions
more broadly than the test supports. The cancelled status itself is implemented,
not directly compared after cancellation in that test.

Every completed attempt is retained, including these regressions; successful
older sessions do not certify newer skill bytes. The planned two-clean-session
confirmation per model/fixture was not achieved. Subsequent cross-review is a
different, assisted condition, not a replacement score for independent review.

Some earlier outputs also used numbered claim cross-references despite the
full-description request, approximate line anchors, or overbroad wording about
what is unit-testable. A proposed booking API sentence mislabeled its constructor
as a service method. These are additional reasons not to describe the earlier
outputs as flawless.

## Cross-review and reconciliation

Each model received the other model's public audit as explicitly authorized,
untrusted review material alongside raw source, tests, requirements, and context
in a fresh repository. The task checks the peer's evidence bases, inventories,
counts, verdicts, and corrections, then produces a corrected audit. Peer paths
and commit stamps remain visibly those of the earlier run, not this repository.
No evaluator answer key or suspected mistake is supplied.

Cross-review reproduced useful findings but was not a dependable correctness
gate. GLM corrected the booking rejection count and incomplete public exports,
then introduced an incorrect settings finding that the peer's regex transcription
lacked its end anchor (it had the anchor). Its settings inventory also said no
state assertion followed the second rejection, despite the next operation being
the final value read. Both settings cross-reviews accepted the broad per-call
I/O statement, overlooking validation returns before I/O. Absence of a recorded
owner decision was also sometimes worded as absence of any decision.

The evaluator therefore supplied explicit, source-backed candidate statements
for a separate human-guided reconciliation. Readers must still check each
statement against raw evidence and scope their verdicts to those statements.
This condition provides correction input and cannot establish independent
discovery, a skill advantage, or reliable unassisted review. Explanations and
recorded calls were inspected against raw evidence, not accepted by agreement.

Both settings reconciliations now agree with the corrected scopes, including
the final read immediately after the second rejection, validation returns before
I/O, the real normal-exit process boundary, and absence of a *documented* owner
decision. The booking reconciliations agree on exports, rejected cases,
cancellation status evidence, and instance-versus-process boundaries, but their
extra chronology prose needed correction. GLM wrote "after neither
rejection"; Claude wrote that no availability call sits between rejections.
An additional, explicitly guided timing correction asked each to verify the
actual ordering from the complete test body. That additional feedback is retained,
not counted as a clean original reconciliation.

Both timing corrections gave the right operation order. GLM nevertheless added
"adjacent to neither rejection," contradicting its own sequence: the availability
assertion directly precedes the identifier-reuse rejection. One additional
single-statement correction produced correct final replacement wording, but its
explanation still added "No availability call follows either rejection," another
overbroad negative. The final replacement faithfully places availability after
overflow and both cancellations, immediately before identifier reuse, without
claiming a comparison after each rejection. This is agreement on corrected
wording, not a fully passing review or a new independent repeat.

| Source-checked corrected scope | GLM | Claude |
|---|---|---|
| Booking public exports, rejected cases, cancelled-status evidence, instance versus process boundary | Agrees after guided reconciliation | Agrees after guided reconciliation |
| Booking availability chronology in final corrected wording | Agrees after additional explicit correction; explanatory overclaim remains | Agrees after explicit timing correction |
| Settings assertion timing, normal-exit persistence, validation-before-I/O, recorded-decision absence, public type and source gap | Agrees after guided reconciliation | Agrees after guided reconciliation |

All twenty-six attempts are retained: fourteen independent completed reviews,
four completed peer cross-reviews, one startup timeout, four guided
reconciliations, two timing corrections, and one adjacency correction.
The machine audit confirmed byte-equivalent application/requirements, normalized
fact inputs, and CLI hashes by fixture; observed skill/reference reads in every
completed run; in-scope direct tool paths; and unchanged before/after repository
manifests, HEAD, staged diff, and index hashes in every attempt. Recorded shell
commands were separately inspected for scope. Exported records passed the
hidden-content/credential scrub audit; native sessions remain private.

The first Claude booking cross-review timed out at eight minutes without a
completed response or recorded tool calls. Its complete repository audit was
unchanged. This is a retained runtime interruption, not a semantic failure;
its zero reported usage does not establish zero provider usage or billing.
The delay's cause was not established. A fresh-session retry uses unchanged
application, fact, skill, and peer-audit bytes, apart from generated run state.

## Deterministic verification

- Main Go vet and race tests passed, and the CLI built successfully.
- Settings fixture vet and race tests passed; the normal child-process test
  executes as part of that suite.
- All eleven Node tests passed: guarded demo preparation, both review fixtures,
  citation coverage, missing sources, unsafe/self/symlink paths, malformed
  metadata, ignored fenced examples, and read-only checks.
- Skill validation and diff whitespace checks passed.
- Local documentation links resolved; fixture Go formatting was clean.
- CI gains deterministic settings/citation checks, not paid model calls or
  keyword-based semantic scoring.

## Limits

This is an explicitly loaded, focused read-only audit, not ordinary incidental
context consumption, native discovery, population, or maintenance validation.
The unassisted baseline was not rerun. The request, reference, citation
convention, and placement of the CLI differ from older experiments, preventing
a causal claim that skill wording alone produced improvement.

The transfer fixture is newly evaluator-authored, not an independent real-world
held-out project. Repeated reviews are a bounded check, not statistical
reliability or a guarantee on larger repositories. Human-guided reconciliation
is not a passing substitute for failed independent review. Mechanical
read-only checks and citation coverage do not establish semantic correctness.
Some outputs used numbered references despite the full-description request,
retried unsupported CLI flags, or skipped parts of the skill workflow. This is
not a full instruction-compliance pass. The narrow Claude timing correction did
not read owner instructions; its source-accurate timing verdict does not erase
that procedural omission.
Sanitized public records omit hidden reasoning, system/developer messages, and
credentials; native sessions remain private.

The practical result is a more inspectable review process and a deterministic
bookkeeping check, not an automatic truth gate. Critical `verified` claims still
need source-level review of the wording and its corrections. Further independent
reliability work should be a separately bounded experiment, not repeated calls
until a favorable explanation appears.

## Follow-up guidance, not another evaluation

After these attempts, the skill and fill-context guide added a recommended
maintainer checkpoint for consequential new or changed guarantees. They
distinguish evidence verification from human approval and leave owner-required
pending review draft. Those instruction changes were not part of the model
trials above; no additional model run or reliability result is claimed.
The later [maintainer-checkpoint comparison](maintainer-checkpoint-results.md)
tests that follow-up guidance separately with old/new bundles and three models;
it does not retroactively change these claim-audit results.
