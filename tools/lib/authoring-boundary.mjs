// Trusted broker: model-controlled paths never reach the host filesystem.
import { createHash } from "node:crypto";

export const digest = value => createHash("sha256").update(value).digest("hex");
export const toolNames = ["ctx_read", "ctx_list", "ctx_search", "ctx_write", "ctx_source_info"];
const categories = "overview|architecture|behavior|caveats|glossary";
const writable = new RegExp(`^\\.ctx/context/(${categories})/(?:[a-z0-9][a-z0-9-]*/)*(?:index|[a-z0-9][a-z0-9-]*)\\.md$`);

export function safePath(value, { root = false } = {}) {
  if (root && value === ".") return value;
  if (typeof value !== "string" || !value || value.length > 512 || /[\\\x00-\x1f\x7f:]/u.test(value) || value.startsWith("/") ||
      value.split("/").some(part => !part || part === "." || part === ".." || part.toLowerCase() === ".git")) {
    throw new Error("Noncanonical or forbidden repository-relative path");
  }
  return value;
}

export function sensitivePath(path) {
  const parts = path.split("/");
  return parts.some((part, index) => /^(?:\.git|\.aws|\.ssh|\.codex|\.claude|\.pi|\.netrc|\.npmrc|\.pypirc|\.git-credentials|node_modules)$/i.test(part) ||
    /^\.env(?:\.|$)/i.test(part) || /\.(?:pem|key|p12|pfx)$/i.test(part) ||
    // Authentication modules are evidence, not credential stores. Limit the
    // exception to source filenames; auth.json and sensitive ancestors stay out.
    (/^(?:auth|credentials|secrets?)(?:\.|$)/i.test(part) &&
      !(index === parts.length - 1 && /\.(?:[cm]?[jt]sx?|go|py|rs|java|kt|rb|c|cc|cpp|h|hpp|cs|swift)$/i.test(part))));
}

export function mayWrite(path) {
  return path === ".ctx/INDEX.md" || path === ".ctx/local/CONTINUE.md" || writable.test(path);
}

function keys(input, allowed, required = allowed) {
  if (!input || typeof input !== "object" || Array.isArray(input) || Object.keys(input).some(key => !allowed.includes(key)) ||
      required.some(key => !Object.hasOwn(input, key))) throw new Error("Invalid tool arguments");
}

export function createBoundary({ files, sourceCommit, sourceOmissions, readOnly = false, budgets, clock = Date.now }) {
  if (typeof readOnly !== "boolean") throw new Error("Invalid read-only selection");
  // Copy inputs so a caller cannot mutate the source view through an alias.
  const view = new Map(Object.entries(files));
  const omissions = sourceOmissions === undefined ? null : structuredClone(sourceOmissions);
  if (omissions !== null && (!omissions || typeof omissions !== "object" || Array.isArray(omissions) ||
      Object.entries(omissions).some(([reason, count]) => !["sensitive-path", "link-or-submodule", "large-file", "binary-file"].includes(reason) || !Number.isSafeInteger(count) || count < 1))) throw new Error("Invalid source omission counts");
  for (const [path, text] of view) {
    safePath(path);
    if (sensitivePath(path) || typeof text !== "string") throw new Error("Unsafe source view");
  }
  for (const name of ["timeoutMs", "maxCalls", "maxOutputBytes", "maxWriteBytes"]) {
    if (!Number.isSafeInteger(budgets[name]) || budgets[name] < 1) throw new Error("Positive integer budgets required");
  }
  const writes = new Map();
  const reads = new Set();
  const readLines = new Map();
  const events = [];
  let calls = 0, outputBytes = 0, writeBytes = 0, closed = false;
  const started = clock();
  const nowFiles = () => new Map([...view, ...writes]);
  const requiredReads = [".agents/skills/ctx/SKILL.md", ".ctx/INDEX.md", ".ctx/local/CONTINUE.md", ...["AGENTS.md", "CLAUDE.md"].filter(path => view.has(path))];

  function execute(tool, input) {
    if (closed) throw new Error("Authoring boundary is closed");
    if (clock() - started >= budgets.timeoutMs || calls >= budgets.maxCalls) {
      closed = true;
      throw new Error("Authoring time or tool-call budget exhausted");
    }
    calls++;
    const event = { tool: toolNames.includes(tool) ? tool : "unauthorized-tool", outcome: "denied", returnedBytes: 0 };
    events.push(event);
    try {
      let result;
      if (tool === "ctx_source_info") {
        keys(input, []);
        result = JSON.stringify({ sourceCommit, inventoryComplete: false,
          availableSourcePaths: [...view.keys()].filter(path => !path.startsWith(".ctx/") && !path.startsWith(".agents/skills/ctx/")).sort(),
          omittedFileCounts: omissions,
          limitation: "Filtered available view, not the complete Git inventory. An unavailable path is not evidence that it is absent from the repository. Sensitive filenames and contents are withheld." });
      } else if (tool === "ctx_read") {
        keys(input, ["path", "offset", "limit"], ["path"]);
        const path = safePath(input.path);
        const text = nowFiles().get(path);
        if (text === undefined) throw new Error("Path is not in this trial's source view");
        const offset = input.offset ?? 1, limit = input.limit ?? 200;
        if (!Number.isSafeInteger(offset) || offset < 1 || !Number.isSafeInteger(limit) || limit < 1 || limit > 400) throw new Error("Invalid line range");
        const lines = text.split("\n");
        result = JSON.stringify({ path, hash: digest(text), totalLines: lines.length, offset, lines: lines.slice(offset - 1, offset - 1 + limit) });
        // A partial read is observable but does not claim the complete entrypoint was read.
        const covered = readLines.get(path) || new Set();
        for (let line = offset; line < offset + limit && line <= lines.length; line++) covered.add(line);
        readLines.set(path, covered);
        if (covered.size === lines.length) reads.add(path);
        event.path = path;
      } else if (tool === "ctx_list" || tool === "ctx_search") {
        keys(input, tool === "ctx_list" ? ["path"] : ["path", "query"], tool === "ctx_list" ? ["path"] : ["path", "query"]);
        const path = safePath(input.path, { root: true });
        const candidates = [...nowFiles()].filter(([name]) => path === "." || name === path || name.startsWith(path + "/")).sort(([a], [b]) => a.localeCompare(b));
        if (tool === "ctx_list") result = JSON.stringify(candidates.map(([name]) => name));
        else {
          if (typeof input.query !== "string" || !input.query || input.query.length > 256) throw new Error("A short literal search query is required");
          const hits = [];
          // Literal search only: no regular expressions, shell expansion, or subprocess.
          search: for (const [name, text] of candidates) for (const [line, value] of text.split("\n").entries()) {
            if (value.includes(input.query)) hits.push({ path: name, line: line + 1, text: value.slice(0, 1000) });
            if (hits.length >= 100) break search;
          }
          result = JSON.stringify(hits.slice(0, 100));
        }
        event.path = path;
      } else if (tool === "ctx_write") {
        if (readOnly) throw new Error("This reader trial is read-only; context writes are forbidden");
        keys(input, ["path", "text", "expectedHash"]);
        const path = safePath(input.path);
        if (!mayWrite(path)) throw new Error("Only context facts, routing, and local continuation are editable");
        if (requiredReads.some(path => !reads.has(path))) throw new Error("Read the complete skill, owner instructions, router, and continuation before authoring");
        if (typeof input.text !== "string" || input.text.includes("\0")) throw new Error("Context must be plain text");
        const current = nowFiles().get(path);
        if (input.expectedHash !== (current === undefined ? null : digest(current))) throw new Error("Context preimage does not match");
        if ([...nowFiles().keys()].some(name => name !== path && (name.startsWith(path + "/") || path.startsWith(name + "/")))) throw new Error("File and directory paths conflict");
        const bytes = Buffer.byteLength(input.text);
        if (bytes > 128 * 1024 || writeBytes + bytes > budgets.maxWriteBytes) { closed = true; throw new Error("Context write budget exhausted"); }
        result = JSON.stringify({ path, hash: digest(input.text), bytes });
        // Reserve output bytes before mutating even the virtual proposal.
        if (outputBytes + Buffer.byteLength(result) > budgets.maxOutputBytes) { closed = true; throw new Error("Tool output budget exhausted"); }
        writes.set(path, input.text);
        writeBytes += bytes;
        event.path = path;
      } else throw new Error("Tool is not authorized");
      const bytes = Buffer.byteLength(result);
      if (outputBytes + bytes > budgets.maxOutputBytes) { closed = true; throw new Error("Tool output budget exhausted"); }
      outputBytes += bytes;
      event.outcome = "allowed";
      event.returnedBytes = bytes;
      return result;
    } catch (error) {
      // Never save arbitrary arguments, document contents, or exception strings.
      // All broker validation errors are fixed descriptions, without interpolation
      // of model input or source text. Preserve them so the author can recover.
      throw new Error(error.message);
    }
  }
  return {
    execute,
    exhausted() { return closed || calls >= budgets.maxCalls || clock() - started >= budgets.timeoutMs; },
    close() { closed = true; },
    snapshot() { return { writes: Object.fromEntries(writes), events: structuredClone(events), fullReads: [...reads].sort(), calls, outputBytes, writeBytes }; },
  };
}

// The coordinator replays every write before accepting a runtime's proposal.
export function validateProposal(input, proposal) {
  if (!proposal?.writes || typeof proposal.writes !== "object" || Array.isArray(proposal.writes)) throw new Error("Invalid proposal");
  let bytes = 0;
  const paths = new Set(Object.keys(input.files));
  for (const [path, text] of Object.entries(proposal.writes)) {
    safePath(path);
    if (!mayWrite(path) || typeof text !== "string" || text.includes("\0") || Buffer.byteLength(text) > 128 * 1024) throw new Error("Invalid context proposal");
    bytes += Buffer.byteLength(text);
    paths.add(path);
  }
  if (bytes > input.budgets.maxWriteBytes) throw new Error("Proposal exceeds write budget");
  for (const path of paths) {
    const parts = path.split("/");
    for (let count = 1; count < parts.length; count++) if (paths.has(parts.slice(0, count).join("/"))) throw new Error("Conflicting proposal paths");
  }
  return { ...input.files, ...proposal.writes };
}
