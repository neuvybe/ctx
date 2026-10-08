---
name: ctx
description: Initialize, populate, use, and maintain ctx-managed repository context. Use for ctx setup, coding tasks that use existing ctx context, evidence reconciliation, and local handoffs.
---

# ctx repository context

Use ctx to reduce repeated discovery without replacing source truth or the
repository owner's instructions. This skill needs filesystem access, Git, and
the ctx CLI; it does not itself install them or grant permission to change files.

## Establish the scope

Read the applicable owner instructions. Identify the requested repository,
context folder, and task: populate, consume, maintain, or explain. Respect a
custom folder and existing config rather than assuming every scaffold is
`.ctx/`. If multiple scaffolds could apply and the task does not identify one,
ask which scope to use.

Inspect existing config and `ctx --version` when command compatibility matters.
Read [the CLI contract](references/cli.md) when initializing, extending,
refreshing managed guidance, handling a custom folder, or encountering an older
layout. This skill describes layout v2; do not reshape legacy scaffolds to match.

Skill activation is not authorization to initialize, update, change sharing
mode, install add-ons, stage, commit, or publish. Answer/review tasks remain
read-only. For authorized writes, preserve existing facts and unrelated edits.
Do not write new owner instructions or operating policies merely to fill a
template placeholder.

## Consume context for a task

After owner instructions, read any installed owner-ratified operating
supplement, local continuation, and INDEX. Continuation is one clone's
checkpoint: confirm branch and worktree state before relying on it.

Follow INDEX to the smallest relevant fact set. Read a parent before its
relevant child; do not concatenate all context by default. Load architecture
for implementation and behavior for domain rules. Follow glossary or other
add-on routes only when that concern is needed.

Treat `draft` facts as leads and `verified` facts as scoped to their recorded
commit and sources. Check relevant evidence against current source/tests,
especially when metadata is stale, missing, or contradicted. Context is not
permission to skip source inspection before a consequential change.
When an explanation depends on a claim about test coverage or owner approval,
use the evidence-review pass below. Valid metadata and passing tests are not
substitutes for checking that claim.

## Populate or extend facts

When setup is requested, initialize only a missing scaffold or hydrate missing
team-local state through the CLI. New scaffolds default to team mode, behavior,
and glossary; respect explicit choices. Init generates prompts, not knowledge.
Do not replace an existing scaffold to make initialization succeed.

Inspect entrypoints, source, tests, and canonical requirements for the selected
scope. Fill installed documents by ownership:

| Concern | Owner |
|---|---|
| Purpose, users, scope, capabilities, and non-goals | `context/overview.md` |
| Domain concepts, decisions, workflows, transitions, and outcomes | `context/behavior.md`, when installed |
| Components, implementation flows, and technical invariants | `context/architecture.md` |
| Confirmed limitations and decision-relevant gotchas | `context/caveats.md` |
| Ambiguous project-specific terminology | `context/glossary.md`, when installed |
| Representations, extension points, or team procedures | Relevant installed add-on |

Do not silently add omitted documents or manufacture requirements. Distinguish
implemented behavior from owner-approved intended changes; source, tests,
and canonical requirements are evidence for different claims.
An observed limitation is not an accepted tradeoff without an owner decision
or explicit requirement. Record "owner decision not documented" when unknown.

Remove filled template prompts and inapplicable sections. Keep parent summaries
short; prefer linking canonical detail over copying it. If a coherent topic
needs deeper explanation, author a child under `context/`, give it its own
scope and metadata, and link it from the owning parent. Behavior topics belong
under `context/behavior/`; ctx does not generate their names or contents.
INDEX routes tasks, not duplicated facts, and behavior routes through its parent.

## Record evidence honestly

Every layout-v2 fact document, including nested Markdown children, has exactly
one metadata line:

```html
<!-- ctx:doc {"status":"draft","verifiedAt":"","sources":[]} -->
```

- Use `verified` only after checking claims. Record an actual immutable source
  commit as `<commit-hash> @ YYYY-MM-DD` and supporting repository-relative paths
  in `sources`. Paths are relative to the target repository, not the context
  folder. Include supporting files cited in the prose, not just the main source
  file. Do not cite the fact document itself as evidence.
- Listed sources must be tracked at that commit and match current evidence.
  Check relevant working-tree/staged changes as well as HEAD. Uncommitted source
  changes cannot be certified by stamping the old HEAD hash; leave affected
  facts draft and report the pending source commit. Do not commit merely to make
  readiness pass.
- Use `draft` for unknown, incomplete, inferred, changed, or unverified claims.
  Mark affected facts draft while reconciling them; uncertainty is a valid result.
  Clear `verifiedAt`, but retain known relevant `sources`. Revise paths when
  evidence changes; do not empty the list merely because work is uncommitted.
- Use `not-applicable` only for a genuinely irrelevant installed concern, with
  a brief reason. Do not use it to suppress unfinished work.

## Review material evidence claims

Before certifying authored facts or repeating material context claims, make a
brief claim-to-evidence pass for the relevant topic; no extra document is needed.

1. Identify the claim and its basis: implementation reasoning, a direct test
   assertion, or an owner requirement/decision. Read that evidence, not just
   the context's description of it.
2. For a "tested" claim, inspect the setup and assertions: which cases, fields,
   and state are checked, and when? A check at the end of several operations is
   not a check after each one. Test names and comments do not extend assertion
   coverage; an error assertion alone does not check unchanged state. Passing
   tests or statement coverage does not establish exhaustive behavioral coverage.
3. Match the wording to that scope and reconcile supporting files with metadata
   `sources`. Narrow an overclaim to the actual asserted case or to "follows
   from implementation"; report missing evidence or owner decisions as unknown.

Preserve supported behavior even when its claimed test coverage is overstated.
For read-only tasks, report discrepancies and proposed corrections without
editing. For authorized maintenance, correct the owning facts within scope;
do not add tests or owner decisions just to make a sentence true.

## Maintain and hand off

When context maintenance is in scope, reconcile the owning facts and child
routes for affected source changes. Preserve unrelated documents and retain
explicit implementation gaps. Reverify against a new source commit only when
the evidence supports it.

Refresh platform guidance with `ctx update` when requested/appropriate; that
command does not rewrite or reverify project facts. Put lasting discoveries in
their owning shared document. Record current objective, checkpoint, tests,
next action, blockers, and shared-context follow-ups in local continuation when
a handoff/checkpoint is requested. Do not put secrets or shared facts solely
in ignored state, or copy another clone's session as current truth.

For authorized population/maintenance, run `ctx doctor` and `ctx status` with
the selected repository/folder. Doctor checks structure and Git boundaries;
status checks metadata, listed-source freshness, and non-failing size guidance.
Neither proves that prose is true.

Report which documents changed, what was checked, and remaining draft/stale
evidence. A non-zero readiness check is not a reason to invent facts or widen
scope. ctx never stages or commits; leave the sharing decision to the human.
