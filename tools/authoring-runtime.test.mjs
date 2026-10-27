import assert from "node:assert/strict";
import { createServer } from "node:http";
import { chmod, lstat, mkdtemp, mkdir, readFile, realpath, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { codexConfig, codexThreadParams, dynamicTools, qualifyCodexModel, runCodex } from "./lib/authoring-codex.mjs";
import { coordinatorEnv } from "./lib/authoring-workspace.mjs";
import { toolNames } from "./lib/authoring-boundary.mjs";
import { readOnlyCredentialStore, selectedCodexAuth, selectedPiCredential } from "./lib/authoring-auth.mjs";
import { createIsolatedPi, loadPi, selectedModelConfig, validatePiModel } from "./authoring-pi-sdk.mjs";
import { preflightFailure, preflightRuntime } from "./preflight-authoring-runtime.mjs";
import { qualifiedThinking } from "./authoring-pi-extension.mjs";

const budgets = { timeoutMs: 15000, maxCalls: 20, maxOutputBytes: 65536, maxWriteBytes: 65536 };
const input = { files: { "AGENTS.md": "Use source truth.", ".ctx/INDEX.md": "Router", ".ctx/local/CONTINUE.md": "Checkpoint", ".agents/skills/ctx/SKILL.md": "Skill", "source.txt": "No provider secrets." }, sourceCommit: "a".repeat(40), budgets };

test("Pi thinking qualification refuses silent setting changes and records an unspecified effective setting honestly", () => {
  assert.equal(qualifiedThinking("high", "high"), "high");
  assert.equal(qualifiedThinking(null, "low"), "low");
  for (const requested of ["off", "minimal"]) assert.throws(() => qualifiedThinking(requested, "low"), /changed the requested thinking/);
  assert.throws(() => qualifiedThinking(null, undefined), /changed the requested thinking/);
});

async function temporary(t) {
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-runtime-auth-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  return root;
}

test("native Pi credentials stay single-provider, read-only, unlogged, and helper-free", async t => {
  const root = await temporary(t), path = join(root, "auth.json");
  const now = 1000000;
  const data = { anthropic: { type: "oauth", access: "SYNTHETIC-ACCESS", refresh: "SYNTHETIC-REFRESH", expires: now + 10000 }, unrelated: { type: "api_key", key: "UNRELATED-SYNTHETIC-KEY" } };
  await writeFile(path, JSON.stringify(data));
  const before = await readFile(path);
  const selected = await selectedPiCredential(path, "anthropic", 1000, now);
  const store = readOnlyCredentialStore("anthropic", selected);
  assert.deepEqual(await store.list(), [{ providerId: "anthropic", type: "oauth" }]);
  assert.equal(await store.read("unrelated"), undefined);
  const alias = await store.read("anthropic"); alias.access = "changed copy";
  assert.equal((await store.read("anthropic")).access, "SYNTHETIC-ACCESS");
  await assert.rejects(store.modify("anthropic", () => {}), /not authorized/);
  await assert.rejects(store.delete("anthropic"), /not authorized/);
  await assert.rejects(selectedPiCredential(path, "anthropic", 20000, now), /renewal/);
  assert.deepEqual(await readFile(path), before);
  await writeFile(path, JSON.stringify({ anthropic: { type: "api_key", key: "!touch /must-not-run" } }));
  await assert.rejects(selectedPiCredential(path, "anthropic", 1, now), /no helper/);
});

test("native Codex login selection preserves ChatGPT billing and refuses stale or API-key substitutes", async t => {
  const root = await temporary(t), path = join(root, "auth.json"), now = 1000000;
  const token = "fixture." + Buffer.from(JSON.stringify({ exp: (now + 10000) / 1000 })).toString("base64url") + ".unsigned-fixture";
  await writeFile(path, JSON.stringify({ tokens: { access_token: token, account_id: "fixture-account", refresh_token: "MUST-NOT-IMPORT" } }));
  const original = await readFile(path);
  const auth = await selectedCodexAuth(path, 1000, now);
  assert.equal(auth.type, "chatgptAuthTokens");
  assert.equal(JSON.stringify(auth).includes("MUST-NOT-IMPORT"), false);
  await assert.rejects(selectedCodexAuth(path, 20000, now), /renewal/);
  assert.deepEqual(await readFile(path), original);
  await writeFile(path, JSON.stringify({ OPENAI_API_KEY: "SYNTHETIC-API-KEY" }));
  await assert.rejects(selectedCodexAuth(path, 1000, now), /ChatGPT login/);
});

test("custom Pi registry selection imports only the exact model and refuses credential helpers or fallback routes", async t => {
  const root = await temporary(t), path = join(root, "models.json");
  const config = { providers: { fixture: { api: "openai-completions", baseUrl: "https://example.invalid/v1", headers: { "User-Agent": "fixture-version" }, models: [{ id: "exact-model", name: "Fixture", reasoning: false, input: ["text"], cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 }, contextWindow: 4096, maxTokens: 256 }] }, unrelated: { apiKey: "!must-not-run" } } };
  await writeFile(path, JSON.stringify(config));
  const selected = await selectedModelConfig(path, "fixture", "exact-model", { getModel: () => undefined });
  assert.equal(selected.models.length, 1);
  assert.equal(selected.models[0].id, "exact-model");
  await assert.rejects(selectedModelConfig(path, "fixture", "missing", { getModel: () => undefined }), /no fallback/);
  config.providers.fixture.apiKey = "!must-not-run";
  await writeFile(path, JSON.stringify(config));
  await assert.rejects(selectedModelConfig(path, "fixture", "exact-model", { getModel: () => undefined }), /resolver/);
  delete config.providers.fixture.apiKey;
  config.providers.fixture.headers.Authorization = "SYNTHETIC-HEADER-SECRET";
  await writeFile(path, JSON.stringify(config));
  await assert.rejects(selectedModelConfig(path, "fixture", "exact-model", { getModel: () => undefined }), /Credential headers/);
});

test("Ollama Cloud daemon route permits only its existing loopback placeholder and never requires a new key", async t => {
  const root = await temporary(t), path = join(root, "models.json");
  const config = { providers: { ollama: { api: "openai-completions", baseUrl: "http://127.0.0.1:11434/v1", apiKey: "ollama", models: [{ id: "glm-fixture:cloud", name: "Fixture", reasoning: true, input: ["text"], cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 }, contextWindow: 4096, maxTokens: 256 }] } } };
  await writeFile(path, JSON.stringify(config));
  const runtime = { getModel: () => undefined };
  const selected = await selectedModelConfig(path, "ollama", "glm-fixture:cloud", runtime, { localDaemon: true });
  assert.equal(selected.apiKey, "ollama", "This is the existing public local placeholder, not a new credential");
  assert.equal(selected.models.length, 1);
  await assert.rejects(selectedModelConfig(path, "ollama", "glm-fixture:cloud", runtime), /resolver/);
  config.providers.ollama.baseUrl = "https://example.invalid/v1";
  await writeFile(path, JSON.stringify(config));
  await assert.rejects(selectedModelConfig(path, "ollama", "glm-fixture:cloud", runtime, { localDaemon: true }), /loopback/);
  config.providers.ollama.baseUrl = "http://127.0.0.1:11434/v1";
  config.providers.ollama.apiKey = "SYNTHETIC-SECRET-NOT-THE-PLACEHOLDER";
  await writeFile(path, JSON.stringify(config));
  await assert.rejects(selectedModelConfig(path, "ollama", "glm-fixture:cloud", runtime, { localDaemon: true }), /resolver/);
  await writeFile(path, JSON.stringify({ providers: { unrelated: { apiKey: "!must-not-run" } } }));
  assert.equal(await selectedModelConfig(path, "anthropic", "native-exact-model", { getModel: () => ({ id: "native-exact-model" }) }), null, "An unrelated registry must not block a native catalog model");
});

test("local preflight failures expose fixed actionable categories rather than private error payloads", () => {
  assert.equal(preflightFailure(new Error("Stored OAuth credential needs renewal outside the runner")).reason, "native-login-needs-renewal");
  const failure = preflightFailure(new Error("PRIVATE-TOKEN-IN-ERROR"));
  assert.equal(JSON.stringify(failure).includes("PRIVATE-TOKEN"), false);
  assert.equal(failure.inferenceRequested, false);
});

test("Pi request metadata rejects missing, nonfinite, negative, and inconsistent runtime fields before inference", () => {
  const model = { name: "Fixture", api: "openai-completions", baseUrl: "http://127.0.0.1:11434/v1", reasoning: true, input: ["text"], contextWindow: 8192, maxTokens: 1024, cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 } };
  assert.equal(validatePiModel(model), model);
  for (const field of Object.keys(model)) {
    const incomplete = { ...model }; delete incomplete[field];
    assert.throws(() => validatePiModel(incomplete), /complete valid request/);
  }
  for (const change of [{ maxTokens: NaN }, { maxTokens: Infinity }, { contextWindow: 0 }, { maxTokens: -1 }, { maxTokens: 8193 }, { maxTokens: 1.5 }, { reasoning: "true" }, { input: [] }, { input: ["audio"] }, { cost: { ...model.cost, output: -1 } }, { cost: { ...model.cost, tiers: [{ ...model.cost, inputTokensAbove: NaN }] } }]) assert.throws(() => validatePiModel({ ...model, ...change }), /complete valid request/);
  assert.equal(preflightFailure(new Error("Selected Pi model requires complete valid request limits, input, reasoning, and cost metadata")).reason, "selected-model-metadata-incomplete-or-invalid");
});

test("sparse custom Pi models are refused instead of inheriting unsafe extension-registration defaults", async t => {
  const root = await temporary(t), path = join(root, "models.json");
  await writeFile(path, JSON.stringify({ providers: { fixture: { api: "openai-completions", baseUrl: "https://example.invalid/v1", models: [{ id: "exact-model", name: "Fixture", input: ["text"], contextWindow: 8192 }] } } }));
  await assert.rejects(selectedModelConfig(path, "fixture", "exact-model", { getModel: () => undefined }), /complete valid request/);
});

test("Codex model routing requires an exact advertised model and reasoning effort, never a display-name alias", () => {
  const entries = [{ id: "picker-id", model: "exact-model", displayName: "Requested Display Name", supportedReasoningEfforts: [{ reasoningEffort: "high" }] }];
  assert.deepEqual(qualifyCodexModel(entries, "exact-model", "high"), { exactModelAdvertised: true, requestedEffortAdvertised: true });
  for (const model of ["picker-id", "Requested Display Name", "other-model"]) assert.throws(() => qualifyCodexModel(entries, model, "high"), /not advertised/);
  assert.throws(() => qualifyCodexModel(entries, "exact-model", "max"), /not advertised/);
});

test("authenticated Codex rejects an unadvertised model before creating a thread or requesting inference", async t => {
  const root = await temporary(t), executable = join(root, "scripted-catalog"), methods = join(root, "methods.jsonl");
  await writeFile(executable, `#!${process.execPath}
const {appendFileSync}=require('node:fs');
require('node:readline').createInterface({input:process.stdin}).on('line',line=>{
 const m=JSON.parse(line);appendFileSync(${JSON.stringify(methods)},JSON.stringify(m.method)+'\\n');if(!m.id)return;
 const result=m.method==='skills/list'?{data:[]}:m.method==='model/list'?{data:[{model:'different-model'}],nextCursor:null}:{};
 process.stdout.write(JSON.stringify({id:m.id,result})+'\\n');
});`, { mode: 0o700 });
  const result = await runCodex({ executable, cwd: root, env: coordinatorEnv(), input, model: "exact-requested-model", provider: "openai", thinking: "high", systemPrompt: "No inference.", prompt: "Must not be sent.", auth: { type: "chatgptAuthTokens", accessToken: "SYNTHETIC", chatgptAccountId: "SYNTHETIC" } });
  assert.equal(result.runtime.incomplete, true);
  assert.equal(result.runtime.interrupted, "model-route-not-qualified");
  assert.equal(result.runtime.modelQualification.inferenceRequested, false);
  assert.equal(/thread\/start|turn\/start/.test(await readFile(methods, "utf8")), false);
});

test("installed Pi constructs finite request limits and accounts for a scripted completion using a complete custom model", { skip: !process.env.CTX_TEST_PI }, async t => {
  const root = await temporary(t), agentDir = join(root, "agent"), modelsFile = join(root, "models.json");
  await mkdir(agentDir);
  const previous = { input: process.env.CTX_AUTHORING_INPUT, result: process.env.CTX_AUTHORING_RESULT };
  const inputPath = join(root, "input.json");
  await writeFile(inputPath, JSON.stringify({ ...input, thinking: "high" }));
  process.env.CTX_AUTHORING_INPUT = inputPath; process.env.CTX_AUTHORING_RESULT = join(root, "proposal.json");
  t.after(() => { for (const [key, value] of [["CTX_AUTHORING_INPUT", previous.input], ["CTX_AUTHORING_RESULT", previous.result]]) if (value === undefined) delete process.env[key]; else process.env[key] = value; });
  const requests = [];
  const server = createServer(async (request, response) => {
    let data = ""; for await (const chunk of request) data += chunk;
    requests.push(JSON.parse(data));
    response.writeHead(200, { "Content-Type": "text/event-stream" });
    for (const chunk of [{ choices: [{ index: 0, delta: { role: "assistant", content: "READY" }, finish_reason: null }] }, { choices: [{ index: 0, delta: {}, finish_reason: "stop" }], usage: { prompt_tokens: 12, completion_tokens: 2, total_tokens: 14 } }]) response.write("data: " + JSON.stringify({ id: "fixture", object: "chat.completion.chunk", model: "glm-fixture:cloud", ...chunk }) + "\n\n");
    response.end("data: [DONE]\n\n");
  });
  await new Promise((resolve, reject) => { server.once("error", reject); server.listen(0, "127.0.0.1", resolve); });
  t.after(() => { server.closeAllConnections(); server.close(); });
  await writeFile(modelsFile, JSON.stringify({ providers: { ollama: { api: "openai-completions", baseUrl: "http://127.0.0.1:" + server.address().port + "/v1", apiKey: "ollama", models: [{ id: "glm-fixture:cloud", name: "Scripted fixture", reasoning: true, input: ["text"], contextWindow: 32768, maxTokens: 1024, cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 }, compat: { supportsReasoningEffort: true } }] } } }));
  const pi = await loadPi(process.env.CTX_TEST_PI);
  const session = await createIsolatedPi({ pi, cwd: root, agentDir, provider: "ollama", model: "glm-fixture:cloud", thinking: "high", credential: { type: "api_key", key: "ollama" }, systemPrompt: "Scripted test only.", modelsFile, localDaemon: true });
  const messages = [];
  try { session.subscribe(event => { if (event.type === "message_end" && event.message?.role === "assistant") messages.push(event.message); }); await session.prompt("Reply READY. No tools."); }
  finally { session.dispose(); }
  assert.equal(requests.length, 1);
  assert.equal(requests[0].model, "glm-fixture:cloud");
  assert.equal(requests[0].reasoning_effort, "high");
  assert.equal(requests[0].max_tokens ?? requests[0].max_completion_tokens, 1024);
  assert.equal(messages.at(-1)?.stopReason, "stop");
  assert.equal(messages.at(-1)?.usage.totalTokens, 14);
  assert.equal(Number.isFinite(messages.at(-1)?.usage.cost.total), true);
});

test("installed Pi SDK starts with an in-memory session and the same guarded tools without inference", { skip: !process.env.CTX_TEST_PI }, async t => {
  const root = await temporary(t), agentDir = join(root, "agent"), inputPath = join(root, "input.json"), proposalPath = join(root, "proposal.json");
  await mkdir(agentDir);
  await writeFile(inputPath, JSON.stringify(input));
  const previous = { input: process.env.CTX_AUTHORING_INPUT, result: process.env.CTX_AUTHORING_RESULT };
  process.env.CTX_AUTHORING_INPUT = inputPath; process.env.CTX_AUTHORING_RESULT = proposalPath;
  t.after(() => {
    for (const [key, value] of [["CTX_AUTHORING_INPUT", previous.input], ["CTX_AUTHORING_RESULT", previous.result]]) if (value === undefined) delete process.env[key]; else process.env[key] = value;
  });
  const pi = await loadPi(process.env.CTX_TEST_PI);
  await assert.rejects(createIsolatedPi({ pi, cwd: root, agentDir, provider: "anthropic", model: "claude-opus-5-5", thinking: "off", credential: { type: "oauth", access: "SYNTHETIC-ACCESS", refresh: "SYNTHETIC-REFRESH", expires: Date.now() + 3600000 }, systemPrompt: "No inference is authorized." }), /changed the requested thinking/);
  const catalog = await pi.ModelRuntime.create({ credentials: readOnlyCredentialStore("anthropic", { type: "oauth", access: "SYNTHETIC-ACCESS", refresh: "SYNTHETIC-REFRESH", expires: Date.now() + 3600000 }), modelsPath: null, modelsStorePath: join(agentDir, "local-models.json"), allowModelNetwork: false, refreshOnCreate: false });
  const model = "claude-opus-5-5";
  assert.ok(catalog.getModel("anthropic", model));
  const session = await createIsolatedPi({ pi, cwd: root, agentDir, provider: "anthropic", model, thinking: "high", credential: { type: "oauth", access: "SYNTHETIC-ACCESS", refresh: "SYNTHETIC-REFRESH", expires: Date.now() + 3600000 }, systemPrompt: "Test-only fixed scope." });
  try {
    assert.deepEqual(session.getActiveToolNames().sort(), [...toolNames].sort());
    const proposal = JSON.parse(await readFile(proposalPath, "utf8"));
    assert.equal(proposal.calls, 0);
  } finally { session.dispose(); }
  const auth = join(root, "auth.json");
  await writeFile(auth, JSON.stringify({ anthropic: { type: "oauth", access: "SYNTHETIC-ACCESS", refresh: "SYNTHETIC-REFRESH", expires: Date.now() + 3600000 } }));
  const before = await readFile(auth);
  const qualified = await preflightRuntime({ kind: "pi-sdk", executable: process.env.CTX_TEST_PI, provider: "anthropic", model, thinking: "high", auth: { kind: "native-file", path: auth } });
  assert.equal(qualified.inferenceRequested, false);
  assert.equal(qualified.providerAccess, "not-tested");
  assert.equal(qualified.requestedThinking, "high");
  assert.equal(qualified.effectiveThinking, "high");
  assert.equal(JSON.stringify(qualified).includes("SYNTHETIC-ACCESS"), false);
  assert.deepEqual(await readFile(auth), before);
});

test("Codex thread and turn configuration removes environment access and model fallback", () => {
  const params = codexThreadParams({ model: "explicit-model", provider: "openai", cwd: "/private/run", systemPrompt: "Fixed scope" });
  assert.deepEqual(params.environments, []);
  assert.deepEqual(params.runtimeWorkspaceRoots, []);
  assert.equal(params.ephemeral, true);
  assert.equal(params.allowProviderModelFallback, false);
  assert.deepEqual(dynamicTools().map(tool => tool.name), toolNames);
  assert.throws(() => codexConfig({ model: "explicit", fixtureBaseUrl: "https://api.openai.com" }));
});

test("native Codex cannot turn a supervisor timeout into a complete run during termination grace", async t => {
  const root = await temporary(t), executable = join(root, "scripted-codex");
  await writeFile(executable, `#!${process.execPath}
const readline = require("node:readline");
const send = value => process.stdout.write(JSON.stringify(value) + "\\n");
const completed = () => send({method:"turn/completed", params:{threadId:"fixture", turn:{status:"completed"}}});
process.on("SIGTERM", () => { completed(); process.exitCode = 0; process.stdin.destroy(); });
readline.createInterface({input:process.stdin}).on("line", line => {
  const message = JSON.parse(line);
  if (!message.id) return;
  let result = {};
  if (message.method === "skills/list") result = {data:[]};
  if (message.method === "thread/start") result = {model:"fixture-model", modelProvider:"openai", thread:{id:"fixture", ephemeral:true}};
  send({id:message.id, result});
  if (message.method === "turn/start") send({method:"item/completed", params:{threadId:"fixture", item:{type:"agentMessage", phase:"final_answer", text:"Late answer"}}});
});
`, { mode: 0o700 });
  await chmod(executable, 0o700);
  const result = await runCodex({ executable, cwd: root, env: coordinatorEnv(), input: { ...input, budgets: { ...budgets, timeoutMs: 1000 } }, model: "fixture-model", provider: "openai", thinking: "low", systemPrompt: "No provider calls.", prompt: "Scripted timeout fixture." });
  assert.equal(result.final, "Late answer");
  assert.equal(result.runtime.interrupted, "time-budget");
  assert.equal(result.runtime.incomplete, true);
});

function sse(response, item, number) {
  response.writeHead(200, { "Content-Type": "text/event-stream" });
  const emit = (type, payload) => response.write(`event: ${type}\ndata: ${JSON.stringify({ type, ...payload })}\n\n`);
  emit("response.created", { response: { id: "fixture-" + number, status: "in_progress", output: [] } });
  emit("response.output_item.added", { output_index: 0, item });
  emit("response.output_item.done", { output_index: 0, item });
  emit("response.completed", { response: { id: "fixture-" + number, status: "completed", output: [item], usage: { input_tokens: 10, output_tokens: 2, total_tokens: 12 } } });
  response.end();
}

test("installed native Codex confines authoring tools and refuses scripted boundary escapes", { skip: !process.env.CTX_TEST_CODEX }, async t => {
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-native-codex-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const home = join(root, "native-state"), cwd = join(root, "workspace");
  await mkdir(home); await mkdir(cwd);
  const secretRoot = join(root, "peer");
  await mkdir(secretRoot);
  await writeFile(join(secretRoot, "SKILL.md"), "PEER-SKILL-SECRET-CANARY");
  const marker = join(root, "must-not-write");
  const calls = [
    { name: "ctx_list", arguments: { path: "." } },
    { type: "custom_tool_call", namespace: "functions", name: "exec", input: 'text(await tools.ctx_read({path:"source.txt"}));' },
    { type: "custom_tool_call", namespace: "functions", name: "exec", input: 'text({process:typeof process,require:typeof require,fetch:typeof fetch});' },
    { type: "custom_tool_call", namespace: "functions", name: "exec", input: 'await import("node:fs");' },
    { type: "custom_tool_call", namespace: "functions", name: "exec", input: 'text(await tools.apply_patch(' + JSON.stringify("*** Begin Patch\n*** Add File: " + marker + "\n+unauthorized\n*** End Patch") + '));' },
    { name: "exec_command", arguments: { cmd: "touch " + marker } },
    { name: "spawn_agent", arguments: { message: "This must be refused" } },
  ];
  const requests = [];
  const server = createServer(async (request, response) => {
    let data = "";
    for await (const chunk of request) data += chunk;
    if (!request.url.endsWith("/responses")) { response.writeHead(404); response.end(); return; }
    const body = JSON.parse(data);
    requests.push(body);
    const call = calls[requests.length - 1];
    if (call) sse(response, { id: "call-item-" + requests.length, type: "function_call", ...call, call_id: "fixture-call-" + requests.length, arguments: JSON.stringify(call.arguments) }, requests.length);
    else sse(response, { id: "answer", type: "message", role: "assistant", phase: "final_answer", content: [{ type: "output_text", text: "Deterministic fixture completed.", annotations: [] }] }, requests.length);
  });
  await new Promise((resolve, reject) => { server.once("error", reject); server.listen(0, "127.0.0.1", resolve); });
  t.after(() => { server.closeAllConnections(); server.close(); });
  const config = codexConfig({ model: "gpt-6.1-sol", provider: "ctx-fixture", fixtureBaseUrl: "http://127.0.0.1:" + server.address().port });
  await writeFile(join(home, "config.toml"), config, { mode: 0o600 });
  const result = await runCodex({ executable: process.env.CTX_TEST_CODEX, cwd, env: { ...coordinatorEnv(), CODEX_HOME: home }, input, model: "gpt-6.1-sol", provider: "ctx-fixture", thinking: "low", systemPrompt: "Deterministic boundary test. Use guarded tools only.", prompt: "This is not a model trial. Execute the scripted fixture." });
  assert.equal(result.runtime.incomplete, false, JSON.stringify(result.runtime));
  assert.equal(result.final, "Deterministic fixture completed.");
  assert.equal(result.proposal.events[0]?.tool, "ctx_list");
  const declared = requests[0].tools || requests[0].input.filter(item => item.type === "additional_tools").flatMap(item => item.tools);
  const names = declared.flatMap(tool => tool.type === "namespace" ? tool.tools.map(child => tool.name + "." + child.name) : [tool.name || tool.type]);
  assert.deepEqual(names.sort(), ["functions.exec", "functions.wait", "functions.request_user_input", "functions.request_user_input_async"].sort(), "Unexpected native capability declaration");
  const nestedDescription = declared.find(tool => tool.name === "functions").tools.find(tool => tool.name === "exec").description;
  const nestedNames = [...nestedDescription.matchAll(/declare const tools: \{ ([A-Za-z0-9_]+)/g)].map(match => match[1]);
  assert.deepEqual(nestedNames.sort(), [...toolNames, "apply_patch", "list_mcp_resource_templates", "list_mcp_resources", "read_mcp_resource", "clock__curr_time"].sort());
  assert.equal(requests.length, calls.length + 1);
  const outputs = requests.flatMap(body => (body.input || []).filter(item => ["function_call_output", "custom_tool_call_output"].includes(item.type)));
  const nestedRead = outputs.find(item => item.call_id === "fixture-call-2");
  assert.ok(nestedRead, "The native code-mode guarded read must actually execute");
  assert.match(JSON.stringify(nestedRead.output), /No provider secrets/);
  const globals = outputs.find(item => item.call_id === "fixture-call-3");
  for (const name of ["process", "require", "fetch"]) assert.match(JSON.stringify(globals.output), new RegExp(name + '.{0,12}undefined'));
  assert.match(JSON.stringify(outputs.find(item => item.call_id === "fixture-call-4").output), /(?:not supported|not allowed|disabled|cannot|error)/i);
  assert.match(JSON.stringify(outputs.find(item => item.call_id === "fixture-call-5").output), /(?:environment|not allowed|disabled|cannot|error)/i);
  assert.equal(JSON.stringify(requests).includes("PEER-SKILL-SECRET-CANARY"), false);
  await assert.rejects(lstat(marker), { code: "ENOENT" });
  assert.equal(JSON.stringify(requests).includes("OPENAI_API_KEY"), false);
  assert.equal(await readFile(join(home, "config.toml"), "utf8"), config);
  // Native MCP bookkeeping is not an authoring capability. Even with no
  // configured server, an attempted native MCP item quarantines the run.
  calls.splice(0, calls.length, { type: "custom_tool_call", namespace: "functions", name: "exec", input: 'text(await tools.read_mcp_resource({server:"peer",uri:' + JSON.stringify(secretRoot + "/SKILL.md") + '}));' });
  requests.length = 0;
  const denied = await runCodex({ executable: process.env.CTX_TEST_CODEX, cwd, env: { ...coordinatorEnv(), CODEX_HOME: home }, input, model: "gpt-6.1-sol", provider: "ctx-fixture", thinking: "low", systemPrompt: "Scripted boundary check.", prompt: "No real inference." });
  assert.equal(denied.runtime.incomplete, true);
  assert.equal(denied.runtime.interrupted, "unexpected-native-tool");
  assert.equal(JSON.stringify(requests).includes("PEER-SKILL-SECRET-CANARY"), false);
});
