# Bounded evidence audit

Use this for requested evidence review, certification of authored facts, and
material coverage/approval claims a task relies on. Ordinary context consumption
does not require auditing every sentence or every document.

## Observable result

In the review output, give a compact table or equivalent entries containing:

- **Claim:** the material statement being checked, with its owning document.
  Split assertions with different evidence bases or scopes into separate entries.
- **Basis:** implementation, direct test assertion, or documented owner
  requirement/decision. More than one basis may apply, but identify each.
- **Evidence and limit:** repository-relative file and symbol/section, the
  actual observation supporting it, and the boundary of that support.
  For a direct-test claim, identify the executed intervention separately from
  the asserted outcome: cases/actors, fields/state, timing, and named events
  that were not executed. Compare each part of the claim, not just its outcome.
- **Verdict:** supported as written, narrow the wording, or unresolved. Supply
  narrower wording or the missing evidence when needed.

This is an evidence artifact, not private reasoning or another default context
document. Include supported claims as well as discrepancies so the audit's
coverage is inspectable. State what was not checked; do not certify unreviewed
claims with an "everything else matches" conclusion. Prioritize claims about
public boundaries, state changes, failure invariants, persistence, concurrency,
and owner acceptance when they are material to the requested topic.
For a public-surface inventory, compare the claimed boundary with relevant
export declarations and signatures, not just the main callable names.

## Match the strength of the statement

Inspect evidence independently of the context's description. For a test claim,
read the complete cited test body. Before judging a quantifier or lifecycle
claim, include a compact case inventory in the audit: relevant operations in
source order, their asserted results, and which state assertions follow each
operation. Include later cases, not only the first example that disproves the
claim. Derive counts and proposed test-coverage wording from that inventory;
an assertion about one operation is not a count of all operations in the test.
Identify the actual intervention/setup, fields/state asserted, and timing of
those assertions. Distinguish what the test exercises from what its name or
comment suggests. A final check after several operations is not a check after
each operation; an error alone is not a state comparison. An equivalent expected
outcome does not make two different interventions the same test. Passing tests
or statement coverage cannot certify unasserted combinations or outcomes.
Check counts and quantifiers against the actual cases; do not call a test's
coverage complete while omitting another rejection or operation in its sequence.
When the claim names a lifecycle event, storage boundary, or concurrency
scenario, explicitly say whether that intervention occurred in the test.

Implementation can support behavior without a direct test. Requirements can
support intended behavior without proving implementation or test coverage.
Owner acceptance requires a decision/requirement about that particular tradeoff,
not merely an adjacent requirement or a small-project description. An absence
claim is bounded to the materials inspected, not all possible owner knowledge.

Preserve those distinctions in the proposed wording. Do not downgrade correct
behavior merely because it lacks a direct test, or invent a decision/test to
make the original sentence true. A verified document may accurately state that
a case is not directly tested or an owner decision is not documented. Unresolved
unsupported assertions remain draft; corrected, adequately supported wording
can be verified subject to the normal source-commit rules.

## Recommended maintainer checkpoint

Recommend human source-level review for newly authored or materially changed
claims that a reader could treat as consequential guarantees. Examples include
persistence/recovery or access guarantees, business rules and public contracts,
broad failure invariants or exhaustive test coverage, and decisions attributed
to the owner. Consequence and scope determine the need, not the filename or
every use of words such as "all" and "never".

The agent prepares the bounded audit and runs available mechanical checks. In
the handoff, identify the consequential changed claims, exact evidence, limits,
and any pending review. A maintainer familiar with the relevant code checks
the final wording against that evidence, ideally during normal pull-request
review. Check proposed corrections too: a reviewer can correctly reject an
overclaim while introducing another. An audit or a second model's agreement is
transparency, not proof; human review is not infallible either.

For example, two rejected writes followed by one assertion about one key can
support "that key has its expected value after both rejections." They do not
directly test every key after each rejection. Keep that narrower statement;
missing test coverage is not proof of an implementation bug.

This checkpoint is recommended practice, not a new CLI-enforced rule or
permission to author owner policy. Apply it when requested or established by
owner instructions. While a required checkpoint is pending, keep the owning
fact document draft, clear `verifiedAt`, and retain relevant sources; metadata
is document-level, even when other claims in it have been checked. After review,
use verified only when the final claims satisfy the source-commit rules. Record
review scope through the project's existing review mechanism; no new default
document or reviewer metadata field is needed. Do not report approval unless it
actually occurred.

Unchanged, previously reviewed claims need no repeat checkpoint solely because
another document was edited. Revisit them when relevant evidence changes or a
contradiction appears. Routine navigation, terminology, and ordinary context
reading do not need blanket human approval. Verified metadata records evidence
checking at a commit, not who approved it or a timeless truth guarantee.

## Citation bookkeeping

Supporting files cited in prose or the audit must be covered by the owning
fact's metadata `sources`. A real directory source can cover its contained
files; a filename prefix cannot. For new explicit evidence references, this
optional convention permits deterministic checking, one path per line:

```markdown
Evidence source: `repo-relative/path.go` — symbol or section, and observed support
```

Reconcile citations before closing the semantic audit. For each audited fact,
report citation gaps or the supporting paths checked against its metadata;
merely reading a cited file does not establish metadata coverage.

When Node.js is already available and explicit references use this convention,
run the skill's read-only helper rather than relying solely on a manual scan:

```bash
node /path/to/ctx/skill/scripts/check-evidence-citations.mjs \
  --repo /absolute/repository --folder .agent
```

It checks only `Evidence source:` lines in Markdown facts under `context/`,
ignores fenced examples, and reports missing/unsafe paths or metadata coverage.
It does not infer citations from arbitrary prose, assess truth, check Git
freshness, or certify citation completeness. Zero recognized citations is not
a passing completeness audit. Reconcile other prose citations manually and use
`ctx status` for its separate listed-source/commit checks. Do not install a
runtime or rewrite existing context merely to use the helper. If it cannot run,
state that limitation and perform the same path-to-metadata comparison manually.

## Close the audit within scope

For read-only work, report the audit and proposed corrections; do not save a
checkpoint or change metadata. For authorized authoring/maintenance, resolve
overclaims in the owning facts before certification, retain relevant sources,
and report remaining draft claims. A fresh reviewer can check this artifact,
but freshness or agreement between reviewers is not itself proof of correctness.
