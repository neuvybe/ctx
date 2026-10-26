#!/usr/bin/env node
// Evaluator only: equivalent fact payloads, different retrieval layouts.
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { cp, lstat, mkdir, readFile, realpath, writeFile } from "node:fs/promises";
import { basename, dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repository = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const options = new Map();
const args = process.argv.slice(2);
for (let i = 0; i < args.length; i += 2) {
  if (!["--destination", "--layout", "--ctx"].includes(args[i]) || !args[i + 1] || options.has(args[i])) throw new Error("Usage: --destination /absolute/new/path --layout flat|folder --ctx /absolute/ctx");
  options.set(args[i], args[i + 1]);
}
if (!isAbsolute(options.get("--destination") || "") || !isAbsolute(options.get("--ctx") || "") || !["flat", "folder"].includes(options.get("--layout"))) throw new Error("Absolute destination/CLI and flat or folder layout are required");
const input = resolve(options.get("--destination"));
const destination = join(await realpath(dirname(input)), basename(input));
try { await lstat(destination); throw new Error("Refusing existing destination: " + destination); }
catch (error) { if (error.code !== "ENOENT") throw error; }
const cli = options.get("--ctx");
execFileSync(cli, ["--version"], { stdio: "pipe" });
const topics = JSON.parse(await readFile(join(repository, "evals/ctx-skill/fixtures/retrieval/topics.json"), "utf8"));
const layout = options.get("--layout");
const version = layout === "flat" ? 2 : 3;
const revision = layout === "flat" ? "2.0.1" : "3.0.0";
const hash = data => createHash("sha256").update(data).digest("hex");
const sources = [...new Set(topics.flatMap(topic => topic.sources))].sort();
await mkdir(destination);
async function put(name, data) {
  const path = join(destination, name);
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, data);
}
const sourceHashes = {};
for (const source of sources) {
  const data = await readFile(join(repository, source));
  await put(source, data);
  sourceHashes[source] = hash(data);
}
await cp(join(repository, "skills/ctx"), join(destination, ".agents/skills/ctx"), { recursive: true, errorOnExist: true, force: false });
await put("AGENTS.md", "# Retrieval demo owner instructions\n\nUse the installed ctx skill for context questions. This is a read-only source\nsnapshot, not a full buildable checkout. Inspect relevant evidence; do not infer\ntruth from metadata alone. Do not change files, stage, commit, install, publish,\nread outside this repository, or delegate. Answer only the requested topic.\nName findings in full words, not shorthand identifiers.\n");
const git = (...arguments_) => execFileSync("git", ["-C", destination, "-c", "core.hooksPath=" + join(destination, ".git/ctx-no-hooks"), ...arguments_], { encoding: "utf8", stdio: "pipe" }).trim();
git("init", "-q");
git("add", ".");
git("-c", "user.name=ctx-demo", "-c", "user.email=ctx-demo@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "record retrieval source snapshot");
const commit = git("rev-parse", "HEAD");
const verifiedOn = "2026-10-25";
const metadata = paths => "<!-- ctx:doc " + JSON.stringify({ status: "verified", verifiedAt: commit + " @ " + verifiedOn, sources: [...new Set(paths)].sort() }) + " -->\n";
const factPath = category => version === 2 ? "context/" + category + ".md" : "context/" + category + "/index.md";
const categories = [...new Set(topics.map(topic => topic.category))];
for (const category of categories) {
  const selected = topics.filter(topic => topic.category === category);
  let content = "# " + category + "\n\n" + metadata(selected.flatMap(topic => topic.sources)) + "\n";
  content += "This category covers: " + selected.map(topic => topic.title).join("; ") + ".\n\n";
  if (version === 2) {
    for (const topic of selected) content += "## " + topic.title + "\n\n" + topic.body + "\n\n";
  } else {
    content += "| Topic | Use when |\n|---|---|\n";
    for (const topic of selected) {
      content += "| [" + topic.title + "](" + topic.name + ".md) | " + topic.summary + " |\n";
      await put(".ctx/context/" + category + "/" + topic.name + ".md", "# " + topic.title + "\n\n" + metadata(topic.sources) + "\n" + topic.body + "\n");
    }
  }
  await put(".ctx/" + factPath(category), content);
}
const routes = categories.map(category => "| " + category + " | `" + factPath(category) + "` |").join("\n");
await put(".ctx/INDEX.md", "# Context router\n\nCanonical owner instructions: AGENTS.md\n\n<!-- ctx:managed begin index-routing -->\nRead owner instructions, local/CONTINUE.md, then the relevant category. Load only\nneeded material; use targeted search or topic routes after orientation.\n\n| Need | Read |\n|---|---|\n" + routes + "\n<!-- ctx:managed end index-routing -->\n\n## Project-owned routing\n");
let readme = await readFile(join(repository, "pkg/ctx/template/v" + version + "/README.md"), "utf8");
for (const [key, value] of Object.entries({ FOLDER: ".ctx", PROJECT: "ctx-retrieval-demo", MODE: "team", CONTINUE_PATH: "local/CONTINUE.md", OWNER_INSTRUCTIONS_PATH: "AGENTS.md" })) readme = readme.replaceAll("{{" + key + "}}", value);
await put(".ctx/README.md", readme);
await put(".ctx/local/CONTINUE.md", "# Local checkpoint\n\nNo work is in progress. This isolated snapshot is for read-only explanation.\n");
await put(".ctx/.gitignore", "/local/\n**/.ctx-update-*\n");
await put(".ctx/config.json", JSON.stringify({ schemaVersion: 2, layoutVersion: version, templateRevision: revision, project: "ctx-retrieval-demo", mode: "team", addons: ["behavior", "glossary"] }, null, 2) + "\n");
for (const command of ["doctor", "status"]) execFileSync(cli, [command, destination], { stdio: "pipe" });
console.log(JSON.stringify({ repository: destination, layout, sourceCommit: commit, topics: topics.length, semanticHash: hash(JSON.stringify(topics)), sourceHashes, skillHash: hash(await readFile(join(destination, ".agents/skills/ctx/SKILL.md"))) }, null, 2));
