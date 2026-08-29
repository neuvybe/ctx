# CONTINUE.md — continuation prompt (living state) for {{PROJECT}}

> **Purpose:** load this into a fresh agent after a compaction so it can resume
> without re-deriving context. This is the **living** file — update it at the
> end of every exchange. The *rules* live in `OPERATING.md`; this file holds
> *where we are*.

---

## Resume protocol (do this first, before any work)

1. Read in this order: `OPERATING.md` (binding mode) → this file (current state)
   → `INDEX.md` (repo orientation + load order) → the specific `context/*.md`
   docs relevant to the next step. All under `.ctx/`.
2. Recall the hierarchy: the owner's canonical instructions
   ({{OWNER_INSTRUCTIONS_PATH}}) govern repo work and take precedence over
   `OPERATING.md` (see `OPERATING.md` §0a). Owner/founder: {{FOUNDER}};
   collaborator on this machine: {{COLLABORATOR}}.
3. State the current state back to the director in 2–3 lines: mode, last done,
   in-flight, proposed next (from the sections below).
4. Default to **build mode, stopped, awaiting direction** — unless the state
   below says otherwise, do not implement anything. Propose the next smallest
   step (with its §4 concept attestation) and wait.
5. If "Proposed next (awaiting approval)" below is non-empty, that is your
   pending proposal — re-surface it and wait. Do not silently start it.

Do not begin implementation on the first response. The first response is a state
checkpoint, not a work session.

---

## Mode

Current: **build mode, default stopped.** (Set to exploration when the design
space is open and the director asks for options.)

---

## Last completed

- **Generated `.ctx/` context** from `<commit>` on {{DATE}} (initial scaffold
  via the `ctx` platform; `context/*.md` filled per `docs/fill-context-workflow`
  of the platform). Replace this with the real first completed step once work
  begins.

## In flight

- **None.** (If this is ever non-empty after a compaction, the previous session
  violated the stop protocol — surface it explicitly rather than silently
  continuing.)

## Proposed next (awaiting approval)

- **Awaiting direction.** (Keep a short candidate menu here when there are
  known next steps; each a separate build-mode step to propose individually.)

## Open questions / parked concepts

- **None parked.** (Concepts raised mid-step that were deferred go here with a
  one-line note + a pointer to where they surfaced, so they're not lost across
  compaction.)

## Decisions log (compact, append-only)

- {{DATE}} — `.ctx/` scaffolded via the `ctx` platform; ignored via
  `.git/info/exclude` (repo-local, non-tracked). Owner instructions at
  {{OWNER_INSTRUCTIONS_PATH}} govern; `OPERATING.md` is a supplement.

---

## How to update this file (at the end of each exchange)

Keep it honest and short. A stale continuation file is worse than none.

- **Mode:** build/exploration as directed.
- **Last completed:** move the just-finished step here (one line).
- **In flight:** almost always "None."
- **Proposed next:** the next smallest step awaiting approval, or "Awaiting
  direction" + a candidate menu.
- **Open questions / parked concepts:** add any concept raised but not decided;
  never delete an entry until it's resolved in the decisions log.
- **Decisions log:** append one line per ratified decision with the date.

Do not edit `OPERATING.md` to reflect state — that's the constitution.