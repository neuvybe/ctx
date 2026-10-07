# ctx skill forward-testing protocol

The [skill](../../skills/ctx/SKILL.md) should improve an agent's decisions about
repository context, not simply produce familiar headings. This protocol uses
the [booking demo](../../testdata/booking-demo/README.md) rather than ctx's own
implementation as the subject project.

The [recorded forward-test results](results.md) document the first run,
including the baseline's success and the limits of metadata-based activation.

Prepare a new destination for every independent population trial with
`tools/prepare-skill-demo.mjs`. Use `--with-skill` only for skill trials.
Record the starting source commit, CLI version, model/runtime, request, and
actual checks. Do not give agents the evaluator's intended answer or scorecard.
Do not run model evaluations automatically in ordinary CI.

## Population comparison

Use this request for both the baseline and skill-assisted task:

> Initialize team context for this booking demo using the custom folder .agent.
> Populate its default documents from current source, tests, and requirements
> so another developer can understand and work on it. Record a useful local
> checkpoint. Do not change application files or owner instructions, stage,
> commit, publish, install globally, or change sharing mode. Run relevant
> checks and report unresolved draft facts honestly.

For explicit use, add the skill name and its installed path. For a
description-selection trial, expose only its name, description, and path as
available-skill metadata; do not instruct the agent to load it.

Each task starts from the same fixture, without an active scaffold. Installing
the skill changes that trial's initial commit, so compare actual source
content and outcomes rather than requiring identical commit hashes.

## Read-only use

Use a fresh agent against successfully populated context:

> Using existing context, explain what cancellation does to available seats
> and whether cancelling the same booking twice is safe. Do not change files.

Record which context branches and relevant source/test evidence it consulted.
Compare file contents, Git HEAD, and the index before/after to verify the
read-only boundary, including ignored continuation. Do not equate a
self-reported read list with a complete audited tool trace.

## Reconcile uncommitted evidence

In a separate copy of populated context, change
`MaxSeatsPerBooking` from four to two in the demo's source. Do not commit it.
The requirements refer to the source constant and the tests are limit-relative,
so the source diff alone exercises a real business-rule change.

Give a fresh agent this request without supplying the intended new answer:

> Supporting implementation has changed in the working tree. Reconcile the
> affected context with current source and tests and record a local handoff.
> Do not change application files or owner instructions, stage, or commit.

The changed behavior should be described from evidence. Affected facts cannot
honestly claim verification at the old source commit. Expect draft readiness
and a clearly reported pending source commit, not a forced green status.

## Observable acceptance criteria

| Concern | Evidence to inspect |
|---|---|
| Domain accuracy | Claims agree with source/tests/requirements, including capacity, immediate confirmation, cancellation, identifier reuse, and memory-only state |
| Honest evidence | Real immutable commit/date, repository-relative tracked sources, no invented verification or not-applicable escape hatch |
| Useful routing | Ownership is clear; topic detail is routed when needed; no preset irrelevant topic files |
| Sharing boundary | Durable team facts visible, local continuation ignored, requested folder/mode retained |
| Scope preservation | No unauthorized application/owner changes, staging, commits, publication, or global installation |
| Appropriate checks | Structural health is distinct from readiness; a pending source commit is reported accurately |

Use deterministic Git/CLI checks for mechanical criteria and inspect actual
prose for semantic claims. Record failures and unknowns separately. A small
same-runtime comparison does not establish cross-model superiority; the
baseline may already succeed.

Skill loading should be observed, not assumed. Supplied metadata tests selection
under that input, while native Codex discovery needs a separate host integration
check. Report that distinction alongside trial outcomes.
