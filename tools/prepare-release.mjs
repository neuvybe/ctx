#!/usr/bin/env node
// tools/prepare-release.mjs <version>
//
// semantic-release prepare step: sync the version into both package manifests,
// the root tooling lockfile, and pkg/ctx/version.go, then
// cross-compile the release binaries into dist/.
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

const lockFile = "package-lock.json";
const lock = JSON.parse(readFileSync(lockFile, "utf8"));
lock.version = version;
lock.packages[""].version = version;
writeFileSync(lockFile, `${JSON.stringify(lock, null, 2)}\n`);
console.log(`synced ${lockFile} -> ${version}`);

const versionFile = "pkg/ctx/version.go";
const versionSource = readFileSync(versionFile, "utf8");
const versionPattern = /var Version = "[^"]*"/;
if (!versionPattern.test(versionSource)) {
  throw new Error(`${versionFile}: could not locate Version`);
}
const versionUpdated = versionSource.replace(versionPattern, `var Version = "${version}"`);
if (versionUpdated !== versionSource) {
  writeFileSync(versionFile, versionUpdated);
  console.log(`synced ${versionFile} -> ${version}`);
}

// Cross-compile the release binaries into dist/.
execSync(`node tools/build-binaries.mjs ${version}`, { stdio: "inherit" });
