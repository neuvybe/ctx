#!/usr/bin/env node
// @neuvybe/ctx postinstall: fetch the matching OS/arch ctx binary from the
// GitHub release for THIS package version and extract it next to the package
// (so bin/ctx.js can exec it). Fails soft (exit 0) so `npm install` never breaks
// — the user gets a warning + a fallback (`go install`).
import { fileURLToPath } from "node:url";
import { request } from "node:https";
import * as tar from "tar";

const pkgDir = fileURLToPath(new URL(".", import.meta.url));
const version = process.env.npm_package_version;
if (!version) {
  console.warn("@neuvybe/ctx: no npm_package_version env; skipping binary fetch.");
  process.exit(0);
}

// node platform/arch -> Go GOOS/GOARCH (our asset naming).
const goos = { darwin: "darwin", linux: "linux", win32: "windows" }[process.platform];
const goarch = { x64: "amd64", arm64: "arm64" }[process.arch];
if (!goos || !goarch) {
  console.warn(
    `@neuvybe/ctx: no release binary for ${process.platform}/${process.arch}; ` +
      `install via 'go install github.com/neuvybe/ctx/cmd/ctx@latest'.`
  );
  process.exit(0);
}

const asset = `ctx_${version}_${goos}_${goarch}.tar.gz`;
const url = `https://github.com/neuvybe/ctx/releases/download/v${version}/${asset}`;
console.log(`@neuvybe/ctx: fetching ${url}`);

// Follow redirects (GitHub release downloads 302 to the CDN).
const get = (u, hops = 0) =>
  new Promise((resolve, reject) => {
    const req = request(u, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        if (hops > 5) return reject(new Error("too many redirects"));
        return resolve(get(res.headers.location, hops + 1));
      }
      if (res.statusCode !== 200) return reject(new Error(`HTTP ${res.statusCode} for ${u}`));
      resolve(res);
    });
    req.on("error", reject);
    req.end();
  });

try {
  const res = await get(url);
  await res.pipe(tar.x({ cwd: pkgDir })); // extracts the `ctx` entry to pkgDir/ctx
  console.log(`@neuvybe/ctx: installed ctx v${version} -> ${pkgDir}ctx`);
} catch (e) {
  console.warn(
    `@neuvybe/ctx: could not fetch/extract binary (${e.message}). ` +
      `Run 'npm rebuild @neuvybe/ctx' to retry, or install via 'go install github.com/neuvybe/ctx/cmd/ctx@latest'.`
  );
  process.exit(0); // soft fail
}