# REVIEW.md — pre-PR adversarial review pass for {{PROJECT}}

<!-- ctx:managed begin -->
> **Purpose:** a lightweight, second-agent review loop using `codex review`
> (ChatGPT-authed) to get quick adversarial feedback on a change **before**
> opening a PR. In team mode this is shared review guidance, not automated repo
> infrastructure. Findings are **proposals**
> (per `OPERATING.md` §7), not auto-applied patches.

## When to run it

- **Before opening a PR** (main case): after a change is locally verified and
  before `gh pr create`.
- **After merging a small change** if you want a post-hoc second opinion.
- **Not** a substitute for the owner's human PR review — it's a second pair of
  eyes for the collaborator.

## Model + effort (deliberate, documented)

- **Model: `gpt-5.6-sol`** — the frontier agentic coding model, best for
  judgment-heavy review. Alternatives: `gpt-5.6-terra` (workhorse, faster) for
  trivial diffs; `gpt-5.6-luna` is for clear/repeatable tasks, **not** review.
- **Effort: `model_reasoning_effort=high`** for routine review. The codex bundled
  catalog for `gpt-5.6-sol` supports `low` / `medium` / `high` / `xhigh` / `max`
  / `ultra` (in increasing order — `ultra` is the *highest reasoning level*,
  distinct from the desktop app's separate "Ultra" *subagent mode*). Use
  `xhigh` / `max` / `ultra` for large or hard diffs; `medium` for trivial.
- Auth: ChatGPT sign-in (recommended GPT-5.6 models). Check with
  `codex login status`.

## Invocation

`codex review` is non-interactive and non-mutating. Pick **one** target
(`--commit` / `--base` / `--uncommitted`) — these conflict with a custom PROMPT,
so for custom review instructions either drop the target and pass a prompt, or
put them in an `AGENTS.md` the owner owns.

```bash
# Pre-PR: review the current branch against main
codex review --base main \
  -c model="gpt-5.6-sol" -c model_reasoning_effort="high"

# Post-merge: review a specific commit
codex review --commit <SHA> --title "<commit title>" \
  -c model="gpt-5.6-sol" -c model_reasoning_effort="high"

# Working-tree (uncommitted) review
codex review --uncommitted \
  -c model="gpt-5.6-sol" -c model_reasoning_effort="high"
```

For a big/hard diff, swap `high` → `xhigh` / `max` / `ultra`. Capture with
`| tail -N` or redirect to a file.

## Scope & safety notes

- **Diff-scoped:** `--commit`/`--base`/`--uncommitted` review Git changes. In
  team mode, edits to shared `{{FOLDER}}/` files are therefore part of the
  review; `{{FOLDER}}/{{CONTINUE_PATH}}` remains ignored.
- **Exploratory noise is benign:** the review agent may run exploratory
  `rg`/`grep` and list files under build/cache dirs. That's exploration, not the
  verdict — ignore it. The verdict is the natural-language summary at the end.
- **Sandbox `/tmp` errors are benign:** if the agent tries a sandboxed build/make
  and hits `Operation not permitted`, it doesn't block the review — codex
  reasons from the diff. Don't treat those as failures.
- **`AGENTS.md` gap:** codex reads `AGENTS.md` (not `CLAUDE.md`); if the owner's
  canonical instructions aren't at `AGENTS.md`, they aren't auto-applied to
  codex reviews. Pass a custom prompt (drops the target) or ask the owner to add
  an `AGENTS.md`. For changes the owner's rules flag as needing "adversarial
  review," **definitely** run this pass.

## How findings are handled

Per `OPERATING.md` §7: findings are **proposals to the collaborator**, not
patches. Triage each; only approved fixes go in before the PR. `codex review`
does not apply changes.
<!-- ctx:managed end -->

## Dry-run (optional — fill when you first run the pass)

> Record the exact command + the verdict, so future sessions see the expected
> output shape. ctx update preserves this section (it's outside the managed
> block above).

```
# command:
codex review ...
# verdict:
...
```
