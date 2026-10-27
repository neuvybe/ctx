// Loaded explicitly by the coordinator, never discovered from the subject repo.
import { readFileSync, writeFileSync } from "node:fs";
import { createBoundary, toolNames } from "./lib/authoring-boundary.mjs";

export const schemas = {
  ctx_read: { path: { type: "string" }, offset: { type: "integer" }, limit: { type: "integer" } },
  ctx_list: { path: { type: "string" } },
  ctx_search: { path: { type: "string" }, query: { type: "string" } },
  ctx_write: { path: { type: "string" }, text: { type: "string" }, expectedHash: { type: ["string", "null"] } },
  ctx_source_info: {},
};
export const descriptions = {
  ctx_read: "Read a frozen source or proposed context file by repository-relative path, with one-based offset and up to 400 lines. Returns content hash for writes.",
  ctx_list: "List this trial's frozen source and proposed context paths under a repository-relative path; use . for the root.",
  ctx_search: "Literal text search within this trial's frozen source and proposed context. No regex or shell syntax.",
  ctx_write: "Propose a complete context fact, INDEX, or local continuation. expectedHash must equal its latest read hash, or null for a new file. Cannot change config, mode, owner instructions, or source.",
  ctx_source_info: "Return the pinned source commit and available source paths. Source is clean and immutable; no Git command is needed.",
};

export function qualifiedThinking(requested, effective) {
  if (!["off", "minimal", "low", "medium", "high", "xhigh", "max"].includes(effective) ||
      (requested != null && requested !== effective)) throw new Error("Pi changed the requested thinking setting; choose a supported explicit setting");
  return effective;
}

export default function authoringExtension(pi) {
  const input = JSON.parse(readFileSync(process.env.CTX_AUTHORING_INPUT, "utf8"));
  const boundary = createBoundary(input);
  let authorized = false, effectiveThinking = null;
  const save = () => writeFileSync(process.env.CTX_AUTHORING_RESULT, JSON.stringify({ ...boundary.snapshot(), effectiveThinking }), { mode: 0o600 });
  for (const name of toolNames) {
    pi.registerTool({
      name, label: name, description: descriptions[name],
      parameters: { type: "object", properties: schemas[name], required: Object.keys(schemas[name]).filter(key => !["offset", "limit"].includes(key)), additionalProperties: false },
      async execute(_id, args, _signal, _onUpdate, ctx) {
        if (!authorized) throw new Error("Pi tool boundary did not pass startup verification");
        try { return { content: [{ type: "text", text: boundary.execute(name, args) }], details: undefined }; }
        finally { save(); if (boundary.exhausted()) ctx.abort(); }
      },
    });
  }
  pi.on("session_start", () => {
    // Fail before any provider call if a CLI change/config adds another tool.
    authorized = false;
    pi.setActiveTools([]);
    const actual = pi.getAllTools().map(tool => tool.name).sort();
    if (JSON.stringify(actual) !== JSON.stringify([...toolNames].sort())) throw new Error("Unexpected Pi tools; authoring refused");
    effectiveThinking = qualifiedThinking(input.thinking, pi.getThinkingLevel());
    pi.setActiveTools(toolNames);
    authorized = true;
    save();
  });
  pi.on("tool_call", event => {
    if (!authorized || !toolNames.includes(event.toolName)) return { block: true, reason: "Only the guarded authoring tools are authorized" };
  });
  pi.on("session_shutdown", () => { boundary.close(); save(); });
}
