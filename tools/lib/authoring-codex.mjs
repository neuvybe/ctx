import { createBoundary, toolNames } from "./authoring-boundary.mjs";
import { runProcess } from "./authoring-process.mjs";
import { descriptions, schemas } from "../authoring-pi-extension.mjs";

export const codexVersion = "0.154.0";
export function codexConfig({ model, provider = "openai", fixtureBaseUrl }) {
  if (fixtureBaseUrl && !/^http:\/\/127\.0\.0\.1:\d+$/.test(fixtureBaseUrl)) throw new Error("Only loopback fixture endpoints are supported");
  return `model = ${JSON.stringify(model)}
model_provider = ${JSON.stringify(provider)}
approval_policy = "never"
sandbox_mode = "read-only"
cli_auth_credentials_store = "ephemeral"
project_doc_max_bytes = 0
web_search = "disabled"
check_for_update_on_startup = false
[agents]
enabled = false
[skills]
include_instructions = false
[skills.bundled]
enabled = false
[features]
shell_tool = false
shell_snapshot = false
view_image = false
apps = false
plugins = false
memories = false
multi_agent = false
multi_agent_v2 = false
hooks = false
browser_use = false
browser_use_external = false
computer_use = false
image_generation = false
artifact = false
goals = false
sleep_tool = false
workspace_dependencies = false
request_permissions_tool = false
skill_search = false
skill_mcp_dependency_install = false
skip_host_skill_discovery = true
code_mode = false
code_mode_host = false
enable_request_compression = false
remote_compaction_v2 = false
unbounded_connection_retries = false
[analytics]
enabled = false
${fixtureBaseUrl ? `[model_providers.${provider}]
name = "Deterministic loopback fixture, not a model provider"
base_url = ${JSON.stringify(fixtureBaseUrl)}
wire_api = "responses"
requires_openai_auth = false
request_max_retries = 0
stream_max_retries = 0
supports_websockets = false
` : ""}`;
}

export function dynamicTools() {
  return toolNames.map(name => ({ type: "function", name, description: descriptions[name], inputSchema: { type: "object", properties: schemas[name], required: Object.keys(schemas[name]).filter(key => !["offset", "limit"].includes(key)), additionalProperties: false } }));
}

export function codexThreadParams({ model, provider, cwd, systemPrompt }) {
  return {
    model, modelProvider: provider, cwd, ephemeral: true,
    allowProviderModelFallback: false, approvalPolicy: "never", sandbox: "read-only",
    // The installed protocol explicitly defines [] as disabling environment
    // access. Do not replace this with an omitted/default environment.
    environments: [], runtimeWorkspaceRoots: [], selectedCapabilityRoots: [],
    baseInstructions: systemPrompt, dynamicTools: dynamicTools(),
    experimentalRawEvents: false,
  };
}

export async function runCodex({ executable, cwd, env, input, model, provider, thinking, systemPrompt, prompt, auth, onProposal = () => {} }) {
  const boundary = createBoundary(input);
  const pending = new Map();
  let sequence = 0, send, stop, threadId, finished = false, final = "", violation = false;
  const usage = [];
  const request = (method, params) => new Promise((resolve, reject) => {
    const id = ++sequence;
    pending.set(id, { resolve, reject });
    send({ id, method, params });
  });
  function record(message) {
    if (Object.hasOwn(message, "id") && !message.method) {
      const waiting = pending.get(message.id);
      if (!waiting) throw new Error("Unexpected control response");
      pending.delete(message.id);
      if (message.error) waiting.reject(new Error("Codex control request failed")); else waiting.resolve(message.result);
      return;
    }
    if (Object.hasOwn(message, "id")) {
      if (message.method !== "item/tool/call" || message.params?.threadId !== threadId || message.params?.namespace) {
        send({ id: message.id, error: { code: -32601, message: "This capability is not authorized" } });
        violation = true;
        stop("unauthorized-control-request");
        return;
      }
      let text, success = true;
      try { text = boundary.execute(message.params.tool, message.params.arguments); }
      catch { text = "Guarded authoring request refused"; success = false; }
      onProposal(boundary.snapshot());
      send({ id: message.id, result: { success, contentItems: [{ type: "inputText", text }] } });
      if (boundary.exhausted()) stop("authoring-budget");
      return;
    }
    if (message.method === "item/completed" && message.params?.threadId === threadId) {
      const item = message.params.item;
      if (item?.type === "agentMessage" && ["final_answer", null, undefined].includes(item.phase) && typeof item.text === "string") final = item.text.slice(0, 65536);
      if (["commandExecution", "fileChange", "mcpToolCall", "webSearch", "collabAgentToolCall", "imageGeneration"].includes(item?.type)) {
        violation = true;
        stop("unexpected-native-tool");
      }
    }
    if (message.method === "thread/tokenUsage/updated") {
      const total = message.params?.tokenUsage?.total;
      if (total) usage.push(Object.fromEntries(["totalTokens", "inputTokens", "cachedInputTokens", "outputTokens", "reasoningOutputTokens"].filter(key => Number.isFinite(total[key]) && total[key] >= 0).map(key => [key, total[key]])));
    }
    if (message.method === "turn/completed" && message.params?.threadId === threadId) {
      finished = message.params.turn?.status === "completed";
      stop(finished ? "completed" : "turn-failed");
    }
  }
  const runtime = await runProcess({
    executable, args: ["app-server", "--strict-config", "--listen", "stdio://"], cwd, env,
    timeoutMs: input.budgets.timeoutMs, onRecord: record,
    async onStart(write, interrupt) {
      send = write; stop = interrupt;
      await request("initialize", { clientInfo: { name: "ctx_guarded_authoring", version: "1.0.0" }, capabilities: { experimentalApi: true } });
      send({ method: "initialized", params: {} });
      const inventory = await request("skills/list", { cwds: [cwd], forceReload: true });
      if (!Array.isArray(inventory.data) || inventory.data.some(entry => entry.skills?.length || entry.errors?.length)) throw new Error("Native skill inventory is not empty");
      if (auth) await request("account/login/start", auth);
      const started = await request("thread/start", codexThreadParams({ model, provider, cwd, systemPrompt }));
      if (started.model !== model || started.modelProvider !== provider || started.thread?.ephemeral !== true || started.instructionSources?.length) throw new Error("Codex selected unexpected model or instructions");
      threadId = started.thread.id;
      onProposal(boundary.snapshot());
      await request("turn/start", { threadId, environments: [], input: [{ type: "text", text: prompt }], ...(thinking ? { effort: thinking } : {}) });
    },
  });
  for (const waiting of pending.values()) waiting.reject(new Error("Codex process closed"));
  boundary.close();
  const proposal = boundary.snapshot();
  onProposal(proposal);
  const expectedExit = runtime.code === 0 || (runtime.code === null && ["SIGTERM", "SIGKILL"].includes(runtime.signal));
  const incomplete = !finished || violation || !final || runtime.spawnError || runtime.interrupted !== "completed" || !expectedExit;
  return { runtime: { ...runtime, mode: "provider", version: codexVersion, incomplete, usage, messages: [] }, proposal, final };
}
