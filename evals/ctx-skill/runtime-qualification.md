# Guarded authoring runtime qualification

Local qualification on October 27, 2026. Historical checks below are preserved;
the follow-up section records the newer qualification. This is a runtime/configuration check,
not a model evaluation, provider-access guarantee, or approval to incur usage.

## Runtime boundary checks

- Codex CLI 0.154.0: a real ephemeral app-server thread used disabled environment
  access and client-handled guarded tools against a scripted loopback responder.
  The actual tool catalog contained the five context tools plus native skill/
  user-input bookkeeping. Native skill inventory was empty. The skill-list
  fixture reached its real handler; forged peer skill reads and unadvertised
  command/delegation attempts did not access the peer canary or write a file.
- Pi 1.1.0 CLI: the real executable loaded only the five guarded tools in a
  fresh private configuration, without a prompt or provider request. Live CLI
  execution is refused: startup-hook errors can be logged while the CLI still
  starts a model turn. Model-free CLI startup is not provider qualification.
- Pi 1.1.0 SDK: the same engine and extension initialized an in-memory session
  with a single-provider read-only credential store and synthetic OAuth data.
  No inference was requested and the original synthetic auth file was unchanged.
- Independent review found silent Pi thinking-level clamping and a native Codex
  timeout that could be mislabeled complete during termination grace. Explicit
  Pi settings now refuse mismatches before inference, and unexpected supervisor
  stops remain incomplete. Deterministic regressions cover both. Audit manifests
  also preserve filenames matching JavaScript prototype properties.
- Credential unit tests refused mutation, logout, renewal, command-backed keys,
  credential headers, API-key substitution for the Codex ChatGPT route, and
  non-loopback endpoints for the explicit Ollama daemon route.

The deterministic responder never contacted a real model. Supplying an explicit
model name in that fixture does not constitute a successful run of that model.

After the review fixes, the complete Node suite with both installed runtimes
enabled passed all 40 tests with no failures or skips. Root Go vet, the full ctx
race suite, both demo race suites, and settings demo Go vet also passed. Ordinary
CI does not require those native runtimes and skips their optional checks.

## Existing local routes

| Selected model | Transport | Local result | Still unproven |
|---|---|---|---|
| GPT-6.1 Sol, `gpt-6.1-sol` | Native Codex app-server; existing file-backed ChatGPT login | Version and local access-token validity passed; `high` requested, no turn started | Remote model access, effective reasoning setting, and inference |
| Claude Opus 5.5, `claude-opus-5-5` | Pi SDK; existing Pi OAuth login | Passed after the user renewed the existing login; exact local model/tools passed; requested `high` equals effective local `high` | Remote provider acceptance, applied reasoning behavior, and inference |
| GLM 5.3, `glm-5.3:cloud` | Pi SDK through existing loopback Ollama Cloud daemon | Registry/model/tools passed with explicit local `off`; explicit `high` is refused; Ollama 0.40.6 responded and `ollama show` returned metadata | Reasoning configuration, cloud sign-in acceptance, and inference |

The initial Claude expiry and missing Ollama auth-store entry were preserved as
diagnostic observations, not model-trial failures. Ollama's missing login-store
entry was expected for its daemon-mediated route. GLM's absence from the daemon's
local tag/running lists did not establish that cloud metadata was unavailable;
the explicit read-only show request succeeded. No model pull or inference was
performed. No keys were created, credential files copied, original login files changed
by the harness, provider routes replaced, or global tools/settings upgraded.

The stricter thinking check exposed missing reasoning metadata in the existing
GLM registry entry. Pi currently resolves that entry to local `off` and clamps
`high` to `off`; the runner now refuses the latter. The registry was not changed.
This is a configuration gate to resolve explicitly before a reasoning-enabled
comparison, not a model failure or evidence that GLM lacks thinking capability.
Local `off` alone also does not prove that Ollama disables provider-side thinking.

These are point-in-time local results. Before launch, repeat preflight with a
validity window covering the approved trial plus five minutes. Freeze the source
commit, input/skill/harness hashes, transport selections, thinking settings, and
agreed resource/spending limits. Explicitly approve provider execution separately.
Retain real access failures, timeouts, and incomplete attempts without semantic
retries or model substitution. At that initial checkpoint, all three selected
models remained unevaluated by this guarded runner.

## Follow-up after the first private pilot

The first pilot is retained separately, not overwritten or reclassified as a
successful cross-model comparison. It exposed two harness defects: an
authentication source module was mistaken for a credential file, and sparse
custom Pi metadata passed startup despite missing output-limit/cost fields.
The latter was reproduced offline as a nonfinite token limit and an accounting
exception; that does not prove it was the sole cause of the sanitized live error.

The broker now retains authentication source filenames, keeps secret-file and
sensitive-ancestor exclusions, and explicitly describes its inventory as a
filtered available view. Environment examples remain withheld. Pi refuses
incomplete/invalid request and accounting metadata before inference. A real Pi
SDK loopback test checks the emitted output limit and reasoning setting, then
processes scripted usage successfully. No actual provider is used by that test.

The user upgraded Codex CLI from 0.154.0 to 0.164.0. A source-free authenticated
catalog inspection with the same existing ChatGPT login now advertises the exact
`gpt-6.1-sol` identifier and `high`; the older client's catalog did not. No alias,
other model, new credential, or billing route was selected. Catalog acceptance is
not proof that inference will succeed. Native runs now gate thread creation on
the exact advertised model and requested reasoning effort.

The newer client's tool catalog changed to a code-mode wrapper. Installed-runtime
tests were updated to exercise the actual nested guarded read, missing Node/
filesystem/network globals, refused native patch writes, and native MCP-call
quarantine with no peer canary exposure. The wrapper host is enabled because
disabling it leaves advertised guarded tools unusable. Execution environments
and workspace roots remain empty; shell, discovery, delegation, apps, and
plugins remain disabled. See the runner contract for the exact runtime
differences. The adapter now pins 0.164.0 rather than silently accepting upgrades.

After these follow-up changes, the complete Node suite passed 46 tests with no
failures or skips, including all four installed-runtime checks. Root Go vet and
the full ctx race suite also passed. These checks used no real model inference.

Reruns require a new private destination and a newly frozen contract. Preserve
the original pilot, include all new failures, and keep application source,
private proposals, and raw provider diagnostics out of the public tooling PR.
