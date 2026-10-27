import { execFileSync } from "node:child_process";
import { lstat, mkdir, readdir, readFile, readlink, realpath, writeFile } from "node:fs/promises";
import { basename, dirname, isAbsolute, join, resolve } from "node:path";
import { digest, safePath, sensitivePath } from "./authoring-boundary.mjs";

// Git and ctx never receive provider credentials or ambient Git config overrides.
export function coordinatorEnv(env = process.env) {
  const result = {};
  for (const name of ["PATH", "HOME", "TMPDIR", "SystemRoot"]) if (env[name]) result[name] = env[name];
  return { ...result, LANG: "C", LC_ALL: "C", GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: process.platform === "win32" ? "NUL" : "/dev/null", GIT_TERMINAL_PROMPT: "0", GIT_OPTIONAL_LOCKS: "0" };
}

export function git(repo, ...args) {
  return execFileSync("git", ["-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", ...args], { env: coordinatorEnv(), timeout: 30000, maxBuffer: 64 * 1024 * 1024, stdio: ["ignore", "pipe", "pipe"] });
}

export async function freshDestination(requested) {
  if (!isAbsolute(requested) || resolve(requested) === "/") throw new Error("An absolute new destination is required");
  const path = join(await realpath(dirname(resolve(requested))), basename(resolve(requested)));
  try { await lstat(path); throw new Error("Refusing existing destination"); }
  catch (error) { if (error.code !== "ENOENT") throw error; }
  return path;
}

export async function treeManifest(root, { skipGit = false } = {}) {
  // Filenames are data, including names such as __proto__.
  const entries = Object.create(null);
  let bytes = 0, count = 0;
  async function visit(relative = "") {
    for (const name of (await readdir(join(root, relative))).sort()) {
      if (skipGit && !relative && name === ".git") continue;
      const path = relative ? relative + "/" + name : name;
      const info = await lstat(join(root, path));
      if (++count > 30000) throw new Error("Audit entry limit exceeded; use a clean dedicated source checkout");
      const entry = { mode: info.mode & 0o777, type: info.isDirectory() ? "directory" : info.isFile() ? "file" : info.isSymbolicLink() ? "symlink" : "special" };
      if (info.isDirectory()) await visit(path);
      else if (info.isSymbolicLink()) entry.targetHash = digest(await readlink(join(root, path)));
      else if (info.isFile()) {
        bytes += info.size;
        if (bytes > 128 * 1024 * 1024 || info.size > 32 * 1024 * 1024) throw new Error("Audit byte limit exceeded; use a clean dedicated source checkout");
        entry.hash = digest(await readFile(join(root, path)));
      } else throw new Error("Cannot audit special filesystem entries");
      entries[path] = entry;
    }
  }
  await visit();
  return entries;
}

export async function auditRepository(repo) {
  // Resolve through Git: a linked worktree's .git is a pointer, not a directory.
  const gitPath = path => git(repo, "rev-parse", "--path-format=absolute", "--git-path", path).toString().trim();
  const metadata = {};
  for (const path of ["index", "HEAD", "config", "info/exclude", "packed-refs"]) {
    try { metadata[path] = digest(await readFile(gitPath(path))); }
    catch (error) { if (error.code !== "ENOENT") throw error; metadata[path] = null; }
  }
  return {
    entries: await treeManifest(repo, { skipGit: true }),
    gitEntry: (await lstat(join(repo, ".git"))).isFile() ? digest(await readFile(join(repo, ".git"))) : "directory",
    head: git(repo, "rev-parse", "HEAD").toString().trim(),
    refs: digest(git(repo, "for-each-ref", "--format=%(refname) %(objectname)")),
    staged: digest(git(repo, "diff", "--cached", "--no-ext-diff", "--no-textconv", "--binary")),
    metadata,
  };
}

export async function frozenSources(repo, commit) {
  if (!/^(?:[0-9a-f]{40}|[0-9a-f]{64})$/.test(commit)) throw new Error("Pin a full immutable source commit");
  if (git(repo, "rev-parse", "--show-toplevel").toString().trim() !== repo) throw new Error("Repository must be its checkout root");
  if (git(repo, "rev-parse", "HEAD").toString().trim() !== commit) throw new Error("Source HEAD does not match the pinned commit");
  if (git(repo, "status", "--porcelain=v1", "--untracked-files=all").length) throw new Error("Source checkout must be clean, including its index");
  try { await lstat(join(repo, ".ctx")); throw new Error("Existing scaffolds are outside this new-authoring trial"); }
  catch (error) { if (error.code !== "ENOENT") throw error; }
  const files = Object.create(null), omitted = [];
  let total = 0;
  for (const record of git(repo, "ls-tree", "-rz", "--full-tree", commit).toString().split("\0").filter(Boolean)) {
    const separator = record.indexOf("\t");
    const header = record.slice(0, separator), path = record.slice(separator + 1);
    safePath(path);
    if (sensitivePath(path)) { omitted.push({ reason: "sensitive-path" }); continue; }
    const [mode, type, object] = header.split(" ");
    if (type !== "blob" || !["100644", "100755"].includes(mode)) { omitted.push({ path, reason: "link-or-submodule" }); continue; }
    if (Number(git(repo, "cat-file", "-s", object)) > 1024 * 1024) { omitted.push({ path, reason: "large-file" }); continue; }
    const data = git(repo, "cat-file", "blob", object);
    const text = data.toString("utf8");
    if (data.includes(0) || !Buffer.from(text).equals(data)) { omitted.push({ path, reason: "binary-file" }); continue; }
    total += data.length;
    if (total > 32 * 1024 * 1024 || Object.keys(files).length >= 10000) throw new Error("Source snapshot resource limit exceeded");
    files[path] = text;
  }
  if ([...Object.keys(files)].some(path => path === ".ctx" || path.startsWith(".ctx/"))) throw new Error("Existing scaffolds are outside this new-authoring trial");
  return { files, omitted };
}

export async function readSkill(root) {
  const manifest = await treeManifest(root);
  const files = Object.create(null);
  for (const [path, entry] of Object.entries(manifest)) {
    safePath(path);
    if (entry.type === "directory") continue;
    if (entry.type !== "file") throw new Error("Skill bundle must not contain links or special files");
    files[".agents/skills/ctx/" + path] = await readFile(join(root, path), "utf8");
  }
  if (!files[".agents/skills/ctx/SKILL.md"]) throw new Error("Missing ctx skill entrypoint");
  return files;
}

export async function putFiles(root, files) {
  // Coordinator-only, into an exclusively claimed private directory. Never used
  // against the target checkout or an agent-writable host directory.
  for (const [path, text] of Object.entries(files)) {
    safePath(path);
    await mkdir(dirname(join(root, path)), { recursive: true, mode: 0o700 });
    await writeFile(join(root, path), text, { flag: "wx", mode: 0o600 });
  }
}
