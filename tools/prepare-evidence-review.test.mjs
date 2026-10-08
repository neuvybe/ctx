import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdtemp, readFile, readdir, realpath, rm, stat } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";
import test from "node:test";

const repository = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const helper = join(repository, "tools/prepare-evidence-review.mjs");
const git = (repo, ...args) => execFileSync("git", ["-C", repo, ...args], { encoding: "utf8" }).trim();

test("prepares a review fixture with passing application and metadata checks without staging context", async t => {
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-evidence-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const cli = join(root, "ctx");
  execFileSync("go", ["build", "-o", cli, "./cmd/ctx"], { cwd: repository });
  const destination = join(root, "booking");
  const args = [helper, "--destination", destination, "--ctx", cli];
  const prepared = JSON.parse(execFileSync(process.execPath, args, { encoding: "utf8" }));
  assert.equal(prepared.repository, destination);
  assert.equal(prepared.sourceCommit, git(destination, "rev-parse", "HEAD"));
  assert.equal(git(destination, "diff"), "");
  assert.equal(git(destination, "diff", "--cached"), "");
  assert.equal(git(destination, "status", "--porcelain"), "?? .agent/");

  for (const file of ["README.md", "go.mod", "AGENTS.md", "docs/requirements.md", "booking/service.go", "booking/service_test.go"]) {
    assert.deepEqual(await readFile(join(destination, file)), await readFile(join(repository, "testdata/booking-demo", file)));
  }
  const config = JSON.parse(await readFile(join(destination, ".agent/config.json"), "utf8"));
  assert.equal(config.mode, "team");
  assert.deepEqual(config.addons, ["behavior", "glossary"]);
  const facts = (await readdir(join(destination, ".agent/context"))).sort();
  assert.deepEqual(facts, ["architecture.md", "behavior.md", "caveats.md", "glossary.md", "overview.md"]);
  for (const file of facts) {
    const content = await readFile(join(destination, ".agent/context", file), "utf8");
    const expected = (await readFile(join(repository, "evals/ctx-skill/fixtures/evidence-review", file), "utf8"))
      .replaceAll("{{SOURCE_COMMIT}}", prepared.sourceCommit).replaceAll("{{VERIFIED_DATE}}", prepared.verifiedOn);
    assert.equal(content, expected);
    const markers = [...content.matchAll(/<!-- ctx:doc (.*?) -->/g)];
    assert.equal(markers.length, 1);
    const metadata = JSON.parse(markers[0][1]);
    assert.equal(metadata.status, "verified");
    for (const source of metadata.sources) {
      assert.equal(git(destination, "show", prepared.sourceCommit + ":" + source), (await readFile(join(destination, source), "utf8")).trim());
    }
  }
  // Omitted supporting evidence is deliberate: mechanical readiness still
  // passes because status cannot decide whether prose has sufficient sources.
  const overview = await readFile(join(destination, ".agent/context/overview.md"), "utf8");
  assert.equal(JSON.parse(overview.match(/<!-- ctx:doc (.*?) -->/)[1]).sources.includes("go.mod"), false);
  assert.equal(git(destination, "check-ignore", ".agent/local/CONTINUE.md"), ".agent/local/CONTINUE.md");
  for (const command of ["doctor", "status"]) execFileSync(cli, [command, destination, "--folder", ".agent"]);
  execFileSync("go", ["test", "./..."], { cwd: destination });

  const contextBefore = await readFile(join(destination, ".agent/context/behavior.md"));
  const refused = spawnSync(process.execPath, args, { encoding: "utf8" });
  assert.notEqual(refused.status, 0);
  assert.match(refused.stderr, /Refusing existing destination/);
  assert.deepEqual(await readFile(join(destination, ".agent/context/behavior.md")), contextBefore);
  assert.equal(git(destination, "rev-parse", "HEAD"), prepared.sourceCommit);
  assert.equal(git(destination, "diff", "--cached"), "");
});

test("rejects invalid paths and unavailable CLI before creating a destination", async t => {
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-evidence-invalid-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const destination = join(root, "booking");
  for (const args of [
    ["--destination", destination, "--ctx", "relative-ctx"],
    ["--destination", destination, "--ctx", join(root, "missing-ctx")],
    ["--destination", destination, "--ctx", join(root, "missing-ctx"), "--unknown", "value"],
    ["--destination", destination, "--ctx", join(root, "missing-ctx"), "--fixture", "unknown-demo"],
  ]) {
    const result = spawnSync(process.execPath, [helper, ...args], { encoding: "utf8" });
    assert.notEqual(result.status, 0);
    await assert.rejects(stat(destination), { code: "ENOENT" });
  }
});

test("prepares the settings transfer fixture with a real process test and a detectable citation gap", async t => {
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-settings-review-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const cli = join(root, "ctx");
  execFileSync("go", ["build", "-o", cli, "./cmd/ctx"], { cwd: repository });
  const destination = join(root, "settings");
  const args = [helper, "--destination", destination, "--ctx", cli, "--fixture", "settings-demo"];
  const prepared = JSON.parse(execFileSync(process.execPath, args, { encoding: "utf8" }));
  for (const file of ["README.md", "go.mod", "AGENTS.md", "docs/requirements.md", "docs/storage.md", "settings/store.go", "settings/store_test.go"]) {
    assert.deepEqual(await readFile(join(destination, file)), await readFile(join(repository, "testdata/settings-demo", file)));
  }
  assert.equal(git(destination, "rev-parse", "HEAD"), prepared.sourceCommit);
  assert.equal(git(destination, "diff"), "");
  assert.equal(git(destination, "diff", "--cached"), "");
  assert.equal(git(destination, "status", "--porcelain"), "?? .agent/");
  for (const command of ["doctor", "status"]) execFileSync(cli, [command, destination, "--folder", ".agent"]);
  execFileSync("go", ["test", "-race", "./..."], { cwd: destination });
  const check = spawnSync(process.execPath, [join(destination, ".agents/skills/ctx/scripts/check-evidence-citations.mjs"), "--repo", destination, "--folder", ".agent"], { encoding: "utf8" });
  assert.equal(check.status, 1);
  const report = JSON.parse(check.stdout);
  assert.equal(report.checkedCitations, 14);
  assert.deepEqual(report.documents.flatMap(result => result.issues).map(issue => [issue.path, issue.detail]), [["docs/storage.md", "not covered by metadata sources"]]);
  const before = await readFile(join(destination, ".agent/context/behavior.md"));
  assert.notEqual(spawnSync(process.execPath, args, { encoding: "utf8" }).status, 0);
  assert.deepEqual(await readFile(join(destination, ".agent/context/behavior.md")), before);
});
