# Four-stage context evaluation scorecard

Evaluate four different questions independently. There is no averaged score:
successful execution does not establish correctness, valid metadata does not
certify wording, and a reader's correct answer does not repair false context.
These are maintainer tools, not new ctx commands or a scaffold migration.

| Stage | Passing evidence | What does not count |
|---|---|---|
| Successful execution | Explicit model qualification, completed frozen attempt within budgets, observed prerequisite reads, context-only writes, matching source and registry audits | A final message after timeout; prepare/replay as model execution; access to a document as comprehension |
| Structurally valid context | Folder-first scaffold, metadata, available evidence bookkeeping, resolvable relative routes, correct audited fact count, recorded passing doctor | Green status as semantic approval; unavailable filtered files as repository absence |
| Supported wording | Attributed independent review of exact owning claims, source excerpts, evidence type, limits and exceptions, bound to this candidate and frozen rubric | Self-certification, correct child detail excusing an overbroad parent, one assertion proving an entire invariant, implementation implying owner acceptance |
| Useful downstream results | Matched repeated fresh-reader tasks, grounded outcome and citation reviews, preserved scope, and an observed correctness or declared efficiency benefit | One fixed-order run, duplicate artifacts as repeats, fewer calls with wrong answers or misleading citations, a source-only accuracy ceiling as proof of benefit |

Verdicts are `passed`, `failed`, `incomplete`, and `not-evaluated`. The downstream
stage can also be `not-demonstrated`: all observations are valid and correct,
but the declared benefit was not observed. Failed checks and missing evidence
remain separate in every stage.

Draft facts can be structurally valid. Readiness separately reports verified,
draft, and not-applicable counts. Required-claim coverage, unsupported certified
claims, unresolved claims, and unreviewed fact documents remain visible; an empty
or all-draft answer cannot win simply by making fewer assertions. A not-applicable
label needs a grounded applicability review, not just a change to metadata.

## Calibrate before running models

```bash
node --test tools/context-scorecard.test.mjs \
  evals/ctx-skill/fixtures/scorecard/client-reopen.test.mjs
node tools/calibrate-context-scorecard.mjs
```

Optionally retain the calibration in a **new**, private file:

```bash
node tools/calibrate-context-scorecard.mjs \
  --output /absolute/private/new-calibration-report.json
```

The public [synthetic fixture](fixtures/scorecard/entry-store.mjs) and
[named controls](fixtures/scorecard/cases.mjs) are deliberately small. Fifty
controls cover supported wording, overbroad parent rules, a no-op exception,
classifier priority, overstated restart and assertion evidence, invented owner
acceptance, filtered-view absence, incorrect glossary mappings, broken routes,
citation omissions, incorrect fact counts, stale/missing reviews, draft readiness,
runtime qualification and timeout failures, and misleading reader comparisons.
They also distinguish genuinely untriggered conditional audits from missing,
stale, or incomplete applicability inventories, require parent counterexample
inventories, and reject correct behavior paired with misleading citations.
Boundary tests additionally cover malformed metadata, unsafe proposals, source
integrity, grounded reader grading, and private nonoverwriting report output.

The fixture reference oracle recognizes only ten enumerated synthetic claims.
Known incorrect statements are rejected; unfamiliar wording produces abstention.
This tests the scorecard wiring and finite reference, **not a general semantic
grader**. The positive control's reader observations are scripted, not measurements
of real model benefit. Its reopened-client test really executes, but starts no
replacement process. No provider is called and no private project material is
included. Ordinary CI runs only these model-free checks.

## Grade an existing frozen authoring attempt

```bash
node tools/evaluate-context.mjs \
  --trial /absolute/private/frozen-authoring-attempt \
  --rubric /absolute/private/frozen-rubric.json \
  --review /absolute/private/wording-review.json \
  --comparison /absolute/private/reader-comparison.json \
  --output /absolute/private/new-scorecard.json
```

Only `--trial` is required. Missing review or comparison data remains
`not-evaluated`; it is never filled in from metadata or a successful final answer.
The evaluator reads `input.json`, `manifest.json`, `result.json`, and optional
`attempt.json`. It overlays authorized frozen writes in memory; it does not trust
the materialized candidate directory as a substitute for the frozen attempt.
It never resumes an agent, executes repository code, reruns doctor, publishes,
or follows source paths onto the host filesystem. Doctor/status are recorded
coordinator observations, not fresh CLI execution or independent wire attestation.

All file arguments must be absolute coordinator-selected paths. JSON inputs must
be bounded regular files, not final-component symlinks; reports are exclusively
created with private permissions and never overwritten. Keep destinations under
coordinator control. Parent-directory replacement by a hostile same-user process
is outside this boundary, as with the authoring runner. Private reviews and
reports may contain private prose and must stay private.

Exit code zero means **a report was generated**, not that its stages passed.
Add `--require-reviewed-context` to exit unsuccessfully unless execution,
structure, and wording pass and no draft facts remain. Exit code one means that
gate was not met; exit code two means invalid arguments/artifacts or an output
failure. Private parser/error payloads are not printed. Downstream usefulness
is a separate verdict, not part of this reviewed-context gate.

`promotion.eligibleForReviewedUse` is an evaluation-readiness signal, **not merge
permission or human approval**. A downstream pass never waives an earlier
failure. Explicitly untrusted draft or incorrect context can still be studied
in an isolated experiment without being promoted.

The structural implementation currently targets the runner's `.ctx` folder-first
layout. Older layouts produce an incomplete evaluation rather than a migration
or a claim that existing ctx scaffolds are broken. Metadata checks cover fact
documents; route checks also cover the main router. Citation detection is bounded
to explicit `Evidence source:` lines and inline-code tokens equal to available
source paths, outside fenced examples. Zero recognized explicit lines is not
proof of complete citations. Markdown anchors, external URLs, and arbitrary prose
citations are not exhaustively checked.
The structural stage also reconciles every evidence path from a validly bound,
grounded wording or not-applicable review against the owning fact's metadata.
This catches unnamed test references discovered by a reviewer, including in
drafts and parent summaries. Exact files and real containing directories count;
filename prefixes and a child's sources do not. Stale or ungrounded reviews
cannot manufacture these findings. `reviewedEvidence` reports the checked
document/source pairs and gaps; an empty list without a complete review is not
source-completeness certification. Supported meaning and missing metadata remain
separate verdicts. Scorecard version 1.2.0 adds this check; preserve older reports
and write any reanalysis as a new artifact rather than relabelling old results.
Route detection covers relative Markdown links and path-shaped inline-code tokens,
not arbitrary unformatted prose or a complete Markdown parser.
Unchanged draft seed blockquotes (including conditional add-on guidance) are not
active routes; unfilled topic placeholders produce warnings in drafts and the
router. Newly authored blockquotes are checked, and verified facts cannot use
that draft-template exemption.

## Freeze an evidence rubric before authoring

Have the coordinator identify material behaviors and held-out questions from
source, tests, requirements, and decisions. Freeze this before seeing model
outputs and apply the same scope to every model. Keep the answer key and reviews
out of author/reader capabilities. Do not silently drop a difficult claim or
lower the benefit threshold after seeing results.

The versioned rubric has this shape:

```json
{
  "schemaVersion": 1,
  "title": "Entry lifecycle and evidence boundaries",
  "requiredClaims": [
    {
      "title": "Parent mutation preconditions",
      "document": ".ctx/context/behavior/index.md"
    }
  ]
}
```

Document ownership is optional when routing is intentionally left to the author;
when specified it is enforced. Review each material occurrence independently,
including parent summaries, glossary definitions, and local handoffs when they
make material assertions. Add discoveries to the documented review scope; do not
let a correct child claim stand in for an incorrect parent. The required list is
a minimum, not permission to leave other material claims unexamined.

Scorecard 1.3.0 adds opt-in `parentSummaryAudit: true` to the frozen rubric.
For this mode, `review.parentSummaries` inventories every fact `index.md`, main
router and local handoff returned by `parentSummaryDocuments(trial)`. Each entry
has `path`, `hash`, `rationale`, and `claimTitles` linking the real broad summaries
in that document to grounded review claims. An empty list needs a reason why no
material summary is present; it is not a fabricated claim. A missing, duplicate,
stale or unknown inventory cannot pass.

Each linked claim supplies `counterexampleCheck: {coverage, cases}`. Cases need
unique descriptive `title`, `rationale`, source-grounded `evidence` and `relation`:
`supports`, `outside-qualified-scope`, `counterexample`, or `unresolved`. Unresolved
cases also need `missingEvidence`. Inventory actual operations and relevant
actor/identity/state/history variants, not just wording containing “all.” Generic
subjects and arrows can imply the same scope. Include a real in-scope case;
excluding every case is not a check. A recorded counterexample must have a
`narrow` claim verdict, while any unresolved case blocks a supported result.
Check the corrected parent itself, not only the detailed child.

Case evidence joins owning-document metadata reconciliation. Reported case and
summary inventories remain reviewer attestations: the machine binds and checks
their consistency, not whether every summary or counterexample was discovered.
Old rubrics without the flag retain their previous review contract; no scaffold
metadata or ordinary CLI requirements change. Keep historical reports unchanged.
Version 1.3.1 also rejects supplied case checks omitted from their owning parent
inventory. An explicit counterexample cannot be hidden by unlinking its title;
this is a consistency check, separate from discovering an unlisted summary.

Use optional `conditionalClaims: [{title, trigger}]` for audits that apply only
when an author makes a particular kind of assertion: owner acceptance, broad
test coverage, lifecycle-test interventions, or unavailable-file contents. These
are not mandatory positive topics. Do not force boilerplate or invent an authored
quote merely to satisfy a conditional rule.

For every conditional rule, the review supplies `applicability` with its `title`,
`verdict: triggered|not-triggered|unresolved`, a `rationale`, `documents`, and
`claimTitles`. Use `reviewScopeDocuments(trial)` to bind the complete fact/router/
local-handoff Markdown scope by path and content hash. A triggered rule links
nonempty `claimTitles` to actual grounded entries in `claims`; review every
material occurrence independently. A not-triggered rule has an empty occurrence
list and no fabricated quote. Unresolved applicability needs `missingEvidence`
and remains incomplete. Missing, duplicate, unknown, or stale applicability
inventories cannot silently pass. Whether a rule is truly triggered, and whether
all occurrences were found, remain explicit reviewer judgments, not semantic
proof by the evaluator.

## Ground wording assessments

An independent reviewer receives frozen source and candidate text, not authority
to rewrite the unassisted attempt. A human should adjudicate ambiguous evidence
and acceptance decisions; an independent model's assessment is still fallible.
The evaluator validates evidence bookkeeping, **not the semantic verdict itself**.

`review` contains:

- `schemaVersion: 1`, `sourceCommit`, `candidateHash`, and `rubricHash`.
  Use `candidateFingerprint(trial)` and `fingerprint(rubric)` from the scorecard
  library; object-key order is normalized, array order is not.
- `reviewer: {kind, name}` and a readable `scope`. Real reviewers use `human`
  or `independent-model`. `fixture-reference` is restricted to explicitly
  synthetic manifests and must never be added to real trials to bypass review.
- `claims`, with a full descriptive `title`, exact `location: {path, line, quote}`,
  `basis`, `verdict`, `limits`, and source `evidence: [{path, line, quote}]`.
  Lines are one-based and quotes must occur at the stated lines in frozen files.
  Source evidence must be in the source manifest, not another context document.
- `basis` distinguishes `implementation`, `direct-test`, `owner-requirement`, and
  `owner-decision`. A passing test does not prove every possible behavior; an
  implementation limitation is not an accepted owner tradeoff.
- `verdict` is `supported`, `narrow`, or `unresolved`. Narrow assessments require
  `correction`; unresolved assessments require `missingEvidence`. Supported and
  narrow assessments require actual source excerpts and an explicit limit.
- `documents: [{path, hash, materialClaimsInventoried: true}]`, covering every fact
  document. The hash is `digest(documentText)` from the boundary library. A
  not-applicable document also needs
  `notApplicable: {rationale, evidence: [{path, line, quote}]}`.
- Optional `reportedFactDocuments: {value, quote}` records a reviewer-recognized
  count from the final answer. The quote must match the final text; router and
  continuation files are not fact documents. Arbitrary natural-language counts
  are not automatically extracted.

Candidate/rubric fingerprints reject stale reviews. Document inventories are the
reviewer's explicit attestation of complete material-claim coverage, not machine
proof that unlisted sentences are true. The synthetic
[review builder](fixtures/scorecard/calibration.mjs) demonstrates the data shape,
but must not be used as a semantic reviewer for another repository.

## Compare useful downstream work

Freeze a versioned comparison `protocol` with the source commit, exact reader
model/provider/runtime/version/thinking, a budget fingerprint, task titles and
required outcomes, at least three repetitions per task, selected efficiency
metric, and minimum improvement fraction. Counterbalance baseline-first and
context-first ordering for each task. Fresh sessions get the same source,
instructions, and budgets; only the context condition differs. Retain all
failures, timeouts, and denied operations. Do not silently retry or substitute
models. Three repetitions is a minimum diagnostic, not a power calculation.

New protocols use `schemaVersion: 2` and `citationPolicy: "all-material-claims"`.
Citation accuracy is required independently of the task's behavior outcomes:
correct behavior with an inaccurate source attribution is not a fully correct
answer. Freeze this criterion and request accurate source citations in the
reader prompt before launching. Do not retrofit it onto old sessions or treat
assisted answer corrections as fresh model outputs.

Use realistic source-location/exception questions, bug diagnosis, and eventually
small isolated change tasks. Include cases for which source-only work is costly,
not only easy questions that already score perfectly. A change task requires
separately authorized isolated execution, proportional tests, and patch/scope
review. This scorecard can record such observations but does not launch or
sandbox change agents or execute their tests.

`comparison` has `schemaVersion: 1`, attributed `reviewer`, readable `scope`,
`protocol`, and `pairs`. Each pair names a task, one-based repetition, execution
order (`baseline-first` or `context-first`), and baseline/context observations.
The [scripted comparison builder](fixtures/scorecard/calibration.mjs) is an
executable shape example. Each observation contains:

- `protocolHash: fingerprint(protocol)`, unique `recordLabel`, `final` answer,
  baseline `candidateHash: null` or the context candidate fingerprint,
  `completed`, `scopePreserved`, `writeAttempts`, and numeric `metrics`.
- `artifactHash`: `fingerprint(record)`, where `record` is that entire observation
  excluding `artifactHash`, `outcomes`, and (for the citation-aware protocol)
  `citationReview`. Freeze from actual reader artifacts;
  do not reconstruct favorable records from a model's self-report.
- Grounded `outcomes`, exactly matching the task's required outcome titles, with
  `verdict: supported|unsupported|unresolved`, exact `answerQuote`, `rationale`,
  and frozen source `evidence: [{path, line, quote}]`. Unresolved assessments need
  `missingEvidence`; otherwise evidence must be nonempty. Outcome correctness
  remains the attributed reviewer's judgment, not keyword matching.

For the citation-aware protocol, each observation also needs this independently
graded `citationReview` (the shown quote/evidence are shape placeholders):

```json
{
  "answerHash": "sha256 of the exact final answer, using digest(final)",
  "materialClaimsInventoried": true,
  "scope": "All material factual claims and their source attributions in this answer",
  "claims": [
    {
      "title": "Publication retry source attribution",
      "answerQuote": "Exact answer text containing the claim and its citation",
      "verdict": "supported",
      "rationale": "What the cited location actually supports, and its limits",
      "evidence": [{"path": "src/entry-store.mjs", "line": 1, "quote": "Exact frozen source excerpt"}]
    }
  ]
}
```

Inventory material assertions throughout the answer, including those outside the
required behavior outcomes. Each entry has a unique descriptive title and exact
answer text. Inspect the literal cited file, symbol, and line range against the
attributed claim, not merely whether some other source supports the behavior.
For a claim about a complete test body, check its setup, interventions, assertions,
and ending rather than certifying a nearby or truncated range. Ground an
`unsupported` assessment in the cited location and/or actual supporting source,
and explain the mismatch. A material claim missing a required citation also
receives an unsupported attribution assessment. Honest uncertainty is not itself
an error: judge what the answer actually asserts and how it scopes its evidence.
`unresolved` needs a nonempty `missingEvidence` string and remains incomplete.

Both conditions need this audit. Missing, empty, stale, malformed or ungrounded
inventories cannot establish full correctness. A context citation error fails the
downstream stage even if all required behavior outcomes pass. A baseline-only
citation error can support an observed quality difference, but never an
equal-quality efficiency comparison. `quality.citationAccuracy` counts fully
citation-supported answers; `quality.fullyCorrectAnswers` additionally requires
every behavior outcome. Existing outcome counts remain visible separately.
These are bounded reviewer attestations, not automatic discovery of all claims
or semantic proof that citations are adequate. Do not omit a known material error
merely because the behavior checklist did not ask about it.

Legacy protocols (absent `schemaVersion`, or version one) retain their historical
outcomes-only contract and record `citationAccuracy.required: false` with no
fully-correct-answer claim. Supplying a citation audit without freezing the new
policy is incomplete. Scorecard version 1.4.0 adds this distinction; keep existing
results unchanged and use new artifacts for any separately labeled reanalysis.

Record labels and distinct fingerprints detect mismatches/replays; they do not
prove that a coordinator actually launched independent sessions. The adapter,
coordinator, outcome adjudicator, and their raw audit records remain trusted.

Any context-assisted correctness or scope failure fails the downstream stage.
Missing pairs, stale records, unresolved outcome reviews, or missing ordering
evidence make it incomplete. A complete all-correct context condition improving
on the baseline can demonstrate an observed quality benefit; efficiency is not
then compared as though quality were equal. At equal complete correctness,
compare one prespecified metric (`calls`, `returnedBytes`, `totalTokens`, or
`elapsedMs`) and require the declared improvement threshold. Do not choose the
best-looking metric afterward. Report other metrics and upfront authoring,
review, and maintenance costs separately; do not infer dollar costs from zero
SDK accounting rates. No observed advantage is an informative result.

## Gate the next model trial

Keep the previous experiments immutable. Run calibration and boundary tests,
freeze the common rubric/protocol, review the source filter, then obtain the
normal explicit provider-run authorization. Score every unassisted attempt;
do not replace a miss with a later assisted revision. Review before promotion,
and report quality and cost without claiming causal benefit or a model ranking
from a small experiment. PR preparation/publication remains a separate authorized
step for the next trial, not something this scorecard performs automatically.

OpenAI Docs informed the separation of deterministic checks, grounded grading,
and downstream outcomes, as described in
[testing agent skills](https://developers.openai.com/blog/eval-skills) and
[evaluation best practices](https://developers.openai.com/api/docs/guides/evaluation-best-practices).
The finite controls calibrate this implementation; they do not establish the
accuracy of a future independent model reviewer.
