# Guarded context authoring runner

`tools/run-guarded-authoring.mjs` prepares a pinned, private authoring trial.
Preparation and replay do not call a model. Ordinary CI runs only deterministic
boundary tests, never provider trials. Existing retrieval results and their
historical runner remain unchanged.

## Current readiness

See the [point-in-time runtime qualification results](runtime-qualification.md)
for the installed versions and existing local authentication routes checked.

The model-neutral broker, replay runner, Pi SDK adapter, native Codex
app-server adapter, and disposable CLI verification are implemented. Native
runtime qualification uses synthetic credentials and a local scripted Responses
responder, not a model. This is **not clearance to launch the complete cross-model
experiment**. Before provider calls:

- Validate each explicit model identifier, account access, credential route,
  runtime version, and agreed spending/time budgets. No substitution is automatic.
- Run local runtime preflight with the explicit existing credential file.
  File-backed OAuth must remain valid for the trial plus five minutes; refresh,
  login, keychain extraction, and credential helpers are not performed here.
  Native Codex accepts the existing ChatGPT login route, never an implicit
  separately billed API-key substitute. Pi SDK reuses a single selected stored
  credential through a read-only store. No credential file is copied.
- Run the optional installed-runtime qualification tests against the actual
  approved executables. Current pins are Pi 1.1.0 and Codex CLI 0.164.0. Other
  versions are refused, not upgraded.
- Confirm the explicit Pi transport in the trial plan. `pi-sdk` uses
  the Pi engine/guard with read-only existing credentials. `pi` is accepted for
  model-free preparation/replay only; live CLI execution is refused before
  claiming a destination. Pi CLI startup-hook errors alone do not guarantee
  zero inference, so it is not an eligible provider adapter. No transport,
  credential source, model, or billing route is changed automatically.
- Review the frozen file inventory for sensitive material not recognizable by
  filename. No filename filter can certify that arbitrary source is secret-free.

The OpenAI Docs skill informed the native-runtime design. The installed
[Codex app-server](https://learn.chatgpt.com/docs/app-server) protocol defines an
empty `environments` list as disabling environment access; both thread and turn
requests explicitly use it. Filesystem permission profiles or
[tool hooks](https://learn.chatgpt.com/docs/hooks) alone are not the complete
boundary. A Git worktree, a prompt, and a post-run diff are not a security sandbox.

## Enforcement and trust boundary

The trusted coordinator reads a clean repository at an explicit full HEAD commit.
It reads tracked regular UTF-8 blobs from Git, not agent-selected host paths.
Ignored and untracked files are unavailable to tools. Symlinks, submodules,
binary/large files, common secret filenames, and runtime/auth directories are
omitted. The selected ctx skill is overlaid without changing owner instructions.
A conflicting installed ctx skill is refused rather than silently replaced.
Source filenames such as `auth.ts` are retained; credential files such as
`auth.json` and sensitive ancestors remain excluded. Environment examples are
still withheld: an example suffix is not proof that contents contain no secrets.
`ctx_source_info` names the list `availableSourcePaths`, explicitly marks it
incomplete, and reports omission counts by reason without sensitive filenames.
It is never evidence that an unavailable path is absent from the repository.

The coordinator initializes a genuine team scaffold in a private verification
repository. The author sees this scaffold and source through an immutable
virtual file view. Neither the source checkout nor the verification directory
is its working directory. The original source repository is never initialized,
staged, committed, switched, or populated by this runner.

The authoring capabilities in both runtimes are:

| Tool | Capability |
|---|---|
| `ctx_read` | Read bounded line ranges from this trial's virtual files |
| `ctx_list` | Discover virtual repository-relative paths |
| `ctx_search` | Bounded literal search; no regular expression or shell |
| `ctx_source_info` | Obtain the immutable source commit and available source paths |
| `ctx_write` | Propose context text with a matching preimage hash |

Writes stay in memory until the coordinator freezes the attempt. Authors can
edit `.ctx/INDEX.md`, `.ctx/local/CONTINUE.md`, and Markdown facts/topics under
the five default categories. Config, sharing mode, ignore rules, managed README,
owner instructions, application files, and uninstalled add-ons cannot be edited.
There is no authorized deletion, filesystem link, shell, Git, network, publication, or
delegation tool. Tool arguments have a strict schema. All path forms are
canonical relative paths; parent traversal, absolute paths, backslashes, control
characters, and Git internals are rejected. Virtual paths never resolve symlinks
or reach the host filesystem. Full skill, root owner, router, and continuation
reads are prerequisites for writes; this observes access, not comprehension.
The broker also accepts a coordinator-selected `readOnly: true` input for fresh
reader comparisons; it refuses all context writes regardless of preimage or
orientation. Any attempted reader write must be reported as a task violation.
This does not change the authoring CLI plan or give the model mode-selection power.

Tool-call, returned UTF-8 byte, cumulative write-byte, and elapsed-time budgets
are enforced in the broker, synchronously even when Pi schedules parallel calls.
The extension aborts the turn when a hard budget is reached. A process-group
supervisor adds a wall-clock cap, bounded event framing/stream bytes, and
termination escalation for descendants. Windows provider execution is refused
until equivalent process containment exists.

Explicit Pi thinking settings must match the engine's effective setting before
inference; silent clamping is refused. Preflight and Pi results record the
effective local setting. A null setting is unspecified, not a claim of equivalent
reasoning across runtimes. Local engine settings do not prove what a remote
provider actually applies. Supervisor timeouts, malformed events, stream limits,
and abnormal process failures remain incomplete even if a final message and
completion arrive during termination grace.
Pi extension registration does not apply the CLI JSON loader's model defaults.
Custom profiles must explicitly provide valid context/output limits, reasoning,
input types, and cost rates; sparse or invalid metadata is refused before
inference, not guessed. Native catalog models are also validated after selection.
Use a separate reviewed trial profile instead of editing a user's registry.
Zero cost metadata can be an SDK accounting placeholder, not evidence that
inference is free. The runner reports token usage, not a pricing estimate.

The Pi/Codex runtimes, extension, coordinator, selected CLI binaries, operating system,
and Git are trusted software. This is a model-tool capability boundary, **not an
OS sandbox against a compromised runtime process or malicious installed executable**.
The runtime processes necessarily have provider transport access; model tools do not.
No project scripts, hooks, tests, shell commands, or extensions are executed by
the author. Native discovery, saved sessions, MCP, built-in tools, global
extensions, context files, and prompt templates are disabled. A fresh private
agent directory and explicit system prompt avoid inherited settings and helper
commands. Unexpected registered tools block startup and all tool execution.

Native Codex uses fresh process-local state, an in-memory thread, no environment
or workspace capability roots, disabled command/image/browser/app/plugin/memory/
delegation features, and the same five tools supplied as client-handled dynamic
tools. Codex 0.164.0 advertises these through its isolated JavaScript code-mode
wrapper, even with the optional code-mode feature disabled. The wrapper host
must be enabled for guarded nested calls to execute. It has no Node, filesystem,
or network globals; this is runtime orchestration, not an added source tool.
Its nested catalog also advertises native patch, MCP-resource, and clock helpers,
plus user-input bookkeeping. With no execution environments, native patch writes
are refused; no MCP servers are configured. An observed native MCP call
quarantines the run. Tests exercise real nested guarded reads, disabled filesystem
imports, missing process/network globals, refused patch writes, peer-resource
reads, and unadvertised command/delegation calls. Native skill discovery is
disabled and its inventory must be empty before starting a thread.
User-input, permission, token-refresh, and other unexpected control requests are
denied by the client and stop the trial. These known native declarations are
recorded runtime differences, not equivalent extra authoring capabilities.
Authenticated native runs check the entire paginated `model/list` catalog for
the exact model and requested effort before creating a thread. A display name,
picker identifier, or another model is not an automatic alias or fallback.

Codex authentication uses experimental externally supplied ChatGPT access tokens
with ephemeral credential storage. The coordinator supplies only the existing
valid access token/account identifier and owns the stop-on-refresh policy; it
does not import refresh tokens, renew the login, or modify its source file. A
renewal request ends the attempt rather than changing credentials or retrying it.
Native diagnostics/databases in the exclusively created runtime directory are
removed after execution; private operational results and proposals are retained.

Concurrent hostile host processes replacing coordinator-owned directories are
outside this threat model. Use exclusively controlled private destinations;
do not treat ordinary same-user directory permissions as hostile-host isolation.

## Model-free preparation and replay

Build a trusted ctx binary, then write a private plan with exactly these fields.
Paths and the source commit below are placeholders, not runnable defaults:

```json
{
  "repository": "/absolute/clean/source-checkout",
  "sourceCommit": "full-immutable-commit-hash",
  "ctx": "/absolute/trusted/ctx",
  "runtime": {
    "kind": "pi-sdk",
    "executable": "/absolute/approved/pi",
    "provider": "anthropic",
    "model": "explicit-approved-model-identifier",
    "thinking": "high",
    "auth": { "kind": "native-file", "path": "/absolute/existing/auth.json" }
  },
  "budgets": {
    "timeoutMs": 900000,
    "maxCalls": 200,
    "maxOutputBytes": 4194304,
    "maxWriteBytes": 1048576
  }
}
```

These are example resource ceilings, not an approved provider spending budget.
Preparation is the default and never starts Pi:

```bash
node tools/run-guarded-authoring.mjs \
  --plan /absolute/private/plan.json \
  --destination /absolute/private/new-preparation
```

For deterministic replay, add `--replay /absolute/private/operations.json`.
Its data-only shape is `{"operations":[{"tool":"ctx_list","input":{"path":"."}}],"final":"Replay summary"}`.
It is not an executable command file. Writes require the same observed-entrypoint
and preimage checks as live tools. Refused calls and partial proposals remain in
the result; they are not replaced by a later, favorable attempt.

Destinations must be absent, with an existing parent, outside the source checkout.
Existing scaffolds, dirty/index changes, wrong HEAD, and oversized source/audit
trees are refused. Use a clean dedicated checkout rather than an installed
dependency tree. Preparation preserves the target's shared Git state and linked
worktree index. A source ignore rule hiding `.ctx` correctly makes team init fail.

Provider execution additionally requires `--allow-provider-run`, an eligible
adapter, pinned runtime, and approved credentials. This flag is never supplied
by CI and does not bypass any boundary or readiness requirement above. There is
no native CLI fallback and no implicit resume, semantic retry, or model change.

### Reusing existing authentication

Select `runtime.kind: "pi-sdk"` or `"codex"` explicitly and add:

```json
"auth": { "kind": "native-file", "path": "/absolute/existing/auth.json" }
```

For an existing Pi custom registry, `runtime.modelsFile` may name its absolute
path. Only the exact selected provider/model is imported into memory. Credential
headers, command-backed keys, discovery hooks, and model fallbacks are refused;
unrelated provider settings are not loaded. Static client/version headers and
model compatibility settings can be preserved. No registry or auth file is copied
to the artifact directory. A configured resolver is a gate to resolve explicitly,
not a reason to acquire a new API key or change billing.

For Ollama Cloud already mediated by a local daemon, use `pi-sdk`, provider
`ollama`, the explicit existing registry, and `"auth": {"kind":"local-daemon"}`.
Both provider and model endpoints must stay on HTTP loopback. The native
registry's public `ollama` placeholder is supported only in this explicit route;
it is not a cloud token or a newly created key. Real tokens, helpers, or a
non-loopback endpoint are refused. The daemon retains responsibility for its
existing cloud sign-in. Do not infer model absence from a missing local-list
entry alone; `ollama show <exact-model>` can check cloud model metadata without
inference, pulling, or changing the registry.

Local-only preflight never starts a model turn:

```bash
node tools/preflight-authoring-runtime.mjs \
  --runtime pi-sdk \
  --executable /absolute/approved/pi \
  --provider anthropic \
  --model explicit-approved-model-identifier \
  --thinking high \
  --auth-file /absolute/existing/pi/auth.json \
  --validity-ms 1200000
```

Add `--models-file /absolute/existing/models.json` only when that registry is
part of the approved route. For Codex, use `--runtime codex`, the native Codex
executable, `--provider openai`, and its existing file-backed ChatGPT login.
Preflight checks local version, credential validity, and Pi's exact local model/
tool/thinking configuration. Unsupported explicit thinking levels are refused;
omitting `--thinking` records an unspecified request and Pi's actual local level.
Codex preflight starts no turn, so it cannot observe an effective reasoning level.
Preflight cannot prove that a remote provider accepts the credential
or grants access to the model; outputs explicitly say `providerAccess: not-tested`.
Do not paste tokens or manually copy credentials into the plan.

For the Ollama daemon preflight, replace `--auth-file` with
`--auth-kind local-daemon` and provide its existing `--models-file`.

## Artifacts, checks, and later review

The destination is private (directory permissions restricted at creation):

- `manifest.json`: pinned commit, source and skill hashes, ctx binary/version,
  harness and explicit model-registry hashes, Node/platform, explicit runtime
  selection, budgets, prompt, omissions, and initial audit.
- `input.json`: private frozen source, skill, and generated scaffold.
- `attempt.json`: frozen virtual edits, allowlisted tool activity, returned byte
  counts, observed complete reads, and final public text.
- `candidate/.ctx/`: validated proposed scaffold, including ignored continuation.
- `result.json`: runtime failure/incomplete status, selected numeric usage,
  Pi's effective local thinking setting, before/after audit, and separate
  `ctx doctor` / `ctx status` results.
- `verification/`: private disposable Git/source copy for coordinator checks.

The verification Git database contains the pinned source commit, including blobs
omitted from the model view. Artifacts can contain private source and context;
do not publish this directory or commit it into ctx or the target repository.
No native sessions, hidden reasoning, raw stderr/provider errors, credentials,
or system/developer event messages are retained. The harness's explicit system
prompt is recorded as configuration; private native messages are not collected.
Final public text remains private source-derived data, not automatically safe
for public distribution.

Runtime version pins are qualification checks, not immutable build fingerprints.
The manifest hashes the harness, ctx binary, input, and explicit model registry;
it does not fingerprint the complete installed runtime/SDK package or native
model catalog. Record those additional build/configuration fingerprints before
using trials as a strict reproducibility or model-ranking benchmark.

Audits include ignored working files, content hashes, modes, symlink targets,
HEAD, refs, staged diff, index, Git config and exclude state. Git resolves metadata
locations so linked worktrees work correctly. Audits detect persistent changes;
they cannot prove that no transient action happened. The capability boundary,
not an after-the-fact audit, restricts model actions. Changed source/Git state
quarantines the proposal, as does a changed explicit model registry. Source or audit errors preserve the destination for
inspection; nothing is automatically deleted or retried.

Doctor and status are mechanical checks, not prose verification. Draft readiness
is retained, not treated as an execution failure or rewritten until green.
Application tests are not invoked by this authoring runner: Moonlight's check
suite needs a separate coordinator step with isolated PostgreSQL and no provider
credentials. No branches, commits, PRs, or merges are created here. Later model
access checks, semantic evaluation, independent final-candidate review, human
review, and publication remain explicit separate steps.

## Boundary tests

```bash
node --test tools/guarded-authoring.test.mjs
node --test tools/authoring-runtime.test.mjs
```

The ordinary suite uses data-only replay, fake process output, and a real built
ctx CLI. It tests traversal/peer/Git denial, immutable reads, secret and link
omission, protected writes, evidence-entrypoint reads, preimage conflicts,
hard simultaneous-call/byte/time caps, forged proposals, worktree metadata and
ignored state, environment hygiene, log filtering, process failures, and genuine
scaffold preparation/checks. These are enforcement tests, not model evaluations.

Optional installed-Pi startup check (no prompt and no provider request):

```bash
CTX_TEST_PI=/absolute/approved/pi node --test tools/guarded-authoring.test.mjs
```

Do not equate this startup test with model availability, authentication,
provider behavior, or a successful authoring trial.

Native qualification (the Codex case needs permission to bind loopback):

```bash
CTX_TEST_PI=/absolute/approved/pi \
CTX_TEST_CODEX=/absolute/approved/codex \
node --test tools/authoring-runtime.test.mjs
```

The optional native Codex test routes its explicit model name exclusively to a
local deterministic responder with no real credentials. It inspects the actual
tool catalog and exercises guarded reads plus attempted peer reads, commands,
and delegation. It is not an evaluation of that model. Pi SDK tests use synthetic
credentials and initialize the real engine without requesting inference. Ordinary
CI runs the pure configuration/credential tests and skips optional native tests
when those executables are not explicitly provided.
