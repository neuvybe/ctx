#!/usr/bin/env node
// tools/build-binaries.mjs <version>
//
// Cross-compiles the ctx CLI for the release matrix and packs each target as
// dist/ctx_<version>_<goos>_<goarch>.tar.gz (containing a `ctx` binary) — the
// naming ctx upgrade's pickAsset expects. Also writes dist/checksums.txt.
// CGO_ENABLED=0 → static binaries (ctx has no C deps).
//
// Used by semantic-release's @semantic-release/exec prepareCmd, and by the
// manual one-time v0.1.0 cut.

import { execSync } from "node:child_process";
import { mkdirSync, rmSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";

const raw = (process.argv[2] ?? "0.0.0-dev").replace(/^v/, "");
const version = raw;
const targets = [
  ["darwin", "amd64"],
  ["darwin", "arm64"],
  ["linux", "amd64"],
  ["linux", "arm64"],
];
// Windows is intentionally excluded from v0.1.0: ctx upgrade's archive entry
// name is `ctx` (not `ctx.exe`); add Windows once upgrade is OS-aware on entry
// names. See docs/known-issues in the .ctx platform.

const dist = "dist";
rmSync(dist, { recursive: true, force: true });
mkdirSync(dist, { recursive: true });

const ldflags = `-X github.com/neuvybe/ctx/pkg/ctx.Version=${version}`;
const shacmd = process.platform === "darwin" ? "shasum -a 256" : "sha256sum";

for (const [goos, goarch] of targets) {
  const staging = join(dist, `ctx_${version}_${goos}_${goarch}`);
  mkdirSync(staging, { recursive: true });
  const out = join(staging, "ctx");
  execSync(`go build -trimpath -ldflags "${ldflags}" -o "${out}" ./cmd/ctx`, {
    stdio: "inherit",
    env: { ...process.env, GOOS: goos, GOARCH: goarch, CGO_ENABLED: "0" },
  });
  execSync(`tar -C "${staging}" -czf "${dist}/ctx_${version}_${goos}_${goarch}.tar.gz" ctx`, {
    stdio: "inherit",
  });
  rmSync(staging, { recursive: true, force: true });
  console.log(`✓ ctx_${version}_${goos}_${goarch}.tar.gz`);
}

execSync(`${shacmd} ctx_*.tar.gz > checksums.txt`, { stdio: "inherit", cwd: dist });
console.log("✓ dist/checksums.txt");

// Summary
const archives = readdirSync(dist).filter((f) => f.endsWith(".tar.gz"));
const totalKb = archives.reduce((s, f) => s + statSync(join(dist, f)).size, 0) / 1024;
console.log(`\nBuilt ${archives.length} archives (${Math.round(totalKb)} KB total) for v${version}:`);
for (const f of archives) console.log(`  ${f}`);