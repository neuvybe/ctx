#!/usr/bin/env node
// Evaluator fixture: correct application, valid metadata, some unsupported prose.
import { execFileSync } from "node:child_process";
import { readFile, writeFile } from "node:fs/promises";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repository = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const options = new Map();
const args = process.argv.slice(2);
for (let index = 0; index < args.length; index += 2) {
  const key = args[index];
  if (!["--destination", "--ctx", "--fixture"].includes(key) || !args[index + 1] || options.has(key)) {
    throw new Error("Usage: node tools/prepare-evidence-review.mjs --destination /absolute/new/path --ctx /absolute/ctx [--fixture booking-demo|settings-demo]");
  }
  options.set(key, args[index + 1]);
}
const destination = options.get("--destination");
const cli = options.get("--ctx");
const fixtureName = options.get("--fixture") || "booking-demo";
if (!["booking-demo", "settings-demo"].includes(fixtureName)) throw new Error("--fixture must be booking-demo or settings-demo");
if (!destination || !isAbsolute(destination) || !cli || !isAbsolute(cli)) {
  throw new Error("--destination and --ctx must be absolute paths; destination must not exist");
}
// Validate the CLI before claiming a destination; the existing helper owns the
// fresh-directory checks and records only the demo baseline commit.
execFileSync(cli, ["--version"], { stdio: ["ignore", "pipe", "pipe"] });
const prepared = JSON.parse(execFileSync(process.execPath, [
  join(repository, "tools/prepare-skill-demo.mjs"), "--destination", destination, "--with-skill", "--fixture", fixtureName,
], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }));
execFileSync(cli, ["init", prepared.repository, "--folder", ".agent", "--mode", "team"], {
  stdio: ["ignore", "pipe", "pipe"],
});
const verifiedOn = new Date().toISOString().slice(0, 10);
const config = JSON.parse(await readFile(join(prepared.repository, ".agent/config.json"), "utf8"));
if (![2, 3].includes(config.layoutVersion)) throw new Error("Review fixtures require layout v2 or v3");
for (const name of ["overview", "behavior", "architecture", "caveats", "glossary"]) {
  const fixtureDirectory = fixtureName === "booking-demo" ? "evidence-review" : "settings-review";
  const template = await readFile(join(repository, "evals/ctx-skill/fixtures", fixtureDirectory, name + ".md"), "utf8");
  let rendered = template.replaceAll("{{SOURCE_COMMIT}}", prepared.sourceCommit)
    .replaceAll("{{VERIFIED_DATE}}", verifiedOn);
  if (config.layoutVersion === 3) rendered = rendered.replaceAll(/\]\((overview|behavior|architecture|caveats|glossary)\.md\)/g, "](../$1/index.md)");
  const path = config.layoutVersion === 3 ? join(name, "index.md") : name + ".md";
  await writeFile(join(prepared.repository, ".agent/context", path), rendered);
}
for (const name of ["README.md", "INDEX.md"]) {
  const file = join(prepared.repository, ".agent", name);
  const content = await readFile(file, "utf8");
  await writeFile(file, content.replaceAll("{{OWNER_INSTRUCTIONS_PATH}}", "AGENTS.md"));
}
console.log(JSON.stringify({ ...prepared, contextFolder: ".agent", layoutVersion: config.layoutVersion, verifiedOn }, null, 2));
