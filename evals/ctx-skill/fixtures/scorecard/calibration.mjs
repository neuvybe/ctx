// A finite, public reference oracle for known synthetic claims. This is NOT a
// semantic grader for arbitrary repositories or paraphrases. Unknowns abstain.
import { readFile } from "node:fs/promises";
import { digest } from "../../../../tools/lib/authoring-boundary.mjs";
import { candidateFiles, candidateFingerprint, fingerprint, parentSummaryDocuments } from "../../../../tools/lib/context-scorecard.mjs";
import { storage, owner, requiresRevision, publish, classify, outcome, reopenClient } from "./entry-store.mjs";

const commit = "a".repeat(40);
const sourceNames = { "src/entry-store.mjs": "entry-store.mjs", "test/client-reopen.test.mjs": "client-reopen.test.mjs", "docs/source-view.md": "source-view.md" };
const definition = [
  { title: "Storage durability boundary", document: ".ctx/context/overview/index.md", good: "Entries are stored only in process memory.", bad: "Entries survive a process restart.", basis: "implementation", path: "src/entry-store.mjs", evidence: 'export const storage = "memory";', supported: () => storage === "memory", limit: "Storage is process-local; no restart or persistence mechanism is exercised." },
  { title: "Component ownership", document: ".ctx/context/architecture/index.md", good: "The entry store owns entry state.", basis: "implementation", path: "src/entry-store.mjs", evidence: 'export const owner = "entry store";', supported: () => owner === "entry store", limit: "Component ownership does not imply durable storage." },
  { title: "Parent mutation preconditions", document: ".ctx/context/behavior/index.md", good: "Creation needs no revision token; updates and deletes do.", bad: "Every mutation requires a revision token.", basis: "implementation", path: "src/entry-store.mjs", evidence: 'return ["update", "delete", "publish"].includes(operation);', supported: () => !requiresRevision("create") && requiresRevision("update") && requiresRevision("delete"), limit: "Creation is a counterexample to the universal claim; header presence is distinct from matching the current revision." },
  { title: "Repeated publication revision exception", document: ".ctx/context/behavior/publication.md", good: "An already-published entry accepts a valid older revision token on publication retry.", bad: "Every publication requires the current revision token.", basis: "implementation", path: "src/entry-store.mjs", evidence: 'if (entry.state === "published") return entry;', supported: () => { const entry = { state: "published", revision: "new" }; return publish(entry, "old") === entry; }, limit: "A revision header is still required. The already-published no-op returns before revision comparison." },
  { title: "Import classification priority", document: ".ctx/context/behavior/imports.md", good: "An unsupported item without content is classified as unsupported, not inaccessible.", bad: "Missing content is classified as inaccessible before item kind is checked.", basis: "implementation", path: "src/entry-store.mjs", evidence: 'if (kind !== "entry") return "unsupported";', supported: () => classify("draft", false) === "unsupported", limit: "The kind check precedes content availability; this is a classifier-order claim, not a list of possible reasons." },
  { title: "Restart evidence boundary", document: ".ctx/context/caveats/index.md", good: "Reopening a client does not test a process restart.", bad: "Reopening a client proves process-restart recovery.", basis: "direct-test", path: "test/client-reopen.test.mjs", evidence: 'const second = reopenClient(first.store);', supported: () => { const store = new Map(); return reopenClient(store).store === store; }, limit: "The complete test constructs two clients over one Map and checks one value; it starts no replacement process." },
  { title: "Owner acceptance boundary", document: ".ctx/context/caveats/index.md", good: "Implementation of memory-only storage does not establish owner acceptance.", bad: "The owner has accepted memory-only storage as an intentional tradeoff.", basis: "implementation", path: "src/entry-store.mjs", evidence: 'export const storage = "memory";', supported: () => storage === "memory", limit: "An implementation choice is not an owner decision. An acceptance claim requires an explicit owner requirement or decision, not code alone." },
  { title: "Test assertion strength", document: ".ctx/context/caveats/index.md", good: "The reopened-client test checks one saved value, not all failure paths.", bad: "The reopened-client test proves all failure paths preserve entry state.", basis: "direct-test", path: "test/client-reopen.test.mjs", evidence: 'assert.equal(second.store.get("entry"), "published");', supported: () => true, limit: "One saved-value assertion does not exercise failure paths or prove an entire invariant family." },
  { title: "Filtered view absence boundary", document: ".ctx/context/caveats/index.md", good: "An unavailable path does not establish repository absence.", bad: "A path missing from this view is absent from the repository.", basis: "owner-requirement", path: "docs/source-view.md", evidence: "An unavailable path does not establish that it is absent from the repository.", supported: () => true, limit: "The available source view is explicitly filtered." },
  { title: "Outcome terminology", document: ".ctx/context/glossary/index.md", good: "An origin change maps to needs-review, not unresolved.", bad: "An origin change maps to unresolved.", basis: "implementation", path: "src/entry-store.mjs", evidence: 'return reason === "origin_changed" ? "needs-review" : "unresolved";', supported: () => outcome("origin_changed") === "needs-review", limit: "A reason and its outcome are separate concepts; the glossary must preserve their mapping." },
];
definition.find(item => item.title === "Parent mutation preconditions").badAlternatives = ["Entry mutation flows through revision validation."];
export const statements = Object.fromEntries(definition.map(item => [item.title, { good: item.good, bad: item.bad, badAlternatives: item.badAlternatives, document: item.document }]));

export async function calibrationTrial() {
  const sources = Object.fromEntries(await Promise.all(Object.entries(sourceNames).map(async ([path, file]) => [path, await readFile(new URL(file, import.meta.url), "utf8")])));
  sources["AGENTS.md"] = "Preserve source and distinguish implementation from requirements.\n";
  sources["CLAUDE.md"] = sources["AGENTS.md"];
  const files = { ...sources, ".agents/skills/ctx/SKILL.md": "Read references/evidence-review.md before certification.\n", ".agents/skills/ctx/references/evidence-review.md": "Audit owning claims, exact evidence, limits and verdicts.\n", ".ctx/INDEX.md": "# Routes\n", ".ctx/README.md": "# Synthetic team scaffold\n", ".ctx/config.json": JSON.stringify({ mode: "team", layoutVersion: 3 }), ".ctx/.gitignore": "/local/\n", ".ctx/local/CONTINUE.md": "# Local handoff\n" };
  const writes = {};
  for (const document of new Set(definition.map(item => item.document))) {
    const items = definition.filter(item => item.document === document);
    const meta = { status: "verified", verifiedAt: commit + " @ 2026-10-27", sources: [...new Set(items.map(item => item.path))] };
    writes[document] = "# " + items[0].title + "\n\n<!-- ctx:doc " + JSON.stringify(meta) + " -->\n\n" + items.map(item => item.good).join("\n\n") + "\n\n" + meta.sources.map(path => "Evidence source: `" + path + "` — synthetic evidence.").join("\n") + "\n";
    files[document] = "# Seed\n\n<!-- ctx:doc {\"status\":\"draft\",\"verifiedAt\":\"\",\"sources\":[]} -->\n";
  }
  writes[".ctx/INDEX.md"] = "# Routes\n\nUse context/behavior/index.md for entry rules.\n";
  writes[".ctx/local/CONTINUE.md"] = "# Handoff\n\nSynthetic calibration only.\n";
  const final = "All 7 fact documents are verified.";
  const audit = { head: commit, entries: { "src/entry-store.mjs": { hash: digest(sources["src/entry-store.mjs"]) } } };
  const events = [...Object.keys(files).map(path => ({ tool: "ctx_read", outcome: "allowed", path, returnedBytes: 10 })), ...Object.keys(writes).map(path => ({ tool: "ctx_write", outcome: "allowed", path, returnedBytes: 10 }))];
  const budgets = { timeoutMs: 60000, maxCalls: 200, maxOutputBytes: 1048576, maxWriteBytes: 1048576 };
  return { input: { files, sourceCommit: commit, sourceOmissions: { "sensitive-path": 1 }, thinking: "high", budgets },
    manifest: { fixture: true, sourceCommit: commit, budgets: structuredClone(budgets), sourceHashes: Object.fromEntries(Object.entries(sources).map(([path, text]) => [path, digest(text)])), sourceBefore: audit, runtime: { kind: "codex", provider: "fixture-provider", model: "scripted-calibration-model", thinking: "high" } },
    attempt: { writes, final, events, calls: events.length, outputBytes: events.length * 10, writeBytes: Object.values(writes).reduce((n, text) => n + Buffer.byteLength(text), 0), fullReads: Object.keys(files) },
    result: { sourceCommit: commit, sourceUnchanged: true, modelRegistryUnchanged: true, sourceAfter: structuredClone(audit), final, failure: null, checks: [{ command: "doctor", code: 0 }, { command: "status", code: 0 }], runtime: { mode: "provider", version: "synthetic-fixture-1", fixture: true, incomplete: false, code: 0, interrupted: "completed", modelQualification: { exactModelAdvertised: true, requestedEffortAdvertised: true } } } };
}

export function calibrationRubric() {
  return { schemaVersion: 1, title: "Synthetic entry-context evidence scope", requiredClaims: definition.map(({ title, document }) => ({ title, document })) };
}

export function calibrationReview(trial, rubric = calibrationRubric()) {
  const files = candidateFiles(trial);
  const claims = definition.map(item => {
    const text = files[item.document];
    // A supported sentence never excuses a contradictory known overclaim in
    // the same document. This remains finite matching, not semantic inference.
    const overclaim = [item.bad, ...(item.badAlternatives || [])].find(quote => quote && text.includes(quote));
    const known = overclaim || (text.includes(item.good) ? item.good : null);
    const quote = known || text.split("\n").find(line => line && !line.startsWith("#") && !line.startsWith("<!--") && !line.startsWith("Evidence source:")) || "Unknown wording";
    const supported = known === item.good && item.supported();
    const result = { title: item.title, location: { path: item.document, line: text.split("\n").findIndex(line => line.includes(quote)) + 1, quote }, basis: item.basis, verdict: !known ? "unresolved" : supported ? "supported" : "narrow", limits: item.limit,
      evidence: [{ path: item.path, line: files[item.path].split("\n").findIndex(line => line.includes(item.evidence)) + 1, quote: item.evidence }] };
    if (!known) result.missingEvidence = "Finite fixture oracle abstains on unfamiliar wording; independent review is needed.";
    if (known && !supported) result.correction = item.good;
    return result;
  });
  return { schemaVersion: 1, sourceCommit: trial.input.sourceCommit, candidateHash: candidateFingerprint(trial), rubricHash: fingerprint(rubric), reviewer: { kind: "fixture-reference", name: "Finite synthetic claim oracle" }, scope: "Only the ten enumerated synthetic claims, grounded in the public fixture; not a general semantic grader.", claims,
    documents: [...new Set(definition.map(item => item.document))].map(path => ({ path, hash: digest(files[path]), materialClaimsInventoried: true })),
    reportedFactDocuments: { value: 7, quote: "All 7 fact documents are verified." } };
}

// Finite synthetic case inventories only, not a detector for arbitrary parents.
export function calibrationParentSummaries(trial, review) {
  review.parentSummaries = parentSummaryDocuments(trial).map(document => {
    const claims = review.claims.filter(claim => claim.location.path === document.path);
    for (const claim of claims) claim.counterexampleCheck = { coverage: "Only this finite synthetic claim and its enumerated source operation; not repository-wide completeness.",
      cases: [{ title: "Inspected source case for " + claim.title, relation: claim.verdict === "narrow" ? "counterexample" : claim.verdict === "unresolved" ? "unresolved" : "supports", rationale: claim.limits,
        evidence: structuredClone(claim.evidence), ...(claim.verdict === "unresolved" ? { missingEvidence: claim.missingEvidence } : {}) }] };
    return { ...document, claimTitles: claims.map(claim => claim.title), rationale: claims.length ? "These finite fixture claims are inventoried as parent summaries." : "The synthetic router/handoff contains no material behavioral summary." };
  });
  const mutation = review.claims.find(claim => claim.title === "Parent mutation preconditions");
  mutation.counterexampleCheck.coverage = "Create, update, delete and publication are the complete operation vocabulary of this synthetic revision predicate.";
  mutation.counterexampleCheck.cases = ["create", "update", "delete", "publish"].map(operation => ({ title: "Revision predicate for " + operation,
    relation: operation === "create" && mutation.verdict === "narrow" ? "counterexample" : "supports",
    rationale: operation === "create" ? "Creation is not in the token-requiring list." : "This operation is in the token-requiring list.", evidence: structuredClone(mutation.evidence) }));
  return review;
}

export function calibrationComparison(trial, { baselineCost = 100, contextCost = 80, repetitions = 3, legacy = false } = {}) {
  const protocol = { ...(legacy ? {} : { schemaVersion: 2, citationPolicy: "all-material-claims" }), sourceCommit: trial.input.sourceCommit, reader: { model: "scripted-reader", provider: "fixture-provider", runtime: "fixture", version: "1", thinking: "high" }, budgetFingerprint: fingerprint(trial.input.budgets), repetitions, efficiencyMetric: "calls", minimumImprovementFraction: 0.1,
    tasks: [{ title: "Explain publication retry without inventing restart recovery", kind: "read-only", requiredOutcomes: ["Preserves revision exception", "Does not claim process restart"] }] };
  const protocolHash = fingerprint(protocol);
  const files = candidateFiles(trial);
  const observation = (condition, repetition, cost) => {
    const record = { protocolHash, recordLabel: `${condition} synthetic repetition ${repetition}`, candidateHash: condition === "context" ? candidateFingerprint(trial) : null, completed: true, scopePreserved: true, writeAttempts: 0,
      final: "An already-published entry accepts an older revision on retry (src/entry-store.mjs, publish). Reopening a client is not process restart evidence (test/client-reopen.test.mjs, reopened-client test).", metrics: { calls: cost } };
    const outcomes = [definition.find(item => item.title === "Repeated publication revision exception"), definition.find(item => item.title === "Restart evidence boundary")].map((item, index) => ({ title: protocol.tasks[0].requiredOutcomes[index], verdict: "supported", answerQuote: index ? "Reopening a client is not process restart evidence" : "An already-published entry accepts an older revision on retry", rationale: item.limit, evidence: [{ path: item.path, line: files[item.path].split("\n").findIndex(line => line.includes(item.evidence)) + 1, quote: item.evidence }] }));
    const citationReview = { answerHash: digest(record.final), materialClaimsInventoried: true, scope: "Both material assertions and their source-file/symbol attributions; no other factual claims occur in the scripted answer.",
      claims: outcomes.map((outcome, index) => ({ ...structuredClone(outcome), title: "Source attribution for " + outcome.title,
        answerQuote: index ? "Reopening a client is not process restart evidence (test/client-reopen.test.mjs, reopened-client test)." : "An already-published entry accepts an older revision on retry (src/entry-store.mjs, publish)." })) };
    return { ...record, artifactHash: fingerprint(record), outcomes, ...(legacy ? {} : { citationReview }) };
  };
  return { schemaVersion: 1, reviewer: { kind: "fixture-reference", name: "Synthetic held-out outcome reference" }, scope: "Known scripted outcomes only; no real reader performance measured.", protocol, pairs: Array.from({ length: repetitions }, (_, index) => ({ task: protocol.tasks[0].title, repetition: index + 1, order: index % 2 ? "context-first" : "baseline-first", baseline: observation("baseline", index + 1, baselineCost), context: observation("context", index + 1, contextCost) })) };
}

// Refresh only when deliberately constructing a new fixture record, never when
// checking whether an external record still matches its original fingerprint.
export function bindComparisonRecords(comparison) {
  for (const pair of comparison.pairs) for (const observation of [pair.baseline, pair.context]) {
    const { artifactHash, outcomes, ...record } = observation;
    if (comparison.protocol.schemaVersion === 2) delete record.citationReview;
    observation.artifactHash = fingerprint(record);
  }
}

// Deliberate synthetic mistake: behavior remains correct, but an unrelated
// classifier is cited as publication evidence. The oracle is supplied, not
// inferred from the final answer by the scorecard.
export function misleadingReaderCitation(comparison, trial, condition = "context") {
  const observation = comparison.pairs[0][condition];
  observation.final = observation.final.replace("src/entry-store.mjs, publish", "src/entry-store.mjs, classify");
  const audit = observation.citationReview;
  audit.answerHash = digest(observation.final);
  audit.claims[0].answerQuote = audit.claims[0].answerQuote.replace("src/entry-store.mjs, publish", "src/entry-store.mjs, classify");
  audit.claims[0].verdict = "unsupported";
  audit.claims[0].rationale = "The cited classifier does not implement publication retry; the publication branch supports the behavior but not this attribution.";
  const quote = 'if (kind !== "entry") return "unsupported";';
  audit.claims[0].evidence.push({ path: "src/entry-store.mjs", line: trial.input.files["src/entry-store.mjs"].split("\n").findIndex(line => line.includes(quote)) + 1, quote });
  bindComparisonRecords(comparison);
}
