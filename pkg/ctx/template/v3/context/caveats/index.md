# Caveats — {{PROJECT}}

<!-- ctx:doc {"status":"draft","verifiedAt":"","sources":[]} -->

> **Record confirmed, decision-relevant caveats only, then delete this note.**
> Keep this document near 400 words. For `verified`, set `verifiedAt` to
> `<commit-hash> @ YYYY-MM-DD` and list supporting source, test, or owner-document
> paths. Distinguish product behavior from environment/tooling constraints. If
> there are no material caveats, use `not-applicable` and state why. Remove
> resolved entries once their history belongs in a changelog, issue, or ADR.
> Observed behavior does not establish owner acceptance. Use "accepted
> limitation" only with an explicit requirement or decision record. Describe
> what the evidence establishes; a passing test is not proof of every invariant.

## Active caveats

### <Short, specific name>

- **Status:** active / accepted limitation / under investigation
- **Impact:** who or what is affected
- **Symptom:** what an agent or developer will observe
- **Evidence:** source path, test, issue, or reproducible command
- **Workaround:** safe current response, if one exists
- **Owner decision (if documented):** requirement or decision pointer; otherwise
  "owner decision not documented" (do not infer acceptance)

## Environment and tooling constraints

> Include only constraints that change how work is verified or shipped. Never
> classify an environment failure as benign without evidence.

## Focused topic routes

> During population, give independently useful explanations a sibling
> `<topic>.md` with its own scope and `ctx:doc` evidence metadata, and link it
> here with a brief description of when it is useful. Keep shared context here,
> not copied topic detail. Short related facts may stay together. Remove this
> prompt and unused sections when filled; ctx creates no topic stubs.

| Topic or question | Read |
|---|---|
| [project-specific topic] | `<topic>.md` |

> Verification applies to this document's own claims and sources, not to linked
> children. Check each needed topic's metadata and evidence independently.
