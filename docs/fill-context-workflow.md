# Fill-context workflow

This workflow turns a layout-v2 scaffold into useful, evidence-backed project
context. The templates are prompts, not answers: an agent must inspect the
actual repository and record what it verified, what remains draft, and what does
not apply.

## 1. Confirm the sharing boundary and selected scope

- `ctx init` defaults to **team mode**: durable files are available to review and
  share, while `<folder>/local/CONTINUE.md` stays ignored.
- `ctx init --mode local` keeps the whole selected folder ignored through the
  repository's common `.git/info/exclude`.
- `ctx` never stages or commits files in either mode.
- The v2 core is intentionally small. New scaffolds select the glossary add-on
  by default because project-specific language is commonly useful; omit it for
  a core-only scaffold with `--without glossary`. Select other add-ons with
  repeatable/comma-friendly `--with`, or install one later:

```bash
ctx init --without glossary
ctx init --with operating,contracts
ctx add --list
ctx add review
```

Keep an add-on only when its concern is real. An empty document adds routing and
maintenance cost without adding context; an installed glossary can instead be
marked `not-applicable` with a reason if inspection shows ordinary language is
sufficient.

## 2. Read authority before context

Start with the repository's canonical owner instructions, such as `AGENTS.md`,
`CLAUDE.md`, `CONTRIBUTING.md`, or an owner-authored skill. Those instructions
govern. If the optional `OPERATING.md` is present, it is a project-owned
supplement and must not duplicate or override higher-priority guidance.
Replace `{{OWNER_INSTRUCTIONS_PATH}}` in README.md and INDEX.md with that
canonical repo-relative path; the placeholder is intentionally owner-supplied.

Then inspect the root README, manifests, build/test configuration, entrypoints,
and top-level source tree. Commit messages and existing documentation are leads;
verify important claims against current source.

## 3. Fill the core from parent summary to specialized detail

1. **`context/overview.md`** — users, current purpose, capabilities, boundaries,
   non-goals, maturity, and canonical roadmap pointers. Keep technical flow
   detail out of this parent summary.
2. **`context/architecture.md`** — components, entrypoints, important flows,
   invariants, state ownership, lifecycle/concurrency where relevant, and active
   runtime integrations. Cite source paths.
3. **`context/caveats.md`** — confirmed limitations and gotchas that change how
   an agent should work. Include evidence and a safe workaround; distinguish
   product behavior from environment constraints. Do not create a speculative
   bug backlog.
4. **Fact add-ons** — fill the default-selected `glossary` plus `contracts` and
   `extending` when installed. Use `not-applicable` with a reason if later
   inspection proves an installed concern does not apply.
5. **`INDEX.md`** — verify that routing names only installed documents and sends
   readers to the smallest relevant set. Keep facts in their owning document,
   not in the router.
6. **`local/CONTINUE.md`** — record only current objective, repository position,
   completed/in-flight work, verification, next action, blockers, and shared-doc
   follow-ups. Durable discoveries belong in shared context or canonical project
   records.

If the `operating` or `review` add-on is installed, the repository owner should
fill its project-owned policy/profile. Do not invent authorization rules, a base
branch, review tools, or required checks.

## 4. Maintain document metadata

Every v2 project-fact document begins with one JSON line:

```html
<!-- ctx:doc {"status":"draft","verifiedAt":"","sources":[]} -->
```

Use it as follows:

- `draft` — incomplete, inferred, or not yet checked at the current source.
- `verified` — claims were checked; set `verifiedAt` to
  `<commit-hash> @ YYYY-MM-DD` (use `git rev-parse HEAD`, never a mutable ref)
  and list supporting repo-relative paths in `sources`.
- `not-applicable` — the concern genuinely does not apply; retain a short reason
  in the document so the status is intentional.

Keep the line valid JSON. Record unknowns explicitly. Verification is scoped to
the listed commit and sources; it is not a timeless guarantee.

## 5. Keep context hierarchical and bounded

Prefer a short parent summary that routes to detail. Link canonical documentation
instead of copying it. Delete template instructions and inapplicable sections.
Useful guidance targets are:

| Document | Target |
|---|---:|
| `INDEX.md` | about 250 words |
| `local/CONTINUE.md` | about 300 words |
| `context/overview.md` | about 500 words |
| Other project-fact documents (`context/**/*.md`) | about 800 words |

These are guidance, not hard limits. For the listed mechanics and project-fact
documents, `ctx status` emits non-failing warnings only around twice those sizes
so legitimate project complexity is not marked unhealthy. When a file grows,
split a coherent child document and route to it
from INDEX's project-owned routing section rather than flattening all detail
into the always-read path. Lifecycle updates preserve that section. Put the same
`ctx:doc` metadata line on each nested project-fact Markdown document so status
tracks it too.

## 6. Check structure separately from readiness

Run both checks after filling:

```bash
ctx doctor --folder .ctx
ctx status --folder .ctx
```

`doctor` checks scaffold structure, layout/configuration, managed-marker
grammar, and effective Git visibility/privacy. `status` summarizes metadata and
size guidance. Neither proves that prose is true; source verification does.

For a custom folder, repeat the same `--folder` value. In team mode, inspect
`git status --short <folder>` and let the human decide what to stage. In local
mode, commit nothing from the ignored scaffold.

## 7. Reconcile when the repository moves

For each affected fact document:

1. change its status to `draft` while claims are being reconsidered;
2. reread the changed source and relevant tests;
3. update only the owning document and its child links;
4. restore `verified` with the new commit/date and source list;
5. update local continuation with any remaining work.

## V1 compatibility

Do not manually reshape an existing schema-v1 or config-less legacy scaffold to
match this workflow. V1 retains its original file set, root continuation where
applicable, unnamed managed markers, and frozen update templates. `ctx update`
selects the compatible source; layout conversion requires an explicit supported
flow rather than file moves by convention.
