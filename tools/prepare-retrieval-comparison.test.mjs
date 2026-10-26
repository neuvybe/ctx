import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdtemp, readFile, realpath, rm, stat, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const repository = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const helper = join(repository, "tools/prepare-retrieval-comparison.mjs");

test("prepares equivalent retrieval payloads in compatible flat and folder layouts without overwriting", async t => {
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-retrieval-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const cli = join(root, "ctx");
  execFileSync("go", ["build", "-o", cli, "./cmd/ctx"], { cwd: repository });
  const topics = JSON.parse(await readFile(join(repository, "evals/ctx-skill/fixtures/retrieval/topics.json"), "utf8"));
  const prepared = [];
  for (const layout of ["flat", "folder"]) {
    const destination = join(root, layout);
    const args = [helper, "--destination", destination, "--layout", layout, "--ctx", cli];
    const result = JSON.parse(execFileSync(process.execPath, args, { encoding: "utf8" }));
    prepared.push(result);
    for (const topic of topics) {
      const file = layout === "flat" ? topic.category + ".md" : topic.category + "/" + topic.name + ".md";
      const content = await readFile(join(destination, ".ctx/context", file), "utf8");
      assert.equal(content.split(topic.body).length, 2, "each payload occurs exactly once in its owning fact");
    }
    assert.equal(execFileSync("git", ["-C", destination, "diff", "--cached"], { encoding: "utf8" }), "");
    const before = await readFile(join(destination, ".ctx/config.json"));
    assert.notEqual(spawnSync(process.execPath, args, { encoding: "utf8" }).status, 0);
    assert.deepEqual(await readFile(join(destination, ".ctx/config.json")), before);
  }
  assert.deepEqual(prepared[0].sourceHashes, prepared[1].sourceHashes);
  assert.equal(prepared[0].semanticHash, prepared[1].semanticHash);
  assert.equal(prepared[0].skillHash, prepared[1].skillHash);
});

test("manual model runner rejects invalid and existing destinations before any provider run", async t => {
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-retrieval-runner-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const runner = join(repository, "tools/run-retrieval-comparison.mjs");
  const marker = join(root, "keep.txt");
  await writeFile(marker, "preserve evaluator data\n");
  for (const args of [[], ["--destination", "relative", "--ctx", "/missing-ctx"], ["--destination", root, "--ctx", "/missing-ctx"], ["--destination", join(root, "new"), "--ctx", "/missing-ctx"], ["--unknown", "value"]]) {
    const result = spawnSync(process.execPath, [runner, ...args], { encoding: "utf8" });
    assert.notEqual(result.status, 0);
    assert.deepEqual(await readFile(marker, "utf8"), "preserve evaluator data\n");
    await assert.rejects(stat(join(root, "new")), { code: "ENOENT" });
  }
  assert.equal(spawnSync(process.execPath, ["--check", runner]).status, 0);
});
