import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { lstat, mkdtemp, mkdir, open, readFile, realpath, rm, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { digest } from "./lib/authoring-boundary.mjs";
import { candidateFiles, candidateFingerprint, evaluateScorecard, fingerprint, reviewScopeDocuments, parentSummaryDocuments } from "./lib/context-scorecard.mjs";
import { readFrozenTrial, readScorecardJson, writeScorecardJson } from "./lib/scorecard-io.mjs";
import { calibrateScorecard } from "./calibrate-context-scorecard.mjs";
import { calibrationTrial, calibrationRubric, calibrationReview, calibrationComparison, calibrationParentSummaries, statements, bindComparisonRecords, misleadingReaderCitation } from "../evals/ctx-skill/fixtures/scorecard/calibration.mjs";
import { calibrationCases } from "../evals/ctx-skill/fixtures/scorecard/cases.mjs";

const fact = ".ctx/context/behavior/index.md";
const evaluate = trial => { const rubric = calibrationRubric(); return evaluateScorecard(trial, { rubric, review: calibrationReview(trial, rubric), comparison: calibrationComparison(trial) }); };
const statuses = report => Object.fromEntries(Object.entries(report.stages).map(([name, stage]) => [name, stage.verdict]));
function editMetadata(trial, transform, document = fact) {
  trial.attempt.writes[document] = trial.attempt.writes[document].replace(/<!-- ctx:doc (.*?) -->/, (_, json) => "<!-- ctx:doc " + JSON.stringify(transform(JSON.parse(json))) + " -->");
}
async function temporary(t) {
  const root = await mkdtemp(join(await realpath(tmpdir()), "ctx-scorecard-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  return root;
}
async function frozenDirectory(root, trial) {
  const path = join(root, "trial");
  await mkdir(path);
  for (const name of ["input", "manifest", "attempt", "result"]) if (trial[name]) await writeScorecardJson(join(path, name + ".json"), trial[name]);
  return path;
}

test("all named positive, incorrect, missing, and unknown controls match their four independent expected verdicts", async t => {
  const cases = await calibrationCases();
  assert.equal(cases.length, 50);
  for (const item of cases) await t.test(item.title, () => {
    const before = fingerprint(item);
    const report = evaluateScorecard(item.trial, item);
    assert.deepEqual(statuses(report), item.expected);
    assert.equal(fingerprint(item), before, "Grading must not mutate any frozen input");
    assert.equal(report.noCompositeScore, true);
    if (item.expected.supportedWording !== "passed" || item.expected.successfulExecution !== "passed" || item.expected.structurallyValidContext !== "passed") assert.equal(report.promotion.eligibleForReviewedUse, false);
    if (item.title.startsWith("Incorrect wording")) assert.equal(report.stages.supportedWording.metrics.unsupportedCertified, 1);
    if (item.title.startsWith("Honest drafts")) {
      assert.equal(report.stages.structurallyValidContext.readiness.draft, 7);
      assert.equal(report.promotion.eligibleForReviewedUse, false);
    }
  });
  const calibration = await calibrateScorecard();
  assert.equal(calibration.failed, 0);
  assert.equal(calibration.passed, cases.length);
  assert.equal(calibration.modelFree, true);
});

test("supported control reports evidence coverage and descriptive paired benefit without granting merge approval", async () => {
  const report = evaluate(await calibrationTrial());
  assert.equal(report.promotion.eligibleForReviewedUse, true);
  assert.match(report.promotion.limitation, /Not merge permission/);
  assert.deepEqual(report.stages.supportedWording.metrics, { required: 10, reviewed: 10, supported: 10, narrow: 0, unresolved: 0, unsupportedCertified: 0, unreviewedFactDocuments: [], unreviewedVerifiedDocuments: [] });
  assert.equal(report.stages.structurallyValidContext.facts.length, 7);
  assert.equal(report.stages.usefulDownstreamResults.efficiency.improvementFraction, 0.2);
  assert.match(report.stages.usefulDownstreamResults.limitation, /not statistical significance/);
});

test("fingerprints are key-order-independent, array-order-sensitive, and bind every context file", async () => {
  assert.equal(fingerprint({ b: 2, a: { d: 4, c: 3 } }), fingerprint({ a: { c: 3, d: 4 }, b: 2 }));
  assert.notEqual(fingerprint([1, 2]), fingerprint([2, 1]));
  const trial = await calibrationTrial(), before = candidateFingerprint(trial);
  trial.attempt.writes[".ctx/local/CONTINUE.md"] += "Changed handoff";
  assert.notEqual(candidateFingerprint(trial), before);
});

test("protected or conflicting proposals cannot be mistaken for the valid seed scaffold", async () => {
  for (const path of ["src/entry-store.mjs", ".ctx/config.json", "../outside.md", ".ctx/context/behavior/index.md/child.md"]) {
    const trial = await calibrationTrial(); trial.attempt.writes[path] = "Forbidden";
    const report = evaluateScorecard(trial);
    assert.equal(report.stages.successfulExecution.verdict, "failed");
    assert.equal(report.candidateHash, null);
    assert.equal(report.stages.structurallyValidContext.verdict, "not-evaluated");
  }
});

test("runtime selection and frozen source integrity need evidence, not just successful final text", async () => {
  for (const mutate of [
    trial => { trial.result.modelRegistryUnchanged = false; },
    trial => { trial.input.files["src/entry-store.mjs"] += "Changed source"; },
    trial => { trial.input.files["extra-source.txt"] = "Not manifested"; },
    trial => { trial.result.sourceCommit = "b".repeat(40); },
    trial => { trial.attempt.final = "Different final"; },
    trial => { trial.attempt.events.pop(); },
    trial => { trial.attempt.outputBytes = 0; },
    trial => { trial.attempt.writeBytes = 0; },
    trial => { trial.attempt.fullReads = []; },
    trial => { trial.attempt.events.find(event => event.path === ".ctx/INDEX.md").outcome = "denied"; },
  ]) {
    const trial = await calibrationTrial(); mutate(trial);
    assert.equal(evaluateScorecard(trial).stages.successfulExecution.verdict, "failed");
  }
  for (const mutate of [
    trial => { delete trial.attempt; },
    trial => { delete trial.manifest.sourceBefore; },
    trial => { delete trial.result.runtime.modelQualification; },
    trial => { delete trial.result.runtime.version; },
    trial => { trial.result.runtime.mode = "replay"; },
  ]) {
    const trial = await calibrationTrial(); mutate(trial);
    assert.notEqual(evaluateScorecard(trial).stages.successfulExecution.verdict, "passed");
  }
});

test("Pi exact selection, effective thinking, and incomplete responses remain separate execution checks", async () => {
  const trial = await calibrationTrial();
  trial.manifest.runtime.kind = "pi-sdk";
  trial.result.runtime.effectiveThinking = "high";
  trial.result.runtime.messages = [{ model: trial.manifest.runtime.model, provider: trial.manifest.runtime.provider, stopReason: "stop" }];
  assert.equal(evaluateScorecard(trial).stages.successfulExecution.verdict, "passed");
  for (const mutate of [
    copy => { copy.result.runtime.effectiveThinking = "low"; },
    copy => { copy.result.runtime.messages[0].model = "other-model"; },
    copy => { copy.result.runtime.messages[0].stopReason = "length"; },
  ]) { const copy = structuredClone(trial); mutate(copy); assert.equal(evaluateScorecard(copy).stages.successfulExecution.verdict, "failed"); }
  trial.result.runtime.messages = [];
  assert.equal(evaluateScorecard(trial).stages.successfulExecution.verdict, "incomplete");
});

test("metadata rejects malformed, stale, unsafe, self-supported, and impossible verification anchors", async () => {
  for (const mutate of [
    trial => { trial.attempt.writes[fact] += '\n<!-- ctx:doc {"status":"draft","verifiedAt":"","sources":[]} -->\n'; },
    trial => editMetadata(trial, meta => ({ ...meta, status: "certain" })),
    trial => editMetadata(trial, meta => ({ ...meta, verifiedAt: "a".repeat(40) + " @ 2026-02-30" })),
    trial => editMetadata(trial, meta => ({ ...meta, verifiedAt: "b".repeat(40) + " @ 2026-10-27" })),
    trial => editMetadata(trial, meta => ({ ...meta, sources: ["../source"] })),
    trial => editMetadata(trial, meta => ({ ...meta, sources: [".ctx/context/overview/index.md"] })),
    trial => editMetadata(trial, meta => ({ ...meta, sources: [".agents/skills/ctx/SKILL.md"] })),
    trial => editMetadata(trial, meta => ({ ...meta, status: "draft" })),
  ]) { const trial = await calibrationTrial(); mutate(trial); assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "failed"); }
});

test("unavailable filtered evidence causes uncertainty, not a repository-absence claim", async () => {
  const trial = await calibrationTrial();
  editMetadata(trial, meta => ({ ...meta, sources: [...meta.sources, "src/unavailable.mjs"] }));
  const stage = evaluateScorecard(trial).stages.structurallyValidContext;
  assert.equal(stage.verdict, "incomplete");
  assert.match(stage.incomplete[0].title, /repository absence is not established/);
});

test("relative routes include the main router, allow correct cross-category targets, and ignore fenced examples", async () => {
  const trial = await calibrationTrial();
  trial.attempt.writes[fact] += '\nSee `../architecture/index.md` and [publication](publication.md#retry).\n\n```md\nSee `../../not-real.md`.\n```\n';
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "passed");
  trial.attempt.writes[".ctx/INDEX.md"] += "\n[Bad route](context/absent/index.md)\n";
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "failed");
  trial.attempt.writes[".ctx/INDEX.md"] = "# Routes\n\n`context/behavior/index.md`\n";
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "passed");
  for (const target of ["../../../../../outside.md", "/absolute.md", "file:/private.md", "javascript:alert", "data:text/plain,contents"]) {
    const copy = structuredClone(trial); copy.attempt.writes[fact] += `\n[Unshareable](${target})\n`;
    assert.equal(evaluateScorecard(copy).stages.structurallyValidContext.verdict, "failed");
  }
  trial.attempt.writes[fact] += "\n[External](https://example.invalid/doc) and [same document](#retry).\n";
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "passed");
});

test("unchanged draft seed guidance is not an active route but newly authored blockquotes are checked", async () => {
  const trial = await calibrationTrial();
  editMetadata(trial, meta => ({ ...meta, status: "draft", verifiedAt: "" }));
  const guidance = "> When installed, use `../contracts/index.md`.";
  trial.input.files[fact] += "\n" + guidance + "\n";
  trial.attempt.writes[fact] += "\n" + guidance + "\n";
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "passed");
  trial.attempt.writes[fact] += "\n> Newly authored bad route: `../../architecture/index.md`.\n";
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "failed");
  trial.attempt.writes[fact] = trial.attempt.writes[fact].replace("draft", "verified");
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "failed");
});

test("real source-directory coverage is allowed but textual path prefixes do not cover sibling files", async () => {
  const trial = await calibrationTrial();
  editMetadata(trial, meta => ({ ...meta, sources: ["src"] }));
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "passed");
  editMetadata(trial, meta => ({ ...meta, sources: ["sr"] }));
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "failed");
});

test("missing doctor and failed doctor are distinct from draft status and readiness", async () => {
  const trial = await calibrationTrial();
  trial.result.checks = [{ command: "status", code: 1 }];
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "incomplete");
  trial.result.checks.push({ command: "doctor", code: 1 });
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "failed");
  trial.result.checks[1].code = 0;
  assert.equal(evaluateScorecard(trial).stages.structurallyValidContext.verdict, "passed");
});

test("an unchanged flat layout remains unsupported evaluation scope rather than a broken folder-first scaffold", async () => {
  const trial = await calibrationTrial();
  trial.input.files[".ctx/config.json"] = JSON.stringify({ mode: "team", layoutVersion: 2 });
  for (const category of ["overview", "architecture", "behavior", "caveats", "glossary"]) {
    const path = `.ctx/context/${category}/index.md`;
    trial.input.files[`.ctx/context/${category}.md`] = trial.input.files[path];
    delete trial.input.files[path];
  }
  trial.input.files[".ctx/INDEX.md"] = "# Routes\n\nSee `context/behavior.md`.\n";
  trial.attempt.writes = {};
  const before = fingerprint(trial), stage = evaluateScorecard(trial).stages.structurallyValidContext;
  assert.equal(stage.verdict, "incomplete");
  assert.deepEqual(stage.issues, []);
  assert.ok(stage.incomplete.some(item => /folder-first/.test(item.title)));
  assert.equal(fingerprint(trial), before, "Grading never migrates or repairs existing scaffolds");
});

test("parseable JSON is not enough to establish a valid scaffold configuration", async () => {
  for (const config of [null, false, [], {}, { mode: "team" }, { mode: "secret", layoutVersion: 3 }, { mode: "team", layoutVersion: "3" }, { mode: "team", layoutVersion: 0 }, { mode: "team", layoutVersion: 2.5 }]) {
    const trial = await calibrationTrial(); trial.input.files[".ctx/config.json"] = JSON.stringify(config);
    const stage = evaluateScorecard(trial).stages.structurallyValidContext;
    assert.equal(stage.verdict, "failed");
    assert.ok(stage.issues.some(item => /configuration is invalid/.test(item.title)));
  }
});

test("a review must bind its rubric, source, quoted lines, owning document, and reviewer authority", async () => {
  const trial = await calibrationTrial(), rubric = calibrationRubric();
  for (const mutate of [
    review => { review.rubricHash = "b".repeat(64); },
    review => { review.sourceCommit = "b".repeat(40); },
    review => { review.claims[0].evidence[0].quote = "Unseen evidence"; },
    review => { review.claims[0].evidence[0].line = 999; },
    review => { review.claims[0].location.path = fact; },
    review => { review.claims[0].limits = ""; },
    review => { review.claims.push(review.claims[0]); },
    review => { review.documents[0].hash = "b".repeat(64); },
    review => { review.reviewer.kind = "author-self-certification"; },
  ]) {
    const review = calibrationReview(trial, rubric); mutate(review);
    assert.equal(evaluateScorecard(trial, { rubric, review }).stages.supportedWording.verdict, "incomplete");
  }
  const review = calibrationReview(trial, rubric); trial.manifest.fixture = false;
  assert.equal(evaluateScorecard(trial, { rubric, review }).stages.supportedWording.verdict, "incomplete");
});

test("parent claims are reviewed independently even when child details and downstream answers are correct", async () => {
  const trial = await calibrationTrial(), statement = statements["Parent mutation preconditions"];
  trial.attempt.writes[statement.document] = trial.attempt.writes[statement.document].replace(statement.good, statement.bad);
  const report = evaluate(trial);
  assert.equal(report.stages.supportedWording.verdict, "failed");
  assert.equal(report.stages.supportedWording.claims.find(claim => claim.title === "Repeated publication revision exception").verdict, "supported");
  assert.equal(report.stages.usefulDownstreamResults.verdict, "passed");
  assert.equal(report.promotion.eligibleForReviewedUse, false);
});

test("parent counterexample audits require every parent inventory and actual grounded cases without mutating input", async () => {
  const trial = await calibrationTrial(), rubric = { ...calibrationRubric(), parentSummaryAudit: true }, review = calibrationParentSummaries(trial, calibrationReview(trial, rubric));
  const before = fingerprint({ trial, rubric, review }), report = evaluateScorecard(trial, { rubric, review });
  assert.equal(report.stages.supportedWording.verdict, "passed");
  assert.equal(report.stages.supportedWording.parentSummaries.inspectedDocuments, parentSummaryDocuments(trial).length);
  assert.ok(parentSummaryDocuments(trial).some(item => item.path === ".ctx/INDEX.md"));
  assert.ok(parentSummaryDocuments(trial).some(item => item.path === ".ctx/local/CONTINUE.md"));
  assert.equal(fingerprint({ trial, rubric, review }), before);
  for (const mutate of [
    copy => { delete copy.parentSummaries; },
    copy => { copy.parentSummaries.pop(); },
    copy => { copy.parentSummaries.push(copy.parentSummaries[0]); },
    copy => { copy.parentSummaries[0] = null; },
    copy => { copy.parentSummaries[0].hash = "b".repeat(64); },
    copy => { copy.parentSummaries[0].rationale = ""; },
    copy => { copy.parentSummaries[0].claimTitles = ["Invented summary"]; },
    copy => { copy.parentSummaries[0].claimTitles = []; },
    copy => { copy.parentSummaries[0].claimTitles.push(copy.parentSummaries[0].claimTitles[0]); },
    copy => { delete copy.claims[0].counterexampleCheck; },
    copy => { copy.claims[0].counterexampleCheck.coverage = ""; },
    copy => { copy.claims[0].counterexampleCheck.cases = []; },
    copy => { copy.claims[0].counterexampleCheck.cases = [null]; },
    copy => { copy.claims[0].counterexampleCheck.cases.push(copy.claims[0].counterexampleCheck.cases[0]); },
    copy => { copy.claims[0].counterexampleCheck.cases[0].evidence[0].quote = "Not in source"; },
    copy => { copy.claims[0].counterexampleCheck.cases[0].relation = "outside-qualified-scope"; },
    copy => { copy.claims[0].counterexampleCheck.cases[0].relation = "unresolved"; copy.claims[0].counterexampleCheck.cases[0].missingEvidence = "An operation remains uninspected."; },
  ]) {
    const copy = structuredClone(review); mutate(copy);
    assert.equal(evaluateScorecard(trial, { rubric, review: copy }).stages.supportedWording.verdict, "incomplete");
  }
  const invalidRubric = { ...rubric, parentSummaryAudit: "yes" }, invalidReview = calibrationReview(trial, invalidRubric);
  assert.equal(evaluateScorecard(trial, { rubric: invalidRubric, review: invalidReview }).stages.supportedWording.verdict, "incomplete");
});

test("an explicit parent counterexample cannot be hidden by unlinking its inventory title", async () => {
  const trial = await calibrationTrial(), rubric = { ...calibrationRubric(), parentSummaryAudit: true };
  const statement = statements["Parent mutation preconditions"];
  trial.attempt.writes[statement.document] = trial.attempt.writes[statement.document].replace(statement.good, statement.bad);
  const review = calibrationParentSummaries(trial, calibrationReview(trial, rubric));
  const claim = review.claims.find(item => item.title === "Parent mutation preconditions"); claim.verdict = "supported";
  assert.equal(evaluateScorecard(trial, { rubric, review }).stages.supportedWording.verdict, "failed");
  review.parentSummaries.find(item => item.path === fact).claimTitles = [];
  const before = fingerprint({ trial, rubric, review }), report = evaluateScorecard(trial, { rubric, review });
  assert.equal(report.stages.supportedWording.verdict, "incomplete");
  assert.equal(report.promotion.eligibleForReviewedUse, false);
  assert.ok(report.stages.supportedWording.incomplete.some(item => item.title.includes("not linked")));
  assert.equal(fingerprint({ trial, rubric, review }), before);
});

test("parent case evidence participates in owning metadata checks and outside cases require explicit qualification", async () => {
  const trial = await calibrationTrial(), rubric = { ...calibrationRubric(), parentSummaryAudit: true }, review = calibrationParentSummaries(trial, calibrationReview(trial, rubric));
  const claim = review.claims.find(item => item.title === "Parent mutation preconditions"), path = "test/client-reopen.test.mjs";
  claim.counterexampleCheck.cases.push({ title: "A separately scoped test boundary", relation: "outside-qualified-scope", rationale: "Client reopen is not a task revision operation.", evidence: [{ path, line: 1, quote: trial.input.files[path].split("\n")[0] }] });
  const report = evaluateScorecard(trial, { rubric, review });
  assert.equal(report.stages.supportedWording.verdict, "passed");
  assert.deepEqual(report.stages.structurallyValidContext.reviewedEvidence.gaps, [{ document: fact, evidencePath: path }]);
  claim.counterexampleCheck.cases[0].relation = "counterexample";
  const contradicted = evaluateScorecard(trial, { rubric, review });
  assert.equal(contradicted.stages.supportedWording.verdict, "failed");
  assert.ok(contradicted.stages.supportedWording.issues.some(item => item.title.includes("counterexample")));
});

test("a source-manifest mistake cannot let context or the skill serve as its own evidence", async () => {
  const trial = await calibrationTrial(), rubric = calibrationRubric(), review = calibrationReview(trial, rubric);
  const path = ".agents/skills/ctx/SKILL.md";
  trial.manifest.sourceHashes[path] = digest(trial.input.files[path]);
  review.claims[0].evidence = [{ path, line: 1, quote: trial.input.files[path].trim() }];
  const report = evaluateScorecard(trial, { rubric, review });
  assert.equal(report.stages.successfulExecution.verdict, "failed");
  assert.equal(report.stages.supportedWording.verdict, "incomplete");
});

test("draft or not-applicable labels cannot hide unreviewed fact documents", async () => {
  const trial = await calibrationTrial(), rubric = calibrationRubric();
  trial.attempt.writes[".ctx/context/behavior/unreviewed.md"] = '# Unreviewed\n\n<!-- ctx:doc {"status":"draft","verifiedAt":"","sources":[]} -->\n\nUnreviewed extra statement.\n';
  let review = calibrationReview(trial, rubric);
  assert.equal(evaluateScorecard(trial, { rubric, review }).stages.supportedWording.verdict, "incomplete");
  delete trial.attempt.writes[".ctx/context/behavior/unreviewed.md"];
  editMetadata(trial, meta => ({ ...meta, status: "not-applicable", verifiedAt: "" }));
  review = calibrationReview(trial, rubric);
  assert.equal(evaluateScorecard(trial, { rubric, review }).stages.supportedWording.verdict, "incomplete");
});

test("review-grounded source bookkeeping catches unnamed evidence without conflating truth and metadata", async () => {
  const trial = await calibrationTrial(), rubric = calibrationRubric(), document = ".ctx/context/caveats/index.md", path = "test/client-reopen.test.mjs";
  trial.attempt.writes[document] = trial.attempt.writes[document].replace(/^Evidence source:.*\n/gm, "");
  editMetadata(trial, meta => ({ ...meta, sources: meta.sources.filter(source => source !== path) }), document);
  const review = calibrationReview(trial, rubric), before = fingerprint({ trial, review });
  const report = evaluateScorecard(trial, { rubric, review });
  assert.equal(report.stages.supportedWording.verdict, "passed", "Supported meaning does not imply complete metadata");
  assert.equal(report.stages.structurallyValidContext.verdict, "failed");
  assert.deepEqual(report.stages.structurallyValidContext.reviewedEvidence.gaps, [{ document, evidencePath: path }]);
  assert.equal(report.promotion.eligibleForReviewedUse, false);
  assert.equal(fingerprint({ trial, review }), before);
  for (const mutate of [
    copy => { copy.candidateHash = "b".repeat(64); },
    copy => { copy.claims = copy.claims.filter(claim => !claim.evidence.some(evidence => evidence.path === path)); },
    copy => { for (const claim of copy.claims) for (const evidence of claim.evidence) if (evidence.path === path) evidence.quote = "Not in the frozen source"; },
  ]) {
    const copy = structuredClone(review); mutate(copy);
    const report = evaluateScorecard(trial, { rubric, review: copy });
    assert.equal(report.stages.supportedWording.verdict, "incomplete");
    assert.deepEqual(report.stages.structurallyValidContext.reviewedEvidence.gaps, [], "Unbound or ungrounded evidence cannot manufacture a source-bookkeeping finding");
  }
  editMetadata(trial, meta => ({ ...meta, status: "draft", verifiedAt: "" }), document);
  assert.equal(evaluate(trial).stages.structurallyValidContext.verdict, "failed", "Draft status does not waive evidence bookkeeping");
  editMetadata(trial, meta => ({ ...meta, sources: [...meta.sources, "test"] }), document);
  assert.equal(evaluate(trial).stages.structurallyValidContext.verdict, "passed");
  editMetadata(trial, meta => ({ ...meta, sources: meta.sources.map(source => source === "test" ? "test/client" : source) }), document);
  assert.equal(evaluate(trial).stages.structurallyValidContext.verdict, "failed", "A filename prefix is not a directory source");
});

test("grounded not-applicable reviews also reconcile their supporting sources", async () => {
  const trial = await calibrationTrial(), rubric = calibrationRubric(), document = ".ctx/context/glossary/index.md", path = "docs/source-view.md";
  editMetadata(trial, meta => ({ ...meta, status: "not-applicable", verifiedAt: "" }), document);
  const review = calibrationReview(trial, rubric);
  review.documents.find(item => item.path === document).notApplicable = { rationale: "Synthetic applicability evidence for bookkeeping, not a real semantic decision.", evidence: [{ path, line: 1, quote: trial.input.files[path].split("\n")[0] }] };
  const report = evaluateScorecard(trial, { rubric, review });
  assert.equal(report.stages.supportedWording.verdict, "passed");
  assert.equal(report.stages.structurallyValidContext.verdict, "failed");
  assert.ok(report.stages.structurallyValidContext.reviewedEvidence.gaps.some(gap => gap.document === document && gap.evidencePath === path));
});

test("conditional audits validate applicability without fabricating positive authored claims", async () => {
  const trial = await calibrationTrial(), rubric = calibrationRubric();
  rubric.conditionalClaims = [{ title: "Owner-acceptance attribution", trigger: "An authored statement asserts owner acceptance of a limitation" }];
  const review = calibrationReview(trial, rubric);
  review.applicability = [{ title: rubric.conditionalClaims[0].title, verdict: "not-triggered", documents: reviewScopeDocuments(trial), claimTitles: [], rationale: "All candidate fact documents, router, and handoff were inspected; no such attribution is made." }];
  let stage = evaluateScorecard(trial, { rubric, review }).stages.supportedWording;
  assert.equal(stage.verdict, "passed");
  assert.equal(stage.applicability.notTriggered, 1);
  assert.equal(stage.applicability.reviewed, 1);
  assert.equal(stage.claims.some(claim => claim.title === rubric.conditionalClaims[0].title), false);
  assert.ok(review.applicability[0].documents.some(document => document.path === ".ctx/INDEX.md"));
  assert.ok(review.applicability[0].documents.some(document => document.path === ".ctx/local/CONTINUE.md"));
  for (const mutate of [
    copy => { copy.applicability = []; },
    copy => { copy.applicability.push(copy.applicability[0]); },
    copy => { copy.applicability[0].documents[0].hash = "b".repeat(64); },
    copy => { copy.applicability[0].documents.pop(); },
    copy => { copy.applicability[0].documents[0] = copy.applicability[0].documents[1]; },
    ...[null, 1, "not a document", [], {}, { path: ".ctx/INDEX.md", hash: null }].map(value => copy => { copy.applicability[0].documents[0] = value; }),
    copy => { copy.applicability[0].rationale = ""; },
    copy => { copy.applicability[0].title = "Unknown audit"; },
    copy => { copy.applicability[0].claimTitles = ["Invented claim location"]; },
    copy => { copy.applicability[0].verdict = "triggered"; copy.applicability[0].claimTitles = ["Invented claim location"]; },
    copy => { copy.applicability[0].verdict = "unresolved"; copy.applicability[0].missingEvidence = "Could not determine whether an attribution is made."; },
  ]) {
    const copy = structuredClone(review); mutate(copy);
    assert.equal(evaluateScorecard(trial, { rubric, review: copy }).stages.supportedWording.verdict, "incomplete");
  }
  const invalid = structuredClone(rubric); invalid.conditionalClaims[0].title = invalid.requiredClaims[0].title;
  const badReview = calibrationReview(trial, invalid);
  assert.equal(evaluateScorecard(trial, { rubric: invalid, review: badReview }).stages.supportedWording.verdict, "incomplete");
});

test("downstream observations must match the frozen protocol, candidate, task outcomes, and repetition cells", async () => {
  const trial = await calibrationTrial();
  for (const mutate of [
    comparison => { comparison.protocol.repetitions = 1; },
    comparison => { comparison.pairs[0].context.protocolHash = "b".repeat(64); },
    comparison => { comparison.pairs[0].context.candidateHash = "b".repeat(64); },
    comparison => { comparison.pairs[0].baseline.candidateHash = candidateFingerprint(trial); },
    comparison => { comparison.pairs[0].context.outcomes.pop(); },
    comparison => { comparison.pairs[0].context.outcomes[0].verdict = "unresolved"; },
    comparison => { comparison.pairs[0].context.outcomes[0].answerQuote = "Not in the answer"; },
    comparison => { comparison.pairs[0].context.outcomes[0].evidence[0].quote = "Not in the source"; },
    comparison => { comparison.pairs[0].context.outcomes[0].rationale = ""; },
    comparison => { delete comparison.reviewer; },
    comparison => { comparison.pairs[0].context.final += "Altered frozen answer"; },
    comparison => { comparison.pairs[0].context.metrics.calls = -1; },
    comparison => { comparison.pairs[0].context.metrics.calls = NaN; },
    comparison => { comparison.pairs[0].context.metrics = {}; },
    comparison => { comparison.pairs[1].repetition = 1; },
    comparison => { comparison.pairs[0].task = "Unregistered task"; },
  ]) {
    const comparison = calibrationComparison(trial); mutate(comparison);
    assert.equal(evaluateScorecard(trial, { comparison }).stages.usefulDownstreamResults.verdict, "incomplete");
  }
  for (const field of ["completed", "scopePreserved"]) {
    const comparison = calibrationComparison(trial); comparison.pairs[0].context[field] = false; bindComparisonRecords(comparison);
    assert.equal(evaluateScorecard(trial, { comparison }).stages.usefulDownstreamResults.verdict, "failed");
  }
});

test("quality improvement can be observed but efficiency is compared only at equal complete correctness", async () => {
  const trial = await calibrationTrial(), comparison = calibrationComparison(trial, { contextCost: 120 });
  comparison.pairs[0].baseline.outcomes[0].verdict = "unsupported";
  const stage = evaluateScorecard(trial, { comparison }).stages.usefulDownstreamResults;
  assert.equal(stage.verdict, "passed");
  assert.equal(stage.quality.qualityImproved, true);
  assert.equal(stage.efficiency.comparableQuality, false);
  assert.equal(stage.efficiency.improvementFraction, null);
  assert.equal(evaluateScorecard(trial, { comparison: calibrationComparison(trial, { contextCost: 100 }) }).stages.usefulDownstreamResults.verdict, "not-demonstrated");
  assert.equal(evaluateScorecard(trial).stages.usefulDownstreamResults.verdict, "not-evaluated");
});

test("correct behavior with misleading citations is not a fully correct answer or an efficiency benefit", async () => {
  const trial = await calibrationTrial(), comparison = calibrationComparison(trial);
  misleadingReaderCitation(comparison, trial);
  const before = fingerprint(comparison), stage = evaluateScorecard(trial, { comparison }).stages.usefulDownstreamResults;
  assert.equal(stage.verdict, "failed");
  assert.equal(stage.quality.contextPassed, stage.quality.expected, "Behavior outcomes still pass independently");
  assert.deepEqual(stage.quality.citationAccuracy, { required: true, baselinePassed: 3, contextPassed: 2, expected: 3 });
  assert.deepEqual(stage.quality.fullyCorrectAnswers, { baseline: 3, context: 2, expected: 3 });
  assert.equal(stage.quality.qualityImproved, false);
  assert.equal(stage.efficiency.comparableQuality, false);
  assert.equal(stage.efficiency.improvementFraction, null);
  assert.equal(fingerprint(comparison), before);
});

test("a baseline citation error can show an observed quality difference but not equal-quality efficiency", async () => {
  const trial = await calibrationTrial(), comparison = calibrationComparison(trial, { contextCost: 120 });
  misleadingReaderCitation(comparison, trial, "baseline");
  const stage = evaluateScorecard(trial, { comparison }).stages.usefulDownstreamResults;
  assert.equal(stage.verdict, "passed");
  assert.equal(stage.quality.baselinePassed, stage.quality.contextPassed);
  assert.equal(stage.quality.qualityImproved, true);
  assert.deepEqual(stage.quality.fullyCorrectAnswers, { baseline: 2, context: 3, expected: 3 });
  assert.equal(stage.efficiency.comparableQuality, false);
  assert.equal(stage.efficiency.improvementFraction, null);
});

test("known context citation failures survive malformed or unresolved neighboring review entries", async () => {
  const trial = await calibrationTrial();
  for (const mutate of [
    comparison => { comparison.pairs[0].context.citationReview.claims[1] = null; },
    comparison => { comparison.pairs[0].context.citationReview.claims[1].evidence = []; },
    comparison => { comparison.pairs[0].context.citationReview.claims[1].verdict = "unresolved"; comparison.pairs[0].context.citationReview.claims[1].missingEvidence = "Unreviewed attribution"; },
    comparison => { comparison.pairs[0].baseline.citationReview.claims = []; },
    comparison => { delete comparison.pairs[0].context.outcomes; },
    comparison => { comparison.pairs[0].context.outcomes[0] = null; },
  ]) {
    const comparison = calibrationComparison(trial); misleadingReaderCitation(comparison, trial); mutate(comparison);
    const stage = evaluateScorecard(trial, { comparison }).stages.usefulDownstreamResults;
    assert.equal(stage.verdict, "failed");
    assert.ok(stage.issues.some(item => /unsupported source attributions/.test(item.title)));
    assert.ok(stage.incomplete.length);
    assert.equal(stage.efficiency.comparableQuality, false);
  }
});

test("known context behavior failures survive an unresolved baseline or neighboring outcome", async () => {
  const trial = await calibrationTrial();
  for (const condition of ["baseline", "context"]) {
    const comparison = calibrationComparison(trial);
    comparison.pairs[0].context.outcomes[0].verdict = "unsupported";
    const unresolved = comparison.pairs[0][condition].outcomes[1];
    unresolved.verdict = "unresolved"; unresolved.missingEvidence = "Outcome still needs adjudication";
    const stage = evaluateScorecard(trial, { comparison }).stages.usefulDownstreamResults;
    assert.equal(stage.verdict, "failed");
    assert.ok(stage.issues.some(item => /required correctness outcomes/.test(item.title)));
    assert.ok(stage.incomplete.length);
  }
  const comparison = calibrationComparison(trial);
  comparison.pairs[0].context.outcomes[0].verdict = "unsupported";
  comparison.pairs[0].context.outcomes[1] = null;
  assert.equal(evaluateScorecard(trial, { comparison }).stages.usefulDownstreamResults.verdict, "failed");
});

test("citation reviews must inventory this exact answer with grounded attributed assessments", async () => {
  const trial = await calibrationTrial();
  for (const mutate of [
    audit => { audit.answerHash = "b".repeat(64); },
    audit => { audit.materialClaimsInventoried = false; },
    audit => { audit.scope = ""; },
    audit => { audit.claims = []; },
    audit => { audit.claims[0] = null; },
    audit => { audit.claims.push(audit.claims[0]); },
    audit => { audit.claims[0].title = ""; },
    audit => { audit.claims[0].answerQuote = "Not in the answer"; },
    audit => { audit.claims[0].rationale = ""; },
    audit => { audit.claims[0].verdict = "author-approved"; },
    audit => { audit.claims[0].evidence = []; },
    audit => { audit.claims[0].evidence[0].quote = "Not in the source"; },
    audit => { audit.claims[0].evidence[0].path = fact; },
    audit => { audit.claims[0].verdict = "unresolved"; audit.claims[0].missingEvidence = []; },
  ]) {
    const comparison = calibrationComparison(trial); mutate(comparison.pairs[0].context.citationReview);
    const stage = evaluateScorecard(trial, { comparison }).stages.usefulDownstreamResults;
    assert.equal(stage.verdict, "incomplete");
    assert.equal(stage.efficiency.comparableQuality, false);
  }
});

test("citation accuracy is a frozen protocol requirement and legacy protocols cannot claim it", async () => {
  const trial = await calibrationTrial(), legacy = calibrationComparison(trial, { legacy: true });
  const before = fingerprint(legacy), stage = evaluateScorecard(trial, { comparison: legacy }).stages.usefulDownstreamResults;
  assert.equal(stage.verdict, "passed");
  assert.deepEqual(stage.quality.citationAccuracy, { required: false, baselinePassed: null, contextPassed: null, expected: null });
  assert.equal(stage.quality.fullyCorrectAnswers, null);
  assert.match(stage.limitation, /Legacy protocols did not grade citation accuracy/);
  assert.equal(fingerprint(legacy), before);
  for (const mutate of [
    comparison => { delete comparison.protocol.citationPolicy; },
    comparison => { comparison.protocol.citationPolicy = "optional"; },
    comparison => { comparison.protocol.schemaVersion = 3; },
    comparison => { delete comparison.protocol.schemaVersion; },
    comparison => { delete comparison.pairs[0].baseline.citationReview; },
    comparison => { comparison.protocol.tasks[0] = null; },
    comparison => { comparison.pairs[0] = null; },
    comparison => { comparison.pairs[0].context.outcomes[0] = null; },
  ]) {
    const comparison = calibrationComparison(trial); mutate(comparison);
    assert.equal(evaluateScorecard(trial, { comparison }).stages.usefulDownstreamResults.verdict, "incomplete");
  }
  const withUnfrozenAudit = structuredClone(legacy);
  withUnfrozenAudit.pairs[0].context.citationReview = calibrationComparison(trial).pairs[0].context.citationReview;
  bindComparisonRecords(withUnfrozenAudit);
  assert.equal(evaluateScorecard(trial, { comparison: withUnfrozenAudit }).stages.usefulDownstreamResults.verdict, "incomplete");
});

test("scorecard input and output paths are explicit, bounded, regular, private, and never overwritten", async t => {
  const root = await temporary(t), path = join(root, "report.json");
  await assert.rejects(writeScorecardJson("relative.json", {}), /Absolute/);
  await writeScorecardJson(path, { value: 1 });
  assert.deepEqual(await readScorecardJson(path), { value: 1 });
  if (process.platform !== "win32") assert.equal((await lstat(path)).mode & 0o777, 0o600);
  await assert.rejects(writeScorecardJson(path, { value: 2 }), { code: "EEXIST" });
  await assert.rejects(readScorecardJson("relative.json"), /Absolute/);
  await assert.rejects(readScorecardJson(root), /regular/);
  const link = join(root, "link.json"); await symlink(path, link);
  await assert.rejects(readScorecardJson(link), /regular/);
  const large = join(root, "large.json"), handle = await open(large, "wx");
  await handle.truncate(64 * 1024 * 1024 + 1); await handle.close();
  await assert.rejects(readScorecardJson(large), /bounded/);
});

test("CLI distinguishes report generation, reviewed-context gating, and invalid inputs without exposing private errors", async t => {
  const root = await temporary(t), trial = await calibrationTrial(), directory = await frozenDirectory(root, trial);
  const rubric = calibrationRubric(), review = calibrationReview(trial, rubric), comparison = calibrationComparison(trial);
  for (const [name, value] of Object.entries({ rubric, review, comparison })) await writeScorecardJson(join(root, name + ".json"), value);
  const script = new URL("./evaluate-context.mjs", import.meta.url).pathname;
  const run = args => spawnSync(process.execPath, [script, ...args], { encoding: "utf8" });
  const args = ["--trial", directory, "--rubric", join(root, "rubric.json"), "--review", join(root, "review.json"), "--comparison", join(root, "comparison.json")];
  const before = fingerprint(await readFrozenTrial(directory)), output = join(root, "output.json");
  const success = run([...args, "--output", output, "--require-reviewed-context"]);
  assert.equal(success.status, 0, success.stderr);
  assert.equal(JSON.parse(success.stdout).promotion.eligibleForReviewedUse, true);
  const outputHash = digest(await readFile(output));
  assert.equal(fingerprint(await readFrozenTrial(directory)), before);
  assert.equal(run([...args, "--output", output]).status, 2, "Existing reports must never be overwritten");
  const unreviewed = run(["--trial", directory]);
  assert.equal(unreviewed.status, 0, "Report generation is not approval");
  assert.equal(JSON.parse(unreviewed.stdout).stages.supportedWording.verdict, "not-evaluated");
  assert.equal(run(["--trial", directory, "--require-reviewed-context"]).status, 1);
  for (const invalid of [[], ["--trial", "relative"], [...args, "--trial", directory], ["--unknown"]]) assert.equal(run(invalid).status, 2);
  await writeFile(join(root, "private.json"), "PRIVATE-SENTINEL-DO-NOT-PRINT");
  const invalid = run(["--trial", directory, "--review", join(root, "private.json")]);
  assert.equal(invalid.status, 2);
  assert.equal(invalid.stderr.includes("PRIVATE-SENTINEL"), false);
  assert.equal(invalid.stdout, "");
  assert.deepEqual(candidateFiles(await readFrozenTrial(directory)), candidateFiles(trial));
  assert.equal(digest(await readFile(output)), outputHash, "Failed retries must not modify a report");
});

test("calibration CLI persists its model-free report once and refuses invalid or duplicate destinations", async t => {
  const root = await temporary(t), path = join(root, "calibration.json"), script = new URL("./calibrate-context-scorecard.mjs", import.meta.url).pathname;
  const run = args => spawnSync(process.execPath, [script, ...args], { encoding: "utf8" });
  const result = run(["--output", path]);
  assert.equal(result.status, 0, result.stderr);
  assert.equal(JSON.parse(result.stdout).passed, 50);
  const before = await readFile(path, "utf8");
  assert.equal(run(["--output", path]).status, 2);
  assert.equal(run(["--trial", root]).status, 2);
  assert.equal(await readFile(path, "utf8"), before);
});
