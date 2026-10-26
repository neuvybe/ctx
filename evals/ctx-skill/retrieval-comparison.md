# Flat versus folder-first retrieval protocol

See the [recorded results](retrieval-comparison-results.md), including provider
failures and best-effort tool-budget overshoot; no failed trial was replaced.

Use eight fresh Pi sessions: GLM 5.3 and Claude Opus 4.8, each with two tasks
and two layouts. Keep the same Pi version, tool set, skill, source bytes, and
factual payloads across each pair. This is a bounded workflow comparison, not
a model ranking or reliability estimate. No semantic retries or model-guided
fixture changes after the first call; retain failures and timeouts.

The manual runner is `tools/run-retrieval-comparison.mjs`; it launches all eight
provider-backed sessions and requires existing Pi credentials. It is not an
ordinary test command and is never executed by CI. Use a new destination:

```bash
node tools/run-retrieval-comparison.mjs \
  --destination /absolute/new/evaluation \
  --ctx /absolute/working-tree/ctx
```

It uses the versioned scope guard and writes only sanitized public activity,
metrics, and local workspace manifests, never native session or reasoning logs.

Prepare an absent destination using `tools/prepare-retrieval-comparison.mjs`
with `--layout flat` or `--layout folder` and an absolute working-tree CLI path.
The fixture is a partial ctx source snapshot, not a full buildable checkout.
Fifteen evaluator-authored topic payloads are combined into flat category
documents or split into topics linked by category indexes. Both inputs pass
doctor and status. Compare semantic/source/skill hashes before calls; only
generated source-commit stamps, layout paths, navigation, document metadata
granularity, and structural template guidance differ. Folder indexes expose
descriptive topic routes while flat documents expose headings; this evaluates
the whole retrieval layout, not the effect of filenames in isolation.

## Tasks

Focused question:

> What precisely does ctx mean by hydration? In a fresh clone of an existing
> team scaffold, what can init create, what does it preserve, and can it change
> installed add-ons? Explain the important refusal boundaries and cite source.

Cross-category question:

> We have a flat layout-v2 team scaffold. A proposed plan is to run ctx update
> to get folder-first files and the latest default add-ons, then treat a
> verified category index as proof that every linked topic is accurate. Is that
> correct? Explain a safe in-place way to add contracts, what doctor and status
> can establish, and how index and topic readiness relate. Cite source.

Give both tasks the same wrapper: authorized repository and installed skill,
owner-instructions/continuation/INDEX entry points, read-only scope, use current
source for material claims, no delegation or access to parents/other trials.
Supply no answer key, discrepancy count, earlier agent output, or preferred
layout. Layout names on disk should not reveal the comparison arm.

## Measurement and acceptance

Use read, grep, find, and ls only; block paths outside the repository and Git
internals. Disable automatic context, skills, extensions, MCP, prompt-template
discovery, and saved native sessions; explicitly load the same local skill and
scope guard. Cap each trial at six minutes and twenty-four tool calls. Alternate
pair order. No extra inference calls to replace failed results.

Record configured provider/model/thinking, final public text, ordinary tool
calls and their returned text-byte counts, failures, elapsed time, source and
skill hashes, and full before/after workspace manifests including ignored state,
HEAD, staged diff, and index hash. Do not retain hidden reasoning, native
sessions, credentials, or system/developer messages. Measure actual UTF-8 bytes
returned by tools, not guessed token counts or self-reported reads. Separate
context bytes from raw-source bytes and total tool output; wall time includes
provider/tool overhead and is not a speed benchmark.

Judge answers against the actual source, including whether hydration preserves
durable files and saved add-ons, explicit selection conflicts are refused,
update does not migrate paths or install defaults, add-on output follows saved
layout, and mechanical checks/index metadata do not certify prose or children.
Inspect introduced errors and unchecked scope, not just desired phrases.
No answer corrections or findings are supplied to readers.

The fixture is small, authored, and based on this checkout; it is not an
independent real-world holdout. One run per model/task/layout cannot establish
general superiority. Read-only guarded tools do not test authoring, unrestricted
tool safety, native discovery, or source-level implementation performance.
