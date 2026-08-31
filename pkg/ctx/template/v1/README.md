# `{{FOLDER}}/` — agent context for {{PROJECT}}

<!-- ctx:managed begin -->
> **Active visibility mode: `{{MODE}}`.** `ctx` separates durable project context from
> machine-local session state. Team mode leaves the durable files visible to Git
> for review and sharing; local mode excludes this entire folder through Git's
> repository-local exclude file. The configured continuation path,
> `{{CONTINUE_PATH}}`, remains local. New scaffolds use `local/CONTINUE.md`;
> legacy scaffolds retain root `CONTINUE.md`. The CLI never stages or commits
> generated files.
<!-- ctx:managed end -->

<!-- ctx:user — fill this once; ctx update preserves it verbatim (it's outside the managed blocks) -->
Owner's canonical agent instructions live at: {{OWNER_INSTRUCTIONS_PATH}}
<!-- /ctx:user -->

<!-- ctx:managed begin -->
## Audience and ownership

The project owner's canonical instructions govern repo work. `OPERATING.md` is a
supplement, not an override. In team mode, changes to durable context and policy
are ordinary reviewable repository changes. Personal notes and living session
state belong at `{{CONTINUE_PATH}}`; their Git visibility follows the scaffold
mode.

## What's here

| Class | Files | Role |
|---|---|---|
| **Durable context** | `README.md`, `OPERATING.md`, `INDEX.md`, `REVIEW.md`, `context/*.md` | Stable working agreements, orientation, review guidance, and verified project knowledge |
| **Local state** | `{{CONTINUE_PATH}}` | Living session state and resume protocol |
| **Platform-managed portions** | Managed blocks in `README.md` and `REVIEW.md`, `.ctx-version`, and configuration when present | Upgradeable guidance and scaffold metadata; this maintenance ownership is separate from Git visibility |

**Constitution vs. log split:** `OPERATING.md` is the stable rules of engagement
(changed through review/ratification). `{{CONTINUE_PATH}}` is the local living
state and may change every session.

## Load order for a fresh or resumed agent

1. **`OPERATING.md`** — stable operating guidance.
2. **`{{CONTINUE_PATH}}`** — local state and resume protocol.
3. **`INDEX.md`** — repo orientation and context-doc routing.
4. The specific `context/*.md` docs the next step needs.

## Keeping this folder healthy

- **Hydrate a fresh team clone:** run `ctx init --folder {{FOLDER}}` to create
  the missing ignored continuation without rewriting durable files.
- **Update `{{CONTINUE_PATH}}` at the end of every exchange.** A stale
  continuation file is worse than none.
- **Review shared context like code.** In team mode, reconcile relevant
  `context/*.md` files when the implementation changes.
- **Keep local state local.** Do not force-add `{{CONTINUE_PATH}}`; never store
  secrets in agent context merely because Git ignores it.
- **Respect the configured mode.** `ctx update` preserves it and never converts
  a legacy or local scaffold to team mode.
- **Stay current:** run `ctx update --folder {{FOLDER}}` after upgrading the CLI
  to refresh managed blocks, then review any Git-visible diff.

## What this folder is not

- Not user documentation—the project's public documentation is separate.
- Not an authority above owner-maintained instructions.
- Not a substitute for source verification; code remains the source of truth.
- Not secret storage. “Local” describes Git visibility, not access control.
<!-- ctx:managed end -->
