#!/usr/bin/env node
// tools/prepare-release.mjs <version>
//
// semantic-release prepare step: sync the version into root + npm/package.json
// (surgical replace, preserves formatting) and cross-compile the release
// binaries into dist/. Mirrors jarvis's tools/sync-versions.mjs + build step.
import { readFileSync, writeFileSync } from "node:fs";
import { execSync } from "node:child_process";

const version = process.argv[2];
if (!version) {
  console.error("usage: node tools/prepare-release.mjs <version>");
  process.exit(1);
}

// Sync the top-level "version" field in root + npm/package.json.
for (const f of ["package.json", "npm/package.json"]) {
  const content = readFileSync(f, "utf8");
  const updated = content.replace(/"version":\s*"[^"]*"/, `"version": "${version}"`);
  if (updated === content) continue;
  writeFileSync(f, updated);
  console.log(`synced ${f} -> ${version}`);
}

// Cross-compile the release binaries into dist/.
execSync(`node tools/build-binaries.mjs ${version}`, { stdio: "inherit" });