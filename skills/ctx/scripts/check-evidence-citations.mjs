#!/usr/bin/env node
// Checks explicit citation bookkeeping only, never semantic verification.
import { lstat, readFile, readdir, realpath } from "node:fs/promises";
import { isAbsolute, join, resolve } from "node:path";
import { pathToFileURL } from "node:url";

function safeRelative(value) {
  return typeof value === "string" && value !== "" && !isAbsolute(value) &&
    !value.includes("\\") && !value.includes("\0") &&
    !value.split("/").some(part => ["", ".", "..", ".git"].includes(part));
}

async function inspectPath(root, value) {
  if (!safeRelative(value)) throw new Error("requires a safe repository-relative path");
  let current = root;
  let info;
  for (const part of value.split("/")) {
    current = join(current, part);
    info = await lstat(current);
    if (info.isSymbolicLink()) throw new Error("symbolic links are not followed");
  }
  return info;
}

export async function checkEvidenceCitations(repository, folder) {
  const root = await realpath(repository);
  if (!safeRelative(folder)) throw new Error("--folder requires a safe repository-relative path");
  const context = folder + "/context";
  if (!(await inspectPath(root, context)).isDirectory()) throw new Error("context must be a directory");
  const documents = [];
  async function visit(directory) {
    for (const entry of (await readdir(join(root, directory), { withFileTypes: true })).sort((a, b) => a.name.localeCompare(b.name))) {
      const file = directory + "/" + entry.name;
      if (entry.isSymbolicLink()) throw new Error("symbolic links are not followed: " + file);
      if (entry.isDirectory()) await visit(file);
      else if (entry.isFile() && entry.name.endsWith(".md")) documents.push(file);
    }
  }
  await visit(context);
  const results = [];
  for (const document of documents) {
    const text = await readFile(join(root, document), "utf8");
    const citations = [];
    let fence = null;
    for (const [index, line] of text.split(/\r?\n/).entries()) {
      const delimiter = line.match(/^\s{0,3}(`{3,}|~{3,})/);
      if (delimiter) {
        if (!fence) fence = delimiter[1];
        else if (delimiter[1][0] === fence[0] && delimiter[1].length >= fence.length && !line.slice(delimiter[0].length).trim()) fence = null;
        continue;
      }
      if (fence) continue;
      const citation = line.match(/^\s*(?:[-*]\s+)?(?:\*\*)?Evidence source:(?:\*\*)?\s*`([^`\n]+)`/);
      if (citation) citations.push({ path: citation[1], line: index + 1 });
    }
    const issues = [];
    let sources = [];
    if (citations.length) {
      try {
        const markers = [...text.matchAll(/^\s*<!-- ctx:doc (.*?) -->\s*$/gm)];
        if (markers.length !== 1) throw new Error("requires exactly one metadata comment");
        const metadata = JSON.parse(markers[0][1]);
        if (!Array.isArray(metadata.sources) || !metadata.sources.every(safeRelative)) throw new Error("metadata sources must be safe repository-relative paths");
        sources = metadata.sources;
      } catch (error) { issues.push({ detail: "metadata: " + error.message }); }
      for (const citation of citations) {
        try {
          if (citation.path === folder || citation.path.startsWith(folder + "/")) throw new Error("context cannot be its own supporting evidence");
          const info = await inspectPath(root, citation.path);
          if (!info.isFile()) throw new Error("explicit citations must name regular files");
          let covered = sources.includes(citation.path);
          if (!covered) {
            for (const source of sources) {
              if (!citation.path.startsWith(source + "/")) continue;
              if ((await inspectPath(root, source)).isDirectory()) { covered = true; break; }
            }
          }
          if (!covered) issues.push({ ...citation, detail: "not covered by metadata sources" });
        } catch (error) { issues.push({ ...citation, detail: error.code === "ENOENT" ? "source path does not exist" : error.message }); }
      }
    }
    results.push({ document, citations, issues });
  }
  return {
    repository: root, folder, checkedCitations: results.reduce((count, result) => count + result.citations.length, 0),
    scope: "Explicit Evidence source lines only; not semantic verification, Git freshness, or completeness of prose citations",
    documents: results, ok: results.every(result => result.issues.length === 0),
  };
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  try {
    const options = new Map();
    const args = process.argv.slice(2);
    for (let index = 0; index < args.length; index += 2) {
      if (!["--repo", "--folder"].includes(args[index]) || !args[index + 1] || options.has(args[index])) throw new Error("Usage: --repo /absolute/repository --folder .ctx");
      options.set(args[index], args[index + 1]);
    }
    if (!isAbsolute(options.get("--repo") || "")) throw new Error("--repo must be absolute");
    const report = await checkEvidenceCitations(options.get("--repo"), options.get("--folder") || ".ctx");
    console.log(JSON.stringify(report, null, 2));
    process.exitCode = report.ok ? 0 : 1;
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
