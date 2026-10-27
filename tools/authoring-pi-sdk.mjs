// Optional Pi SDK transport: same Pi engine and guarded extension as the CLI,
// with a single-provider, read-only native credential store. No auth file copies.
import { readFile, realpath } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import authoringExtension, { qualifiedThinking } from "./authoring-pi-extension.mjs";
import { readOnlyCredentialStore, selectedPiCredential } from "./lib/authoring-auth.mjs";
import { toolNames } from "./lib/authoring-boundary.mjs";

// registerProvider takes extension definitions, not CLI JSON definitions. It
// does not supply the CLI loader's defaults. Refuse incomplete request/cost
// metadata rather than letting NaN token limits or a cost-calculation crash
// reach inference. Trial-local profiles can supply explicit, reviewed values.
export function validatePiModel(model) {
  const rateNames = ["input", "output", "cacheRead", "cacheWrite"];
  const validRates = rates => rates && typeof rates === "object" && !Array.isArray(rates) && rateNames.every(key => Number.isFinite(rates[key]) && rates[key] >= 0);
  if (!model || typeof model.name !== "string" || !model.name || typeof model.api !== "string" || !model.api ||
      typeof model.baseUrl !== "string" || !model.baseUrl || typeof model.reasoning !== "boolean" ||
      !Array.isArray(model.input) || !model.input.includes("text") || model.input.some(value => !["text", "image"].includes(value)) ||
      !Number.isSafeInteger(model.contextWindow) || model.contextWindow < 1 || !Number.isSafeInteger(model.maxTokens) || model.maxTokens < 1 || model.maxTokens > model.contextWindow ||
      !validRates(model.cost) || (model.cost.tiers !== undefined && (!Array.isArray(model.cost.tiers) || model.cost.tiers.some(tier => !validRates(tier) || !Number.isSafeInteger(tier.inputTokensAbove) || tier.inputTokensAbove < 0)))) {
    throw new Error("Selected Pi model requires complete valid request limits, input, reasoning, and cost metadata");
  }
  return model;
}

export async function selectedModelConfig(path, provider, model, runtime, { localDaemon = false } = {}) {
  const raw = await readFile(path, "utf8");
  if (Buffer.byteLength(raw) > 1024 * 1024) throw new Error("Model registry is too large");
  let config;
  try { config = JSON.parse(raw)?.providers?.[provider]; } catch { throw new Error("Model registry is invalid; private details omitted"); }
  if (!config) {
    if (runtime.getModel(provider, model)) return null;
    throw new Error("Exact model is absent from this isolated local catalog; no fallback selected");
  }
  const placeholder = localDaemon && provider === "ollama" && config.apiKey === "ollama";
  if (typeof config !== "object" || (config.apiKey && !placeholder) || config.oauth || config.refreshModels) throw new Error("Selected registry requires an unsupported credential or discovery resolver");
  if (Object.keys(config).some(key => !["baseUrl", "api", "headers", "authHeader", "models", "modelOverrides", ...(placeholder ? ["apiKey"] : [])].includes(key))) throw new Error("Unsupported provider registry field");
  const defined = config.models?.find(entry => entry.id === model) || runtime.getModel(provider, model);
  if (!defined) throw new Error("Exact configured model is missing; no fallback selected");
  const selected = { ...defined, ...config.modelOverrides?.[model], id: model };
  const validateHeaders = headers => {
    if (!headers) return;
    if (Object.entries(headers).some(([name, value]) => /authorization|api.key|cookie|token/i.test(name) || typeof value !== "string" || value.startsWith("!"))) throw new Error("Credential headers and command helpers are not imported");
  };
  validateHeaders(config.headers); validateHeaders(selected.headers);
  for (const value of [config.baseUrl, selected.baseUrl].filter(Boolean)) {
    const url = new URL(value);
    if (url.username || url.password || url.search || url.hash || (url.protocol !== "https:" && !(url.protocol === "http:" && ["127.0.0.1", "localhost", "[::1]"].includes(url.hostname)))) throw new Error("Unsafe model registry endpoint");
    if (localDaemon && (url.protocol !== "http:" || !["127.0.0.1", "localhost", "[::1]"].includes(url.hostname))) throw new Error("Ollama daemon transport must remain on loopback");
  }
  if (localDaemon) {
    const endpoint = new URL(selected.baseUrl || config.baseUrl);
    if (endpoint.protocol !== "http:" || !["127.0.0.1", "localhost", "[::1]"].includes(endpoint.hostname)) throw new Error("Ollama daemon transport must remain on loopback");
  }
  // Disable provider-level model fallbacks rather than changing the user's
  // requested model after a version/access failure.
  if (selected.compat) selected.compat = { ...selected.compat, allowedFallbackModels: [] };
  validatePiModel({ ...selected, api: selected.api ?? config.api, baseUrl: selected.baseUrl ?? config.baseUrl });
  return { ...Object.fromEntries(["baseUrl", "api", "headers", "authHeader"].filter(key => config[key] !== undefined).map(key => [key, config[key]])), ...(localDaemon ? { apiKey: "ollama" } : {}), models: [selected] };
}

export async function loadPi(executable) {
  const path = await realpath(executable);
  // Supported npm installation shape: package/dist/bundle/cli.js.
  const root = resolve(dirname(path), "../..");
  const pkg = JSON.parse(await readFile(join(root, "package.json"), "utf8"));
  if (pkg.name !== "@earendil-works/pi-coding-agent" || pkg.version !== "1.1.0") throw new Error("Unqualified Pi SDK installation");
  return import(pathToFileURL(join(root, "dist/index.js")).href);
}

export async function createIsolatedPi({ pi, cwd, agentDir, model, provider, thinking, credential, systemPrompt, modelsFile, localDaemon = false }) {
  const modelRuntime = await pi.ModelRuntime.create({ credentials: readOnlyCredentialStore(provider, credential), modelsPath: null, modelsStorePath: join(agentDir, "catalog.json"), allowModelNetwork: false, refreshOnCreate: false });
  if (localDaemon && (!modelsFile || provider !== "ollama")) throw new Error("Ollama daemon transport requires its explicit existing model registry");
  if (modelsFile) {
    const config = await selectedModelConfig(modelsFile, provider, model, modelRuntime, { localDaemon });
    if (config) modelRuntime.registerProvider(provider, config);
  }
  let selected = modelRuntime.getModel(provider, model);
  if (!selected) throw new Error("Exact model is absent from this isolated local catalog; no fallback selected");
  validatePiModel(selected);
  selected = { ...selected, compat: { ...selected.compat, allowedFallbackModels: [] } };
  const settingsManager = pi.SettingsManager.inMemory({ retry: { enabled: false, provider: { maxRetries: 0 } }, compaction: { enabled: false }, cacheWarming: "off", enableAnalytics: false, enableInstallTelemetry: false, defaultProjectTrust: "never", packages: [], extensions: [], skills: [], prompts: [], themes: [] });
  const resourceLoader = new pi.DefaultResourceLoader({ cwd, agentDir, settingsManager, noExtensions: true, disabledBuiltinExtensions: ["mcp", "codemode"], noSkills: true, noPromptTemplates: true, noThemes: true, noContextFiles: true, systemPrompt, extensionFactories: [authoringExtension], agentsFilesOverride: () => ({ agentsFiles: [] }), appendSystemPromptOverride: () => [] });
  await resourceLoader.reload();
  const result = await pi.createAgentSession({ cwd, agentDir, modelRuntime, model: selected, thinkingLevel: thinking || "off", noTools: "builtin", tools: toolNames, resourceLoader, settingsManager, sessionManager: pi.SessionManager.inMemory() });
  try { qualifiedThinking(thinking, result.session.thinkingLevel); }
  catch (error) { result.session.dispose(); throw error; }
  let extensionFailed = false;
  await result.session.bindExtensions({ mode: "print", abortHandler: () => { void result.session.abort(); }, onError: () => { extensionFailed = true; } });
  if (extensionFailed || result.modelFallbackMessage || result.session.model?.id !== model || result.session.model?.provider !== provider || JSON.stringify(result.session.getActiveToolNames().sort()) !== JSON.stringify([...toolNames].sort())) {
    result.session.dispose();
    throw new Error("Pi selected an unexpected model or tool loadout");
  }
  return result.session;
}

async function main() {
  const [executable, provider, model, thinking, credentialPath, mode, modelsFile] = process.argv.slice(2);
  if (!executable || !credentialPath || !["preflight", "run"].includes(mode)) throw new Error("Invalid Pi SDK control arguments");
  const input = JSON.parse(await readFile(process.env.CTX_AUTHORING_INPUT, "utf8"));
  const control = JSON.parse(await readFile(process.env.CTX_AUTHORING_CONTROL, "utf8"));
  const localDaemon = credentialPath === "local-daemon";
  const credential = localDaemon ? { type: "api_key", key: "ollama" } : await selectedPiCredential(credentialPath, provider, input.budgets.timeoutMs + 300000);
  const pi = await loadPi(executable);
  const session = await createIsolatedPi({ pi, cwd: process.cwd(), agentDir: process.env.PI_CODING_AGENT_DIR, model, provider, thinking: thinking === "none" ? null : thinking, credential, systemPrompt: control.systemPrompt, modelsFile, localDaemon });
  try {
    if (mode === "preflight") {
      console.log(JSON.stringify({ type: "ctx_preflight", runtime: "pi-sdk", provider, model, credentialType: credential.type, toolNames: session.getActiveToolNames(), inferenceRequested: false }));
      return;
    }
    session.subscribe(event => {
      // The supervisor's public allowlist filters this in memory; never tee raw
      // stdout to disk. Suppress reasoning deltas and non-public message roles.
      if (event.type === "message_end" && event.message?.role === "assistant") {
        const message = { ...event.message, content: event.message.content.filter(part => part.type === "text" || part.type === "toolCall") };
        console.log(JSON.stringify({ type: "message_end", message }));
      }
    });
    await session.prompt(control.prompt);
  } finally { session.dispose(); }
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main().catch(() => { console.error("Isolated Pi initialization or transport failed; private details omitted"); process.exitCode = 1; });
