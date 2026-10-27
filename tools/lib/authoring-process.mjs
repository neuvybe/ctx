import { spawn } from "node:child_process";

// No raw provider events or stderr are retained. Reasoning and native session
// records are deliberately absent from this operational allowlist.
export function publicEvent(event) {
  if (event.type !== "message_end" || event.message?.role !== "assistant") return null;
  const message = event.message;
  const result = { stopReason: ["stop", "toolUse", "error", "aborted", "length"].includes(message.stopReason) ? message.stopReason : "unknown" };
  for (const field of ["provider", "model"]) {
    if (typeof message[field] === "string" && /^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$/.test(message[field])) result[field] = message[field];
  }
  if (message.stopReason === "stop" && !(message.content || []).some(part => part.type === "toolCall")) {
    result.final = (message.content || []).filter(part => part.type === "text").map(part => part.text).join("\n").slice(0, 65536);
  }
  if (message.errorMessage) result.error = "Provider or model error; raw details omitted";
  result.usage = {};
  for (const name of ["input", "output", "cacheRead", "cacheWrite", "totalTokens"]) {
    if (Number.isFinite(message.usage?.[name]) && message.usage[name] >= 0) result.usage[name] = message.usage[name];
  }
  return result;
}

export async function runProcess({ executable, args, cwd, env, timeoutMs, maxStreamBytes = 32 * 1024 * 1024, maxLineBytes = 2 * 1024 * 1024, onRecord, onStart }) {
  if (process.platform === "win32") throw new Error("Process-group containment is not implemented on Windows");
  const child = spawn(executable, args, { cwd, env, detached: true, stdio: [onStart ? "pipe" : "ignore", "pipe", "pipe"] });
  let interrupted = null, bytes = 0, stderrBytes = 0, buffer = "", killTimer;
  const messages = [];
  const signalGroup = signal => { if (child.pid) { try { process.kill(-child.pid, signal); } catch (error) { if (error.code !== "ESRCH") throw error; } } };
  const stop = reason => {
    if (interrupted) return;
    interrupted = reason;
    signalGroup("SIGTERM");
    killTimer = setTimeout(() => signalGroup("SIGKILL"), 200);
  };
  const timer = setTimeout(() => stop("time-budget"), timeoutMs);
  function line(raw) {
    if (Buffer.byteLength(raw) > maxLineBytes) { stop("event-size-budget"); return; }
    try {
      const record = JSON.parse(raw);
      onRecord?.(record);
      const event = publicEvent(record);
      if (event) messages.push(event);
    } catch { stop("invalid-json-event"); }
  }
  child.stdout.setEncoding("utf8");
  child.stdout.on("data", chunk => {
    bytes += Buffer.byteLength(chunk);
    if (bytes + stderrBytes > maxStreamBytes) { stop("stream-byte-budget"); return; }
    buffer += chunk;
    let end;
    while ((end = buffer.indexOf("\n")) >= 0) { line(buffer.slice(0, end)); buffer = buffer.slice(end + 1); }
    if (Buffer.byteLength(buffer) > maxLineBytes) { buffer = ""; stop("event-size-budget"); }
  });
  child.stderr.on("data", chunk => { stderrBytes += chunk.length; if (bytes + stderrBytes > maxStreamBytes) stop("stream-byte-budget"); });
  if (onStart) {
    child.stdin.on("error", () => stop("control-io-error"));
    Promise.resolve().then(() => onStart(record => child.stdin.write(JSON.stringify(record) + "\n"), stop)).catch(() => stop("control-protocol-error"));
  }
  let spawnError = false;
  child.on("error", () => { spawnError = true; });
  const completion = await new Promise(resolve => child.on("close", (code, signal) => resolve({ code, signal })));
  clearTimeout(timer);
  if (buffer && !interrupted) line(buffer); // Accept a complete final record without a newline.
  // Kill any descendants even when their parent exited normally. Keep escalation
  // alive until after it fired; do not cancel it on the leader's early exit.
  signalGroup("SIGKILL");
  clearTimeout(killTimer);
  return { ...completion, spawnError, interrupted, stdoutBytes: bytes, stderrBytes, messages };
}
