// Public synthetic evidence for scorecard calibration, not application code.
export const storage = "memory";
export const owner = "entry store";

export function requiresRevision(operation) {
  return ["update", "delete", "publish"].includes(operation);
}

export function publish(entry, revision) {
  if (!revision) throw new Error("Revision required");
  if (entry.state === "published") return entry;
  if (revision !== entry.revision) throw new Error("Stale revision");
  return { ...entry, state: "published", revision: "next" };
}

export function classify(kind, hasContent) {
  if (kind !== "entry") return "unsupported";
  if (!hasContent) return "inaccessible";
  return "eligible";
}

export function outcome(reason) {
  return reason === "origin_changed" ? "needs-review" : "unresolved";
}

export function reopenClient(store) {
  return { store };
}
