import assert from "node:assert/strict";
import { execFileSync, spawn, spawnSync } from "node:child_process";
import { chmod, link, mkdtemp, mkdir, readFile, realpath, rm, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import { createBoundary, digest, mayWrite, safePath, sensitivePath, toolNames, validateProposal } from "./lib/authoring-boundary.mjs";
import { auditRepository, coordinatorEnv, freshDestination, frozenSources, git, treeManifest } from "./lib/authoring-workspace.mjs";
import { publicEvent, runProcess } from "./lib/authoring-process.mjs";
import { piArguments, runAuthoring, validatePlan } from "./run-guarded-authoring.mjs";
import authoringExtension from "./authoring-pi-extension.mjs";
import { evaluateScorecard } from "./lib/context-scorecard.mjs";
import { readFrozenTrial } from "./lib/scorecard-io.mjs";

const repository = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const budgets = { timeoutMs: 60000, maxCalls: 50, maxOutputBytes: 1024 * 1024, maxWriteBytes: 1024 * 1024 };
const entrypoints = ["AGENTS.md", "CLAUDE.md", ".agents/skills/ctx/SKILL.md", ".ctx/INDEX.md", ".ctx/local/CONTINUE.md"];
const files = { ...Object.fromEntries(entrypoints.map(path => [path, "Owner or skill guidance\n"])), "src/app.ts": "export const limit = 4;\n", ".ctx/config.json": '{"mode":"team"}\n', ".ctx/context/overview/index.md": "# Overview\n" };
const input = (overrides = {}) => ({ files, sourceCommit: "a".repeat(40), budgets, ...overrides });
function orient(boundary) { for (const path of entrypoints) boundary.execute("ctx_read", { path }); }
async function temporary(t) {
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-authoring-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  return root;
}
async function fixture(t) {
  const root = await temporary(t), repo = join(root, "source");
  await mkdir(repo);
  git(repo, "init", "-q");
  await writeFile(join(repo, "AGENTS.md"), "# Owner instructions\n\nDescribe existing behavior; do not create policy.\n");
  await writeFile(join(repo, "app.txt"), "Implemented behavior: cancellation is idempotent.\n");
  await writeFile(join(repo, ".gitignore"), "/ignored.txt\n/.ctx/\n");
  git(repo, "add", ".");
  git(repo, "-c", "user.name=ctx-test", "-c", "user.email=ctx-test@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "record source fixture");
  return { root, repo, commit: git(repo, "rev-parse", "HEAD").toString().trim() };
}

test("canonical virtual paths reject traversal, absolute paths, Git internals, and platform aliases", () => {
  for (const path of ["", "/tmp/peer", "../peer/file", "a/../../peer", "a/../file", "a//file", "./file", "a/./file", "a\\file", "C:/file", ".git/config", "a/.GIT/config", "a\0b", "a\nb"]) assert.throws(() => safePath(path), path);
  assert.equal(safePath("context/scope.md"), "context/scope.md");
  assert.equal(safePath(".", { root: true }), ".");
  for (const path of [".env", "nested/.env.production", ".aws/config", ".claude/auth.json", ".npmrc", ".netrc", ".git-credentials", "cert.key", "auth.json", "secrets.txt", "nested/node_modules/x"]) assert.equal(sensitivePath(path), true, path);
  for (const path of ["src/auth.ts", "src/auth.test.ts", "src/credentials.py", "src/secrets.go"]) assert.equal(sensitivePath(path), false, path);
  for (const path of [".env.example", ".claude/auth.ts", "secrets/config.ts", "auth.json/module.ts", "auth.ts/credentials.json"]) assert.equal(sensitivePath(path), true, path);
});

test("source information describes a filtered view without exposing withheld filenames or claiming repository absence", () => {
  const counts = { "sensitive-path": 4, "binary-file": 1 };
  const boundary = createBoundary(input({ sourceOmissions: counts }));
  counts["sensitive-path"] = 99;
  const info = JSON.parse(boundary.execute("ctx_source_info", {}));
  assert.equal(info.inventoryComplete, false);
  assert.equal(Object.hasOwn(info, "trackedSourcePaths"), false);
  assert.ok(info.availableSourcePaths.includes("src/app.ts"));
  assert.deepEqual(info.omittedFileCounts, { "sensitive-path": 4, "binary-file": 1 });
  assert.match(info.limitation, /not evidence.*absent/);
  assert.equal(JSON.parse(createBoundary(input()).execute("ctx_source_info", {})).omittedFileCounts, null);
  assert.throws(() => createBoundary(input({ sourceOmissions: { "SECRET-FILENAME": 1 } })));
});

test("fresh-reader boundaries refuse every write even after complete orientation and a valid preimage", () => {
  const boundary = createBoundary(input({ readOnly: true }));
  orient(boundary);
  assert.throws(() => boundary.execute("ctx_write", { path: ".ctx/INDEX.md", text: "x", expectedHash: digest(files[".ctx/INDEX.md"]) }), /read-only/);
  assert.deepEqual(boundary.snapshot().writes, {});
  assert.equal(boundary.snapshot().events.at(-1).outcome, "denied");
  assert.equal(JSON.parse(boundary.execute("ctx_read", { path: "src/app.ts" })).path, "src/app.ts");
  assert.throws(() => createBoundary(input({ readOnly: "true" })));
});

test("tool allowlist denies commands, publication, delegation, peers, and unknown arguments without recording their payload", () => {
  const boundary = createBoundary(input());
  const marker = "PRIVATE-ARGUMENT-MARKER";
  for (const tool of ["bash", "exec", "git", "spawn_agent", "send_message", "mcp_publish"]) assert.throws(() => boundary.execute(tool, { command: marker }));
  for (const path of ["../peer/result.json", "/tmp/credentials", ".git/index", "peer/output.md"]) assert.throws(() => boundary.execute("ctx_read", { path }));
  assert.throws(() => boundary.execute("ctx_list", { path: ".", command: marker }));
  assert.equal(JSON.stringify(boundary.snapshot()).includes(marker), false);
  assert.equal(boundary.snapshot().events.every(event => event.outcome === "denied"), true);
});

test("context-only virtual writes require complete entrypoint reads and matching preimages", () => {
  const boundary = createBoundary(input());
  const path = ".ctx/context/overview/index.md";
  assert.throws(() => boundary.execute("ctx_write", { path, text: "Changed", expectedHash: digest(files[path]) }));
  orient(boundary);
  for (const path of ["src/app.ts", "AGENTS.md", "CLAUDE.md", ".ctx/config.json", ".ctx/.gitignore", ".ctx/README.md", ".ctx/context/contracts/index.md", ".ctx/local/private.key", ".ctx/context/overview/evil.md/child.md"]) {
    assert.equal(mayWrite(path), false);
    assert.throws(() => boundary.execute("ctx_write", { path, text: "unauthorized", expectedHash: null }));
  }
  assert.throws(() => boundary.execute("ctx_write", { path, text: "Changed", expectedHash: "wrong" }));
  boundary.execute("ctx_write", { path, text: "Updated overview", expectedHash: digest(files[path]) });
  boundary.execute("ctx_write", { path: ".ctx/context/behavior/access-rules.md", text: "Scoped draft", expectedHash: null });
  assert.equal(files[path], "# Overview\n", "the frozen source object remains unchanged");
  assert.equal(JSON.parse(boundary.execute("ctx_read", { path })).lines[0], "Updated overview");
  assert.throws(() => boundary.execute("ctx_write", { path, text: "Stale overwrite", expectedHash: digest(files[path]) }));
});

test("long owner instructions can be read in ranges without treating a partial read as complete", () => {
  const long = Array.from({ length: 500 }, (_, index) => "Owner line " + index).join("\n");
  const boundary = createBoundary(input({ files: { ...files, "AGENTS.md": long } }));
  orient(boundary);
  assert.equal(boundary.snapshot().fullReads.includes("AGENTS.md"), false);
  boundary.execute("ctx_read", { path: "AGENTS.md", offset: 201, limit: 300 });
  assert.equal(boundary.snapshot().fullReads.includes("AGENTS.md"), true);
});

test("hard budgets apply before tool output and virtual mutation, including simultaneous requests", async () => {
  const boundary = createBoundary(input({ budgets: { ...budgets, maxCalls: 3 } }));
  const results = await Promise.allSettled(Array.from({ length: 20 }, async () => boundary.execute("ctx_list", { path: "." })));
  assert.equal(results.filter(result => result.status === "fulfilled").length, 3);
  assert.equal(boundary.snapshot().calls, 3);
  const tiny = createBoundary(input({ budgets: { ...budgets, maxOutputBytes: 1 } }));
  assert.throws(() => tiny.execute("ctx_read", { path: "src/app.ts" }));
  assert.equal(tiny.snapshot().outputBytes, 0);
  assert.equal(tiny.exhausted(), true);
  const small = createBoundary(input({ budgets: { ...budgets, maxWriteBytes: 1 } }));
  orient(small);
  assert.throws(() => small.execute("ctx_write", { path: ".ctx/local/CONTINUE.md", text: "é", expectedHash: digest(files[".ctx/local/CONTINUE.md"]) }));
  assert.deepEqual(small.snapshot().writes, {});
  let clock = 0;
  const timed = createBoundary(input({ clock: () => clock, budgets: { ...budgets, timeoutMs: 5 } }));
  clock = 5;
  assert.throws(() => timed.execute("ctx_source_info", {}));
  assert.equal(timed.snapshot().calls, 0);
});

test("literal search and returned UTF-8 bytes are measured without shell or regex execution", () => {
  const boundary = createBoundary(input({ files: { ...files, "unicode.txt": "é $(touch nope) [.*]\n" } }));
  const response = boundary.execute("ctx_search", { path: ".", query: "$(touch nope)" });
  assert.equal(JSON.parse(response)[0].path, "unicode.txt");
  assert.equal(boundary.snapshot().outputBytes, Buffer.byteLength(response));
  const alias = { ...files };
  const frozen = createBoundary(input({ files: alias }));
  alias["src/app.ts"] = "silently mutated";
  assert.equal(JSON.parse(frozen.execute("ctx_read", { path: "src/app.ts" })).lines[0], "export const limit = 4;");
});

test("coordinator independently rejects forged proposal paths, unsupported add-ons, binary data, and file conflicts", () => {
  for (const path of ["../escape", ".ctx/config.json", "src/app.ts", ".ctx/context/contracts/index.md"]) assert.throws(() => validateProposal(input(), { writes: { [path]: "bad" } }));
  assert.throws(() => validateProposal(input(), { writes: { ".ctx/INDEX.md": "bad\0data" } }));
  assert.throws(() => validateProposal(input({ files: { ...files, ".ctx/context/overview/conflict": "file" } }), { writes: { ".ctx/context/overview/conflict/topic.md": "bad" } }));
  assert.equal(validateProposal(input(), { writes: { ".ctx/context/behavior/topic.md": "draft" } })[".ctx/context/behavior/topic.md"], "draft");
});

test("frozen Git sources omit secrets, symlinks, and ignored files, and reject dirty or wrong-commit inputs", async t => {
  const { root, repo } = await fixture(t);
  await writeFile(join(repo, ".env"), "TRACKED-SECRET-MARKER");
  await writeFile(join(repo, "auth.ts"), "export const requireAuthentication = true;\n");
  await writeFile(join(repo, "auth.json"), "CREDENTIAL-SECRET-MARKER");
  await writeFile(join(repo, ".env.example"), "EXAMPLE-STILL-REQUIRES-REVIEW-SECRET-MARKER");
  await writeFile(join(repo, "ignored.txt"), "IGNORED-SECRET-MARKER");
  await symlink(join(root, "outside-secret"), join(repo, "external-link"));
  await writeFile(join(root, "outside-secret"), "OUTSIDE-SECRET-MARKER");
  git(repo, "add", ".env", "auth.ts", "auth.json", ".env.example", "external-link");
  git(repo, "-c", "user.name=ctx-test", "-c", "user.email=ctx-test@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "record excluded input shapes");
  const commit = git(repo, "rev-parse", "HEAD").toString().trim();
  const source = await frozenSources(repo, commit);
  assert.deepEqual(Object.keys(source.files).sort(), [".gitignore", "AGENTS.md", "app.txt", "auth.ts"]);
  assert.equal(JSON.stringify(source).includes("SECRET-MARKER"), false);
  assert.equal(source.omitted.some(entry => entry.reason === "link-or-submodule"), true);
  // A hard link in the worktree cannot turn a model read into a host-file read:
  // contents are frozen Git blobs, not filesystem opens through the tool path.
  await link(join(repo, "app.txt"), join(root, "outside-hardlink"));
  assert.equal((await frozenSources(repo, commit)).files["app.txt"], source.files["app.txt"]);
  await assert.rejects(frozenSources(repo, "a".repeat(40)), /pinned commit/);
  await writeFile(join(repo, "app.txt"), "dirty source");
  await assert.rejects(frozenSources(repo, commit), /clean/);
  git(repo, "add", "app.txt");
  await assert.rejects(frozenSources(repo, commit), /clean/);
});

test("audits resolve linked worktree Git paths and detect index, ignored-state, modes, and symlink changes", async t => {
  const { root, repo, commit } = await fixture(t);
  const linked = join(root, "linked");
  git(repo, "worktree", "add", "--detach", linked, commit);
  const before = await auditRepository(linked);
  assert.notEqual(before.gitEntry, "directory");
  assert.equal(before.head, commit);
  await writeFile(join(linked, "ignored.txt"), "ignored checkpoint");
  assert.notDeepEqual(await auditRepository(linked), before);
  await rm(join(linked, "ignored.txt"));
  assert.deepEqual(await auditRepository(linked), before);
  await chmod(join(linked, "app.txt"), 0o755);
  assert.notDeepEqual((await auditRepository(linked)).entries, before.entries);
  await chmod(join(linked, "app.txt"), 0o644);
  const exclude = git(linked, "rev-parse", "--path-format=absolute", "--git-path", "info/exclude").toString().trim();
  const originalExclude = await readFile(exclude);
  await writeFile(exclude, "# modified shared exclude\n");
  assert.notEqual((await auditRepository(linked)).metadata["info/exclude"], before.metadata["info/exclude"]);
  await writeFile(exclude, originalExclude);
  assert.deepEqual(await auditRepository(linked), before);
  await writeFile(join(linked, "app.txt"), "changed staged source");
  git(linked, "add", "app.txt");
  assert.notEqual((await auditRepository(linked)).metadata.index, before.metadata.index);
  const tree = join(root, "tree");
  await mkdir(tree);
  await symlink("target-one", join(tree, "link"));
  const first = await treeManifest(tree);
  await rm(join(tree, "link"));
  await symlink("target-two", join(tree, "link"));
  assert.notDeepEqual(await treeManifest(tree), first);
});

test("audit manifests preserve filenames that match JavaScript prototype properties", async t => {
  const root = await temporary(t);
  for (const name of ["__proto__", "constructor", "toString"]) await writeFile(join(root, name), "original");
  const before = await treeManifest(root);
  for (const name of ["__proto__", "constructor", "toString"]) assert.equal(Object.hasOwn(before, name), true, name);
  await writeFile(join(root, "__proto__"), "changed");
  assert.notEqual(JSON.stringify(await treeManifest(root)), JSON.stringify(before));
});

test("operational log allowlist drops reasoning, tool payloads, private messages, raw errors, and unknown usage fields", () => {
  for (const event of [{ type: "reasoning", text: "HIDDEN" }, { type: "message_end", message: { role: "system", content: "PRIVATE" } }, { type: "tool_execution_start", args: { text: "PRIVATE" } }]) assert.equal(publicEvent(event), null);
  const result = publicEvent({ type: "message_end", message: { role: "assistant", stopReason: "stop", content: [{ type: "thinking", thinking: "HIDDEN" }, { type: "text", text: "Public final" }], usage: { input: 12, output: 3, secret: "PRIVATE" }, errorMessage: "TOKEN-IN-RAW-ERROR" } });
  assert.equal(result.final, "Public final");
  assert.deepEqual(result.usage, { input: 12, output: 3 });
  assert.equal(/HIDDEN|PRIVATE|TOKEN-IN-RAW-ERROR/.test(JSON.stringify(result)), false);
});

test("process supervisor handles spawn failure, malformed events, stream budgets, final unterminated lines, and timeouts", async t => {
  const root = await temporary(t);
  const options = { cwd: root, env: coordinatorEnv(), timeoutMs: 2000 };
  const run = code => runProcess({ ...options, executable: process.execPath, args: ["-e", code] });
  assert.equal((await runProcess({ ...options, executable: join(root, "missing"), args: [] })).spawnError, true);
  const good = await run('process.stdout.write(JSON.stringify({type:"message_end",message:{role:"assistant",stopReason:"stop",content:[{type:"text",text:"done"}]}}))');
  assert.equal(good.messages[0].final, "done");
  assert.equal((await run('process.stdout.write("not-json\\n")')).interrupted, "invalid-json-event");
  const huge = await runProcess({ ...options, executable: process.execPath, args: ["-e", 'process.stdout.write("x".repeat(10000))'], maxLineBytes: 100 });
  assert.equal(huge.interrupted, "event-size-budget");
  const noisy = await runProcess({ ...options, executable: process.execPath, args: ["-e", 'process.stderr.write("SECRET".repeat(1000))'], maxStreamBytes: 100 });
  assert.equal(noisy.interrupted, "stream-byte-budget");
  assert.equal(JSON.stringify(noisy).includes("SECRET"), false);
  const timed = await runProcess({ ...options, executable: process.execPath, args: ["-e", 'process.on("SIGTERM",()=>{}); setInterval(()=>{},1000)'], timeoutMs: 80 });
  assert.equal(timed.interrupted, "time-budget");
  const descendant = await run('const {spawn}=require("node:child_process");const child=spawn(process.execPath,["-e","setInterval(()=>{},1000)"],{stdio:"ignore"});child.unref();console.log(JSON.stringify({type:"message_end",message:{role:"assistant",stopReason:"stop",content:[{type:"text",text:String(child.pid)}]}}));');
  const pid = Number(descendant.messages[0].final);
  assert.ok(pid > 0);
  let alive = true;
  for (let attempt = 0; attempt < 50 && alive; attempt++) {
    try { process.kill(pid, 0); await new Promise(resolve => setTimeout(resolve, 10)); }
    catch (error) { if (error.code !== "ESRCH") throw error; alive = false; }
  }
  assert.equal(alive, false, "descendant must not outlive the supervised run");
});

test("minimal coordinator environment strips unrelated credentials, Git overrides, and executable injection variables", () => {
  const env = coordinatorEnv({ PATH: "/bin", HOME: "/safe", OPENAI_API_KEY: "secret", AWS_SECRET_ACCESS_KEY: "secret", GIT_DIR: "/peer/.git", NODE_OPTIONS: "--require /evil", BASH_ENV: "/evil" });
  for (const key of ["OPENAI_API_KEY", "AWS_SECRET_ACCESS_KEY", "GIT_DIR", "NODE_OPTIONS", "BASH_ENV"]) assert.equal(Object.hasOwn(env, key), false);
  assert.equal(env.GIT_CONFIG_NOSYSTEM, "1");
});

test("invalid plans and unapproved runtime combinations fail before claiming a destination", async t => {
  const root = await temporary(t);
  const plan = { repository: join(root, "missing-source"), sourceCommit: "a".repeat(40), ctx: join(root, "missing-ctx"), runtime: { kind: "pi", executable: join(root, "missing-pi"), provider: "anthropic", model: "explicit-model", thinking: "high" }, budgets };
  for (const invalid of [{ ...plan, extra: true }, { ...plan, sourceCommit: "HEAD" }, { ...plan, budgets: { ...budgets, maxCalls: 0 } }, { ...plan, runtime: { ...plan.runtime, executable: "pi" } }, { ...plan, runtime: { ...plan.runtime, model: "--fallback-model" } }]) assert.throws(() => validatePlan(invalid));
  const destination = join(root, "must-remain-absent");
  await assert.rejects(runAuthoring({ plan, destination, replay: join(root, "replay.json"), allowProviderRun: true }), /mutually exclusive/);
  await assert.rejects(runAuthoring({ plan, destination, allowProviderRun: true }), /Pi CLI provider execution is not qualified/);
  await assert.rejects(runAuthoring({ plan: { ...plan, runtime: { ...plan.runtime, kind: "codex" } }, destination, allowProviderRun: true }), /Native Codex/);
  await assert.rejects(readFile(destination), { code: "ENOENT" });
  const keep = join(root, "keep.txt");
  await writeFile(keep, "preserve existing artifacts");
  const runner = join(repository, "tools/run-guarded-authoring.mjs");
  for (const args of [[], ["--plan", "relative", "--destination", destination], ["--unknown", "value"], ["--plan", keep, "--destination", root]]) assert.notEqual(spawnSync(process.execPath, [runner, ...args], { encoding: "utf8" }).status, 0);
  assert.equal(await readFile(keep, "utf8"), "preserve existing artifacts");
});

test("Pi adapter registers only guarded tools, checks its loadout before execution, persists proposals, and aborts at the budget", async t => {
  const root = await temporary(t), inputPath = join(root, "input.json"), resultPath = join(root, "proposal.json");
  await writeFile(inputPath, JSON.stringify(input({ budgets: { ...budgets, maxCalls: 1 } })));
  const previous = { input: process.env.CTX_AUTHORING_INPUT, result: process.env.CTX_AUTHORING_RESULT };
  process.env.CTX_AUTHORING_INPUT = inputPath;
  process.env.CTX_AUTHORING_RESULT = resultPath;
  t.after(() => {
    for (const [key, value] of [["CTX_AUTHORING_INPUT", previous.input], ["CTX_AUTHORING_RESULT", previous.result]]) if (value === undefined) delete process.env[key]; else process.env[key] = value;
  });
  const tools = [], hooks = new Map();
  let active, aborted = false;
  authoringExtension({ registerTool: tool => tools.push(tool), on: (name, fn) => hooks.set(name, fn), getAllTools: () => tools, getThinkingLevel: () => "high", setActiveTools: names => { active = names; } });
  hooks.get("session_start")();
  assert.deepEqual(active, toolNames);
  assert.equal(hooks.get("tool_call")({ toolName: "bash" }).block, true);
  await tools.find(tool => tool.name === "ctx_list").execute("call", { path: "." }, undefined, undefined, { abort: () => { aborted = true; } });
  assert.equal(aborted, true);
  assert.equal(JSON.parse(await readFile(resultPath, "utf8")).calls, 1);
  tools.push({ name: "bash" });
  assert.throws(() => hooks.get("session_start")(), /Unexpected Pi tools/);
  const args = piArguments();
  for (const flag of ["--no-builtin-tools", "--no-extensions", "--no-mcp", "--no-skills", "--no-context-files", "--no-session", "--no-approve"]) assert.equal(args.includes(flag), true);
});

test("installed Pi can load the real isolated guard without a prompt or provider request", { skip: !process.env.CTX_TEST_PI }, async t => {
  const root = await temporary(t), agent = join(root, "agent"), inputPath = join(root, "input.json"), proposalPath = join(root, "proposal.json");
  await mkdir(agent);
  await writeFile(inputPath, JSON.stringify(input()));
  const args = piArguments();
  args.splice(args.indexOf("--mode") + 1, 1, "rpc");
  args.splice(args.indexOf("--print"), 1);
  args.pop(); // No author prompt; only a local get_state control request follows.
  const child = spawn(process.env.CTX_TEST_PI, args, { cwd: root, env: { ...coordinatorEnv(), PI_CODING_AGENT_DIR: agent, CTX_AUTHORING_INPUT: inputPath, CTX_AUTHORING_RESULT: proposalPath }, stdio: ["pipe", "pipe", "pipe"] });
  t.after(() => child.kill("SIGKILL"));
  let stdout = "", stderr = "";
  child.stdout.on("data", data => { stdout += data; });
  child.stderr.on("data", data => { stderr += data; });
  const timer = setTimeout(() => child.kill("SIGKILL"), 15000);
  child.stdin.write(JSON.stringify({ id: "model-free-state-check", type: "get_state" }) + "\n");
  child.stdout.on("data", () => { if (stdout.includes('"model-free-state-check"')) child.stdin.end(); });
  const code = await new Promise((resolve, reject) => { child.on("error", reject); child.on("close", resolve); });
  clearTimeout(timer);
  assert.equal(code, 0, "Pi initialization failed: " + stderr.slice(0, 1000));
  const proposal = JSON.parse(await readFile(proposalPath, "utf8"));
  assert.equal(proposal.calls, 0);
  assert.equal(stdout.includes('"message_start"'), false, "No model turn may start during this check");
});

test("runner prepares and replays a genuine scaffold with frozen attempts, coordinator checks, and an untouched linked source worktree", async t => {
  const { root, repo, commit } = await fixture(t);
  // The real CLI rejects team-mode visibility conflicts: use a source ignore
  // rule that doesn't ignore .ctx for this positive preparation test.
  await writeFile(join(repo, ".gitignore"), "/ignored.txt\n");
  git(repo, "add", ".gitignore");
  git(repo, "-c", "user.name=ctx-test", "-c", "user.email=ctx-test@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "allow team context");
  const pinned = git(repo, "rev-parse", "HEAD").toString().trim();
  assert.notEqual(pinned, commit);
  const linked = join(root, "author-source");
  git(repo, "worktree", "add", "--detach", linked, pinned);
  await writeFile(join(linked, "ignored.txt"), "preserve ignored input");
  const cli = join(root, "ctx");
  execFileSync("go", ["build", "-o", cli, "./cmd/ctx"], { cwd: repository });
  const plan = { repository: linked, sourceCommit: pinned, ctx: cli, runtime: { kind: "pi", executable: "/unavailable/pi", provider: "anthropic", model: "explicit-model", thinking: "high" }, budgets };
  validatePlan(plan);
  const before = await auditRepository(linked);
  const prepared = join(root, "prepared");
  const prep = await runAuthoring({ plan, destination: prepared });
  assert.equal(prep.runtime.mode, "prepare-only");
  assert.equal(prep.sourceUnchanged, true);
  const snapshot = JSON.parse(await readFile(join(prepared, "input.json"), "utf8"));
  assert.equal(snapshot.files[".ctx/config.json"].includes('"mode": "team"'), true);
  const replay = join(root, "replay.json");
  const operations = [...entrypoints.filter(path => Object.hasOwn(snapshot.files, path)).map(path => ({ tool: "ctx_read", input: { path, limit: 400 } })), { tool: "ctx_write", input: { path: ".ctx/context/behavior/access-rules.md", text: '# Access rules\n\n<!-- ctx:doc {"status":"draft","verifiedAt":"","sources":["app.txt"]} -->\n\nCancellation is idempotent.\n', expectedHash: null } }];
  await writeFile(replay, JSON.stringify({ operations, final: "Draft proposal; no application tests were run." }));
  const output = join(root, "replayed");
  const result = await runAuthoring({ plan, destination: output, replay });
  assert.equal(result.failure, null);
  assert.equal(result.runtime.incomplete, false);
  assert.deepEqual(result.proposal.writes, [".ctx/context/behavior/access-rules.md"]);
  assert.deepEqual(result.checks.map(check => check.command), ["doctor", "status"]);
  assert.equal(result.checks[0].code, 0);
  assert.equal((await readFile(join(output, "candidate/.ctx/context/behavior/access-rules.md"), "utf8")).includes("Cancellation is idempotent"), true);
  assert.equal(JSON.parse(await readFile(join(output, "attempt.json"), "utf8")).final, result.final);
  assert.deepEqual(await auditRepository(linked), before);
  // A genuine current scaffold remains structurally valid while draft. Replay
  // is not a provider trial, and missing semantic review never becomes approval.
  const scorecard = evaluateScorecard(await readFrozenTrial(output));
  assert.equal(scorecard.stages.structurallyValidContext.verdict, "passed", JSON.stringify(scorecard.stages.structurallyValidContext));
  assert.notEqual(scorecard.stages.successfulExecution.verdict, "passed");
  assert.equal(scorecard.stages.supportedWording.verdict, "not-evaluated");
  assert.equal(scorecard.promotion.eligibleForReviewedUse, false);
  // A refusal is an observed boundary event, not a missing/rewritten attempt.
  await writeFile(replay, JSON.stringify({ operations: [...operations, { tool: "bash", input: { command: "FORBIDDEN-COMMAND-MARKER" } }], final: "Partial attempt with a refused tool" }));
  const partialRoot = join(root, "partial");
  const partial = await runAuthoring({ plan, destination: partialRoot, replay });
  assert.equal(partial.runtime.incomplete, true);
  const attempt = await readFile(join(partialRoot, "attempt.json"), "utf8");
  assert.equal(attempt.includes('"denied"'), true);
  assert.equal(attempt.includes("FORBIDDEN-COMMAND-MARKER"), false);
  assert.deepEqual(await auditRepository(linked), before);
  await assert.rejects(runAuthoring({ plan, destination: output, replay }), /existing destination/);
  await assert.rejects(runAuthoring({ plan: { ...plan, runtime: { ...plan.runtime, kind: "codex" } }, destination: join(root, "blocked"), allowProviderRun: true }), /Native Codex/);
  await assert.rejects(freshDestination(root), /existing destination/);
  assert.notEqual(spawnSync(process.execPath, [join(repository, "tools/run-guarded-authoring.mjs"), "--allow-provider-run"]).status, 0);
});
