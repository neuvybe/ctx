# Using ctx with an Agent Skills-compatible skill

ctx includes a portable [Agent Skills-compatible](https://agentskills.io/specification)
[skill](../skills/ctx/SKILL.md), with optional OpenAI-specific metadata. It teaches
an agent how to populate, consume, and maintain ctx-managed context. Its
[CLI reference](../skills/ctx/references/cli.md) is bundled with it so the installed
skill does not depend on this repository's documentation being present.

The skill is agent-facing workflow guidance. The
[worked example](examples/ctx/README.md) is a human-readable reference result;
neither makes project facts appear automatically.

## What ships and what does not

The maintained bundle lives at `skills/ctx/`:

```text
skills/ctx/
├── SKILL.md
├── references/
│   └── cli.md
└── agents/
    └── openai.yaml
```

`SKILL.md` follows the open format: YAML `name` and `description` metadata,
followed by Markdown instructions and relative links to supporting references.
The Markdown workflow is provider-neutral. `agents/openai.yaml` is an optional
OpenAI-specific extension for UI metadata, not a dependency of the core workflow.
Each host still needs its own supported discovery/loading mechanism; format
compatibility does not mean every host integration has been tested.

The skill bundle describes the reusable workflow. A project's `.ctx/` (or
custom context folder) holds that project's facts and local handoff state;
it is not itself an Agent Skills bundle. The agent needs filesystem access,
Git, and an installed ctx CLI. Plain model access without those tools cannot
execute the workflow.

`ctx init`, the npm launcher, and `ctx update` do not install this skill.
This is a separately installed repository-owned bundle, not a CLI installer,
plugin, or guarantee of support in every agent host. No global settings or
owner instructions are modified by ctx.

## Activate it in Codex

Codex discovers repo-local skills under `.agents/skills/` and loads full
instructions when a skill is selected. It supports explicit invocation and
matching a task to the skill description. See the
[official skill documentation](https://learn.chatgpt.com/docs/build-skills).

From the target repository root, copy the complete bundle into a new skill
directory. Inspect an existing installation before updating it; this guarded
example refuses to merge with an existing directory or symlink:

```bash
mkdir -p .agents/skills
test ! -e .agents/skills/ctx &&
  test ! -L .agents/skills/ctx &&
  cp -R /path/to/ctx/skills/ctx .agents/skills/ctx
```

Use an explicit prompt initially:

> Use $ctx to initialize and populate this repository's context from source
> and tests. Use team mode and the default documents. Do not stage or commit.

For an existing scaffold, the task can be narrower:

> Use $ctx and the existing .agent context to explain cancellation behavior.
> Verify the relevant evidence. Do not change any files.

Check that the skill is visible in the host's selector and that a run actually
reads `SKILL.md`. Do not assume discovery from an arbitrary `.ctx/skills/`
location or a file name alone. For other hosts, use their supported skill
loading mechanism and explicitly test activation; this repository does not
claim those integrations have been exercised.

Automatic matching remains allowed, but is not a substitute for testing.
If the repository owner wants a durable activation reminder, they can add a
short pointer in their canonical instructions. The skill must not add or
rewrite those instructions itself.

## Scope and trust

Activation does not authorize a new init, add-on installation, mode conversion,
staging, or commits. Reading/review tasks remain read-only. For authorized
maintenance, the skill distinguishes durable facts from local checkpoints and
checks the selected custom folder consistently.

Verified metadata refers to a real source commit and listed paths. If relevant
source changes are uncommitted, affected facts remain draft even after checking
the new behavior. The agent reports the pending commit rather than creating one
or stamping an old HEAD solely to make status pass.

The skill treats doctor as structural checking, status as evidence/readiness
checking, and update as managed-guidance refresh. None is an independent
certification of factual truth.

## Try a separate booking demo

The [booking fixture](../testdata/booking-demo/README.md) is a small standalone
Go library with capacity limits, confirmed bookings, idempotent cancellation,
and tests. It has no third-party packages or external service dependencies.

From the ctx checkout root, prepare a fresh isolated Git repository:

```bash
ctx_demo_root=$(mktemp -d)
go build -o "$ctx_demo_root/ctx" ./cmd/ctx
node tools/prepare-skill-demo.mjs \
  --destination "$ctx_demo_root/booking" --with-skill
```

The helper resolves an existing parent, refuses an existing destination, copies
the fixture and optional skill, and records an initial **demo-only** Git commit.
It does not initialize context, install globally, invoke an agent, or contact
a remote. The printed paths identify the new workspace and skill.

Point a fresh agent at that workspace and the built ctx executable:

> Use the installed ctx skill. Initialize team context in the custom folder
> .agent, populate default documents from the current source, tests, and owner
> requirements, and record a useful local checkpoint. Do not change application
> files, stage, commit, publish, or install globally. Run relevant checks and
> report unresolved draft evidence honestly.

The following checks inspect the result; the helper itself does not populate it:

```bash
"$ctx_demo_root/ctx" doctor "$ctx_demo_root/booking" --folder .agent
"$ctx_demo_root/ctx" status "$ctx_demo_root/booking" --folder .agent
git -C "$ctx_demo_root/booking" status --short
```

The active continuation should be ignored. Read the generated facts to assess
accuracy and routing; a green status is not enough. Do not copy this demo's
business rules into another project's context.

## Evaluate behavior, not wording

Use isolated workspaces and fresh agent context for each case. The
[evaluation protocol](../evals/ctx-skill/README.md) covers:

- population without a skill as a baseline;
- the same task with explicit skill use;
- description-only skill selection;
- a narrow, read-only explanation using existing context;
- reconciliation after an uncommitted business-rule change.

Inspect artifacts and relevant command results for accuracy, honest metadata,
appropriate routing, privacy, and scope preservation. Treat each trial as a
small forward test, not a statistical reliability claim or proof that every
agent/model behaves the same.

The [first recorded run](../evals/ctx-skill/results.md) passed population,
read-only, and uncommitted-evidence checks. The unassisted baseline also passed;
the run does not establish a skill advantage.

Native host discovery and metadata-based selection are different checks. A
subagent supplied with skill metadata can test its routing choice, but cannot
by itself prove that a particular desktop/CLI host scanned the right location.
