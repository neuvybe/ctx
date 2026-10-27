# Guarded authoring runtime qualification

Local qualification on October 27, 2026. This is a runtime/configuration check,
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
retries or model substitution. All three selected models remain unevaluated by
this guarded runner at this checkpoint.
