#!/usr/bin/env node
// @neuvybe/ctx bin shim: execs the platform binary that install.js dropped one
// level above this bin/ shim.
import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";

const binary = process.platform === "win32" ? "ctx.exe" : "ctx";
const bin = fileURLToPath(new URL(`../${binary}`, import.meta.url));
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
