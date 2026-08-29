# `.ctx/` — agent working context for {{PROJECT}}

<!-- ctx:managed begin -->
> This folder is the collaborator's private working context — not shipped
> product. It is gitignored via `.git/info/exclude` (repo-local, non-tracked),
> not the repo's `.gitignore`, so it stays private without touching the
> project's tracked files and survives syncing. It is distinct from the
> project owner's tracked instruction namespace.
<!-- ctx:managed end -->

<!-- ctx:user — fill this once; ctx update preserves it verbatim (it's outside the managed blocks) -->
Owner's canonical agent instructions live at: {{OWNER_INSTRUCTIONS_PATH}}
<!-- /ctx:user -->

<!-- ctx:managed begin -->
## Why a private folder (not the owner's namespace)

The owner's agent instructions live in tracked locations. This `.ctx/` folder is
the collaborator's own working context, kept strictly separate to avoid
collision. The owner's rules govern repo work; `OPERATING.md` here is a
session-discipline supplement, not an override.

## What's here

| Kind | Files | Role |
|---|---|---|
| **Governing** | `OPERATING.md`, `CONTINUE.md`, `INDEX.md`, `REVIEW.md` | How we work + where we are + how to orient + pre-PR review |
| **Reference** | `context/*.md` | Knowledge about the repo (architecture, formats, extension points, known issues, glossary) |

**Constitution vs. log split:** `OPERATING.md` is the stable rules of engagement
(changes only on ratification). `CONTINUE.md` is the living state (changes every
session). Keeping them apart means iteration is cheap.

## Load order for a fresh / resumed agent

1. **`OPERATING.md`** — the binding operating mode (session discipline).
2. **`CONTINUE.md`** — current state + the resume protocol (do this first).
3. **`INDEX.md`** — repo orientation + which `context/*.md` to pull.
4. The specific `context/*.md` docs the next step needs.

A compaction only costs re-reading the governing files (~300 lines) plus 1–2
relevant context docs — not the whole folder.

## Keeping this folder healthy

- **Update `CONTINUE.md` at the end of every exchange.** A stale continuation
  file is worse than none.
- **Don't let reference docs drift.** When code changes, update the matching
  `context/*.md` in the same step.
- **Don't commit any of this.** It's ignored via `.git/info/exclude` on this
  clone. To share a doc, promote it to a tracked location the owner reviews.
- **Clone-local caveat:** `.git/info/exclude` doesn't follow a fresh clone.
  Re-run `ctx init` on a new clone, or keep your `.ctx/` backed up.
- **Stay current:** run `ctx update` to refresh the platform-managed files
  (this README + `REVIEW.md`) when you upgrade the `ctx` CLI.

## What this folder is *not*

- Not user documentation — the project's public `README.md` is separate.
- Not a source of truth — the code is. Verify against source before relying on a
  detail in a build step.
- Not the owner's canonical instructions — defer to them on conflict.
- Not permanent — anything here can be rewritten or deleted to stay accurate.
<!-- ctx:managed end -->