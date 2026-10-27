#!/usr/bin/env node
// Local qualification only: no session.prompt(), no turn/start, no provider call.
import { execFileSync } from "node:child_process";
import { mkdtemp, mkdir, readFile, realpath, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { createIsolatedPi, loadPi } from "./authoring-pi-sdk.mjs";
import { selectedCodexAuth, selectedPiCredential } from "./lib/authoring-auth.mjs";
import { codexVersion } from "./lib/authoring-codex.mjs";
import { coordinatorEnv } from "./lib/authoring-workspace.mjs";

export async function preflightRuntime(runtime, validityMs = 1200000) {
  const localDaemon = runtime.auth?.kind === "local-daemon" && runtime.kind === "pi-sdk" && runtime.provider === "ollama" && isAbsolute(runtime.modelsFile || "");
  if (!["codex", "pi-sdk"].includes(runtime.kind) || !isAbsolute(runtime.executable || "") || (!localDaemon && (!isAbsolute(runtime.auth?.path || "") || runtime.auth.kind !== "native-file")) ||
      ![runtime.provider, runtime.model].every(value => typeof value === "string" && /^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$/.test(value)) ||
      (runtime.thinking != null && !["off", "minimal", "low", "medium", "high", "xhigh", "max"].includes(runtime.thinking)) || !Number.isSafeInteger(validityMs) || validityMs < 1) throw new Error("Explicit runtime, model, native credential path, and validity window required");
  if (runtime.kind === "codex" && runtime.provider !== "openai") throw new Error("Native Codex preflight requires the existing ChatGPT route");
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-local-runtime-preflight-"));
  const previous = { input: process.env.CTX_AUTHORING_INPUT, result: process.env.CTX_AUTHORING_RESULT };
  try {
    const agentDir = join(root, "agent"); await mkdir(agentDir, { mode: 0o700 });
    const env = { ...coordinatorEnv(), PI_CODING_AGENT_DIR: agentDir, CODEX_HOME: agentDir, RUST_LOG: "off" };
    const executable = await realpath(runtime.executable);
    const version = execFileSync(executable, ["--version"], { env, stdio: "pipe", encoding: "utf8", timeout: 10000 }).trim();
    let credentialType, tools, effectiveThinking;
    if (runtime.kind === "codex") {
      if (version !== "codex-cli " + codexVersion) throw new Error("Native Codex version is not qualified");
      await selectedCodexAuth(runtime.auth.path, validityMs);
      credentialType = "existing-chatgpt-login";
    } else {
      if (version !== "1.1.0") throw new Error("Pi version is not qualified");
      const credential = localDaemon ? { type: "api_key", key: "ollama" } : await selectedPiCredential(runtime.auth.path, runtime.provider, validityMs);
      const pi = await loadPi(executable);
      const inputPath = join(root, "input.json"), proposalPath = join(root, "proposal.json");
      await writeFile(inputPath, JSON.stringify({ files: {}, sourceCommit: "a".repeat(40), budgets: { timeoutMs: 10000, maxCalls: 1, maxOutputBytes: 1024, maxWriteBytes: 1024 } }), { mode: 0o600 });
      process.env.CTX_AUTHORING_INPUT = inputPath; process.env.CTX_AUTHORING_RESULT = proposalPath;
      const session = await createIsolatedPi({ pi, cwd: root, agentDir, model: runtime.model, provider: runtime.provider, thinking: runtime.thinking, credential, systemPrompt: "Local preflight only. No inference is authorized.", modelsFile: runtime.modelsFile, localDaemon });
      try {
        tools = session.getActiveToolNames();
        effectiveThinking = session.thinkingLevel;
        if (JSON.parse(await readFile(proposalPath, "utf8")).calls !== 0) throw new Error("Preflight unexpectedly invoked an authoring tool");
      } finally { session.dispose(); }
      credentialType = localDaemon ? "existing-ollama-daemon" : credential.type;
    }
    return { runtime: runtime.kind, version, provider: runtime.provider, model: runtime.model, requestedThinking: runtime.thinking ?? null, ...(effectiveThinking === undefined ? {} : { effectiveThinking }), credentialType, tools, localQualification: "passed", inferenceRequested: false, providerAccess: "not-tested", billingRouteChanged: false };
  } finally {
    for (const [key, value] of [["CTX_AUTHORING_INPUT", previous.input], ["CTX_AUTHORING_RESULT", previous.result]]) if (value === undefined) delete process.env[key]; else process.env[key] = value;
    await rm(root, { recursive: true, force: true });
  }
}

async function main() {
  const options = new Map(), args = process.argv.slice(2);
  for (let index = 0; index < args.length; index += 2) {
    if (!["--runtime", "--executable", "--provider", "--model", "--thinking", "--auth-file", "--auth-kind", "--models-file", "--validity-ms"].includes(args[index]) || !args[index + 1] || options.has(args[index])) throw new Error("Invalid local preflight arguments");
    options.set(args[index], args[index + 1]);
  }
  if (options.has("--auth-kind") && !["native-file", "local-daemon"].includes(options.get("--auth-kind"))) throw new Error("Unknown authentication route");
  if (options.get("--auth-kind") === "local-daemon" && options.has("--auth-file")) throw new Error("Do not mix daemon and file credential routes");
  const runtime = { kind: options.get("--runtime"), executable: options.get("--executable"), provider: options.get("--provider"), model: options.get("--model"), auth: options.get("--auth-kind") === "local-daemon" ? { kind: "local-daemon" } : { kind: "native-file", path: options.get("--auth-file") }, modelsFile: options.get("--models-file"), thinking: options.get("--thinking") ?? null };
  console.log(JSON.stringify(await preflightRuntime(runtime, Number(options.get("--validity-ms") || 1200000))));
}
export function preflightFailure(error) {
  const known = new Map([
    ["Pi changed the requested thinking setting; choose a supported explicit setting", "requested-thinking-not-supported"],
    ["Stored OAuth credential needs renewal outside the runner", "native-login-needs-renewal"],
    ["Stored Codex login needs renewal outside the runner", "native-login-needs-renewal"],
    ["Selected provider has no stored credential", "selected-provider-has-no-stored-credential"],
    ["Exact model is absent from this isolated local catalog; no fallback selected", "exact-model-not-in-local-catalog"],
    ["Selected registry requires an unsupported credential or discovery resolver", "selected-registry-resolver-not-supported"],
    ["Selected Pi model requires complete valid request limits, input, reasoning, and cost metadata", "selected-model-metadata-incomplete-or-invalid"],
    ["Stored credential requires an unsupported resolver; no helper was executed", "stored-credential-resolver-not-supported"],
  ]);
  return { localQualification: "not-passed", reason: known.get(error?.message) || (error?.code === "ENOENT" ? "configured-file-unavailable" : "local-configuration-not-qualified"), inferenceRequested: false, billingRouteChanged: false };
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main().catch(error => { console.log(JSON.stringify(preflightFailure(error))); process.exitCode = 1; });
