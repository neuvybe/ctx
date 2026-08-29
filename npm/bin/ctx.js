#!/usr/bin/env node
// @neuvybe/ctx bin shim: execs the platform ctx binary that install.js dropped
// at <pkgDir>/ctx (one level up from this bin/ shim).
import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";

const bin = fileURLToPath(new URL("../ctx", import.meta.url));
if (!existsSync(bin)) {
  console.error(
    `@neuvybe/ctx: binary not found at ${bin}. ` +
      `Run 'npm rebuild @neuvybe/ctx' (or 'npm install -g @neuvybe/ctx' again) to fetch it.`
  );
  process.exit(1);
}
const child = spawn(bin, process.argv.slice(2), { stdio: "inherit" });
child.on("exit", (code, signal) => {
  if (signal) process.kill(process.pid, signal);
  process.exit(code ?? 1);
});