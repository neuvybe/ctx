# OPERATING.md — agent operating mode for {{PROJECT}}

> **Status:** binding for every agent session producing code in this repo.
> **Adapt and ratify:** customize §0a and §8 to this project's reality, then
> treat as the constitution (change only on ratification; record changes at the
> top with date). This is a *supplement* to the owner's canonical instructions —
> see §0a.

## 0. The one-line version

> Default state is **stopped, awaiting direction**. Propose the smallest step →
> get explicit approval → implement exactly that → show it → stop. No concept is
> implemented before approval. Small PRs are the success condition, not a
> compromise.

## 0a. Hierarchy — this is a supplement, not an override

The project owner's canonical agent instructions live at
{{OWNER_INSTRUCTIONS_PATH}}. Those **govern all repo work and take precedence**
over this file. `OPERATING.md` is the *collaborator's session-discipline
supplement* — it refines the increment protocol, the concept gate, and the stop
protocol in more detail. If anything here conflicts with the owner's rules, the
owner's rules win; surface the conflict rather than acting on it.

> **Fill in:** who is the owner/founder ({{FOUNDER}})? who is the collaborator
> directing the agent on this machine ({{COLLABORATOR}})? Adjust the "pair"
> framing in §0–§2 to the actual relationship (e.g. if the collaborator IS the
> owner, the pair = the owner themselves and the approval loop is internal).

## 1. Two modes — know which one you're in

| Mode | When | Your job | Default deliverable |
|---|---|---|---|
| **Build mode** | A concrete change is directed | Implement the approved increment, nothing more | Code + compact result |
| **Exploration mode** | Design space still open; "what are the options?" | Surface options + tradeoffs, **no code** | Written options analysis |

When in doubt, you are in **exploration mode.** If the director goes quiet: in
build mode, **prep and wait** (produce the next smallest-step proposal ready to
approve, then stop); in exploration mode, keep surfacing options until redirected.

## 2. Build mode — the increment protocol

1. **Propose** the next *smallest* step, with its design intent (one short
   paragraph: what, why, what file(s), what concept-categories it touches — §4).
2. **Wait** for explicit approve / adjust / reject. Silence is not approval.
3. **Implement exactly that and nothing more.** No drive-by refactors, no scope
   creep. If you discover mid-step that a concept (§4) is involved that wasn't
   surfaced, **stop the step** and surface it.
4. **Show the result compactly** — a focused diff or short summary plus the
   verification you ran. **For a PR-bound change, run the pre-PR adversarial
   review pass (`REVIEW.md`) before opening the PR**; findings are proposals
   (§7), not patches.
5. **STOP.** Then you may propose the next smallest step and wait.

## 3. Concept-approval before implementation (the core guardrail)

A **concept** is any design decision not verbatim in an approved plan or an
established repo pattern — the expensive failures to reverse (schema shapes,
struct/class structures, naming, abstractions, library choices, error
taxonomies, …). None is implemented before approval. A concept you surface
mid-step **stops the step.**

> **Fill in:** list this project's concept categories (e.g. on-disk format
> changes, AST/node kinds, hash-key design, prompt wording that invalidates
> caches, …). Project-specific.

## 4. The concept attestation

Before any build-mode step, attest explicitly: *"This step touches
concept-categories: [none | list them]."* If none → fast lane (§5). If any →
awaiting approval on the specifics. Make it an artifact in your proposal, not a
hope.

## 5. The no-concept fast lane

Provably concept-free changes (typo/comment fixes, test-only additions mirroring
existing shape, doc edits to `.ctx/`, mechanical renames with no API change) take
the fast lane: attest "none" and proceed, still show + stop. **Never** fast-lane
anything touching a §3 category, anything establishing a pattern others copy, or
anything in the project's sensitive core. When in doubt: it isn't concept-free.

## 6. Never batch concepts — but one *family* per exchange

Genuinely coupled decisions shouldn't be force-serialized into artificial
round-trips. Rule: **one concept-family per exchange**, surfacing the
decomposition first and confirming the family before deciding members together.

## 7. Review findings are proposals, not patches

If acting as a reviewer (or surfacing findings from any tool), present findings
**per-finding.** Only what's approved gets incorporated. Never silently mutate
intent under the guise of "fixing." (The concrete review pass is `REVIEW.md`.)

## 8. Support functions — status (project-specific)

> **Fill in:** what automated gates exist in *this* project? Replace this table.
> Do not rely on a gate that isn't present.

| Function | Status in {{PROJECT}} |
|---|---|
| Automated CI / lead gate | ❓ |
| Scribe protocol | ❓ |
| Second-agent review pass | ✅ `codex review` (see `REVIEW.md`) |
| Verification battery | ❓ — `make test` / `make vet` / `make build` / etc. |
| Dependabot / external PR triage | ❓ |

**Implication:** because there may be no automated gate, the "show the result"
step in §2 must include the verification you actually ran. A step is not "shown"
until it is shown verified.

## 9. Small PRs are the success condition

Optimize for oversight depth, not throughput. If a step grows past one focused
change, it's too big — propose splitting it.

## 10. Stop protocol

After showing a verified result: do not start the next implementation; you may
propose the next smallest step and wait; do not open a second work front; if the
director is quiet, prep the next proposal and do not implement.

## 11. Updating this file

This is a constitution, not a log. Change it only on ratification; record the
change at the top with a one-line note + date. Iteration of *state* goes in
`CONTINUE.md`, not here.