// Trusted transport-only data. Never include returned credentials in artifacts,
// tool results, command arguments, or exception messages.
import { lstat, readFile } from "node:fs/promises";
import { isAbsolute } from "node:path";

async function privateJson(path) {
  if (!isAbsolute(path)) throw new Error("An explicit absolute credential path is required");
  const info = await lstat(path);
  if (!info.isFile() || info.size > 1024 * 1024) throw new Error("Credential source must be a small regular file");
  try { return JSON.parse(await readFile(path, "utf8")); }
  catch { throw new Error("Credential source could not be parsed; details omitted"); }
}

export async function selectedPiCredential(path, provider, minValidityMs, now = Date.now()) {
  const data = await privateJson(path);
  const credential = data?.[provider];
  if (!credential || typeof credential !== "object") throw new Error("Selected provider has no stored credential");
  if (credential.type === "oauth") {
    if (!["access", "refresh"].every(key => typeof credential[key] === "string" && credential[key]) || !Number.isFinite(credential.expires) || credential.expires < now + minValidityMs) throw new Error("Stored OAuth credential needs renewal outside the runner");
    const selected = { type: "oauth", access: credential.access, refresh: credential.refresh, expires: credential.expires };
    for (const key of ["accountId", "projectId"]) if (typeof credential[key] === "string") selected[key] = credential[key];
    return selected;
  }
  if (credential.type === "api_key" && typeof credential.key === "string" && credential.key && !credential.key.startsWith("!") && !credential.env && !/^[A-Z][A-Z0-9_]*$/.test(credential.key)) return { type: "api_key", key: credential.key };
  throw new Error("Stored credential requires an unsupported resolver; no helper was executed");
}

export function readOnlyCredentialStore(provider, credential) {
  // Exactly one provider is visible to Pi. Refresh/login/logout are prohibited;
  // even a runtime bug cannot persist a change through this store's interface.
  return {
    async read(requested) { return requested === provider ? structuredClone(credential) : undefined; },
    async list() { return [{ providerId: provider, type: credential.type }]; },
    async modify() { throw new Error("Credential mutation and refresh are not authorized"); },
    async delete() { throw new Error("Credential mutation and logout are not authorized"); },
  };
}

export async function selectedCodexAuth(path, minValidityMs, now = Date.now()) {
  const data = await privateJson(path);
  // Do not silently turn ChatGPT-plan trials into separately billed API calls.
  const tokens = data?.tokens;
  if (data?.OPENAI_API_KEY || !tokens?.access_token || typeof tokens.account_id !== "string") throw new Error("Expected an existing file-backed Codex ChatGPT login");
  let claims;
  try { claims = JSON.parse(Buffer.from(tokens.access_token.split(".")[1], "base64url").toString("utf8")); }
  catch { throw new Error("Stored access-token expiry could not be established"); }
  // This is a local expiry gate, not signature validation or proof of access.
  if (!Number.isFinite(claims.exp) || claims.exp * 1000 < now + minValidityMs) throw new Error("Stored Codex login needs renewal outside the runner");
  return { type: "chatgptAuthTokens", accessToken: tokens.access_token, chatgptAccountId: tokens.account_id };
}
