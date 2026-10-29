import assert from "node:assert/strict";
import test from "node:test";
import { reopenClient } from "./entry-store.mjs";

test("a reopened client shares the same in-memory store", () => {
  const store = new Map([["entry", "published"]]);
  const first = reopenClient(store);
  const second = reopenClient(first.store);
  assert.equal(second.store.get("entry"), "published");
});
