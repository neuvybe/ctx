import assert from "node:assert/strict";
import { mkdtemp, mkdir, readFile, rm, symlink, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import test from "node:test";
import { checkEvidenceCitations } from "../skills/ctx/scripts/check-evidence-citations.mjs";

async function fixture(t, sources, body) {
  const root = await mkdtemp(join(tmpdir(), "ctx-citations-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  await mkdir(join(root, ".agent/context/topic"), { recursive: true });
  await mkdir(join(root, "src"));
  await writeFile(join(root, "src/logic.go"), "package src\n");
  await writeFile(join(root, "go.mod"), "module example.invalid/citations\n");
  const document = join(root, ".agent/context/topic/behavior.md");
  await writeFile(document, "# Behavior\n\n<!-- ctx:doc " + JSON.stringify({ status: "draft", verifiedAt: "", sources }) + " -->\n\n" + body);
  return { root, document };
}

test("recognizes explicit citations, including directory coverage, without changing files", async t => {
  const { root, document } = await fixture(t, ["src", "go.mod"], "Evidence source: `src/logic.go` — implementation\n- **Evidence source:** `go.mod` — module declaration\n");
  const before = await readFile(document);
  const result = await checkEvidenceCitations(root, ".agent");
  assert.equal(result.ok, true);
  assert.equal(result.checkedCitations, 2);
  assert.deepEqual(result.documents[0].issues, []);
  assert.deepEqual(await readFile(document), before);
});

test("reports a supporting path omitted from metadata rather than certifying prose", async t => {
  const { root } = await fixture(t, ["src/logic.go"], "Evidence source: `go.mod` — module declaration\n");
  const result = await checkEvidenceCitations(root, ".agent");
  assert.equal(result.ok, false);
  assert.deepEqual(result.documents[0].issues, [{ path: "go.mod", line: 5, detail: "not covered by metadata sources" }]);
  assert.match(result.scope, /not semantic verification/);
});

test("ignores arbitrary prose and fenced examples without claiming citation completeness", async t => {
  const { root } = await fixture(t, [], "`go.mod` is mentioned in ordinary prose.\n~~~markdown\nEvidence source: `missing.go`\n~~~\n````markdown\nEvidence source: `other.go`\n````\n");
  const result = await checkEvidenceCitations(root, ".agent");
  assert.equal(result.checkedCitations, 0);
  assert.equal(result.ok, true);
  assert.match(result.scope, /not.*completeness/);
});

test("rejects traversal, context self-citation, missing paths, and symlink evidence", async t => {
  const { root } = await fixture(t, ["src/logic.go"], "Evidence source: `../outside.go`\nEvidence source: `.agent/context/topic/behavior.md`\nEvidence source: `missing.go`\nEvidence source: `src/link.go`\n");
  await symlink(join(root, "go.mod"), join(root, "src/link.go"));
  const result = await checkEvidenceCitations(root, ".agent");
  assert.equal(result.ok, false);
  assert.equal(result.documents[0].issues.length, 4);
  assert.match(result.documents[0].issues[0].detail, /safe repository-relative/);
  assert.match(result.documents[0].issues[1].detail, /own supporting evidence/);
  assert.match(result.documents[0].issues[2].detail, /does not exist/);
  assert.match(result.documents[0].issues[3].detail, /symbolic links/);
  await assert.rejects(checkEvidenceCitations(root, "../other"), /safe repository-relative/);
});

test("refuses symlinked context and reports invalid metadata for recognized citations", async t => {
  const { root, document } = await fixture(t, [], "Evidence source: `src/logic.go`\n");
  await writeFile(document, "<!-- ctx:doc invalid -->\nEvidence source: `src/logic.go`\n");
  const result = await checkEvidenceCitations(root, ".agent");
  assert.equal(result.ok, false);
  assert.match(result.documents[0].issues[0].detail, /metadata:/);
  await symlink(join(root, ".agent/context"), join(root, "linked-context"));
  await assert.rejects(checkEvidenceCitations(root, "linked-context"), /symbolic links/);
});
