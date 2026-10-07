import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdtemp, readFile, realpath, rm, stat } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";
import test from "node:test";

const repository = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const helper = join(repository, "tools", "prepare-skill-demo.mjs");
const git = (repo, ...args) => execFileSync("git", ["-C", repo, ...args], { encoding: "utf8" }).trim();

test("materializes a clean standalone demo with optional repo-local skill", async (t) => {
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-demo-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  for (const withSkill of [false, true]) {
    const destination = join(root, withSkill ? "with-skill" : "without-skill");
    const args = [helper, "--destination", destination, ...(withSkill ? ["--with-skill"] : [])];
    const result = JSON.parse(execFileSync(process.execPath, args, { encoding: "utf8" }));
    assert.equal(result.repository, destination);
    assert.equal(result.sourceCommit, git(destination, "rev-parse", "HEAD"));
    assert.equal(git(destination, "status", "--porcelain"), "");
    for (const file of ["README.md", "go.mod", "booking/service.go", "booking/service_test.go", "AGENTS.md", "docs/requirements.md"]) {
      assert.equal(
        await readFile(join(destination, file), "utf8"),
        await readFile(join(repository, "testdata", "booking-demo", file), "utf8"),
      );
      assert.equal(git(destination, "ls-files", "--", file), file);
    }
    await assert.rejects(stat(join(destination, ".agent")), { code: "ENOENT" });
    if (withSkill) {
      assert.equal(result.skill, join(destination, ".agents", "skills", "ctx", "SKILL.md"));
      for (const file of ["SKILL.md", "references/cli.md", "agents/openai.yaml"]) {
        assert.equal(
          await readFile(join(destination, ".agents", "skills", "ctx", file), "utf8"),
          await readFile(join(repository, "skills", "ctx", file), "utf8"),
        );
      }
    } else {
      assert.equal(result.skill, null);
      await assert.rejects(stat(join(destination, ".agents")), { code: "ENOENT" });
    }
  }
});

test("refuses an existing destination without overwriting data or committing", async (t) => {
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-demo-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const destination = join(root, "booking");
  execFileSync(process.execPath, [helper, "--destination", destination]);
  const before = await readFile(join(destination, "booking", "service.go"));
  const head = git(destination, "rev-parse", "HEAD");
  const result = spawnSync(process.execPath, [helper, "--destination", destination, "--with-skill"], { encoding: "utf8" });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Refusing existing destination/);
  assert.deepEqual(await readFile(join(destination, "booking", "service.go")), before);
  assert.equal(git(destination, "rev-parse", "HEAD"), head);
  assert.equal(git(destination, "status", "--porcelain"), "");
  await assert.rejects(stat(join(destination, ".agents")), { code: "ENOENT" });
});

test("refuses relative destinations before creating files", () => {
  const result = spawnSync(process.execPath, [helper, "--destination", "relative-demo"], { encoding: "utf8" });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /absolute, not-yet-existing/);
});
