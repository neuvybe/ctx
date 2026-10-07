#!/usr/bin/env node
// Materialize a fresh, isolated demo; never modify an existing destination.
import { cp, lstat, mkdir, readdir, realpath } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import { basename, dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repository = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const args = process.argv.slice(2);
let destination;
let withSkill = false;
for (let index = 0; index < args.length; index++) {
  if (args[index] === "--destination" && args[index + 1]) {
    destination = args[++index];
  } else if (args[index] === "--with-skill") {
    withSkill = true;
  } else {
    throw new Error("Usage: node tools/prepare-skill-demo.mjs --destination /absolute/new/path [--with-skill]");
  }
}
if (!destination || !isAbsolute(destination)) {
  throw new Error("--destination must be an absolute, not-yet-existing directory");
}
destination = resolve(destination);
// Resolve only an existing parent; no guessed recursive parent creation.
destination = join(await realpath(dirname(destination)), basename(destination));
try {
  await lstat(destination);
  throw new Error("Refusing existing destination: " + destination);
} catch (error) {
  if (error.code !== "ENOENT") throw error;
}
await mkdir(destination);
// Claim the root with mkdir, then copy into absent child paths. Copying the
// fixture onto that existing root with errorOnExist is rejected by newer Node.
const fixture = join(repository, "testdata", "booking-demo");
for (const entry of await readdir(fixture)) {
  await cp(join(fixture, entry), join(destination, entry), {
    recursive: true, force: false, errorOnExist: true,
  });
}
if (withSkill) {
  const skillParent = join(destination, ".agents", "skills");
  await mkdir(skillParent, { recursive: true });
  await cp(join(repository, "skills", "ctx"), join(skillParent, "ctx"), {
    recursive: true, force: false, errorOnExist: true,
  });
}
const git = (...arguments_) => execFileSync("git", [
  "-C", destination, "-c", "core.hooksPath=" + join(destination, ".git", "ctx-no-hooks"), ...arguments_,
], {
  encoding: "utf8", stdio: ["ignore", "pipe", "pipe"],
}).trim();
git("init", "-q");
git("add", ".");
git("-c", "user.name=ctx-demo", "-c", "user.email=ctx-demo@example.invalid",
  "-c", "commit.gpgsign=false", "commit", "-qm", "record booking demo baseline");
console.log(JSON.stringify({
  repository: destination,
  sourceCommit: git("rev-parse", "HEAD"),
  skill: withSkill ? join(destination, ".agents", "skills", "ctx", "SKILL.md") : null,
}, null, 2));
