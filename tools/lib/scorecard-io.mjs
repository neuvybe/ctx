import { lstat, readFile, writeFile } from "node:fs/promises";
import { isAbsolute, join } from "node:path";

export async function readScorecardJson(path) {
  if (!isAbsolute(path || "")) throw new Error("Absolute scorecard paths required");
  const info = await lstat(path);
  if (!info.isFile() || info.size > 64 * 1024 * 1024) throw new Error("Scorecard data must be a bounded regular JSON file");
  return JSON.parse(await readFile(path, "utf8"));
}

export async function readFrozenTrial(directory) {
  if (!isAbsolute(directory || "")) throw new Error("Absolute trial directory required");
  const entries = await Promise.all(["input", "manifest", "result"].map(async name => [name, await readScorecardJson(join(directory, name + ".json"))]));
  const trial = Object.fromEntries(entries);
  try { trial.attempt = await readScorecardJson(join(directory, "attempt.json")); }
  catch (error) { if (error.code !== "ENOENT") throw error; }
  return trial;
}

export async function writeScorecardJson(path, value) {
  if (!isAbsolute(path || "")) throw new Error("Absolute new report path required");
  await writeFile(path, JSON.stringify(value, null, 2) + "\n", { flag: "wx", mode: 0o600 });
}
