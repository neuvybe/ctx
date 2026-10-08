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
  if (!["--destination", "--ctx"].includes(key) || !args[index + 1] || options.has(key)) {
    throw new Error("Usage: node tools/prepare-evidence-review.mjs --destination /absolute/new/path --ctx /absolute/ctx");
  }
  options.set(key, args[index + 1]);
}
const destination = options.get("--destination");
const cli = options.get("--ctx");
if (!destination || !isAbsolute(destination) || !cli || !isAbsolute(cli)) {
  throw new Error("--destination and --ctx must be absolute paths; destination must not exist");
}
// Validate the CLI before claiming a destination; the existing helper owns the
// fresh-directory checks and records only the demo baseline commit.
execFileSync(cli, ["--version"], { stdio: ["ignore", "pipe", "pipe"] });
const prepared = JSON.parse(execFileSync(process.execPath, [
  join(repository, "tools/prepare-skill-demo.mjs"), "--destination", destination, "--with-skill",
], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }));
execFileSync(cli, ["init", prepared.repository, "--folder", ".agent", "--mode", "team"], {
  stdio: ["ignore", "pipe", "pipe"],
});
const verifiedOn = new Date().toISOString().slice(0, 10);
for (const name of ["overview", "behavior", "architecture", "caveats", "glossary"]) {
  const template = await readFile(join(repository, "evals/ctx-skill/fixtures/evidence-review", name + ".md"), "utf8");
  const rendered = template.replaceAll("{{SOURCE_COMMIT}}", prepared.sourceCommit)
    .replaceAll("{{VERIFIED_DATE}}", verifiedOn);
  await writeFile(join(prepared.repository, ".agent/context", name + ".md"), rendered);
}
for (const name of ["README.md", "INDEX.md"]) {
  const file = join(prepared.repository, ".agent", name);
  const content = await readFile(file, "utf8");
  await writeFile(file, content.replaceAll("{{OWNER_INSTRUCTIONS_PATH}}", "AGENTS.md"));
}
console.log(JSON.stringify({ ...prepared, contextFolder: ".agent", verifiedOn }, null, 2));
