import { calibrationTrial, calibrationRubric, calibrationReview, calibrationComparison, calibrationParentSummaries, statements, bindComparisonRecords, misleadingReaderCitation } from "./calibration.mjs";
import { reviewScopeDocuments } from "../../../../tools/lib/context-scorecard.mjs";

export async function calibrationCases() {
  const base = await calibrationTrial(), cases = [];
  function add(title, { mutate = () => {}, editRubric = () => {}, editReview = () => {}, editComparison = () => {}, noReview = false, expected = {} } = {}) {
    const trial = structuredClone(base);
    mutate(trial);
    trial.attempt.writeBytes = Object.values(trial.attempt.writes).reduce((sum, text) => sum + Buffer.byteLength(text), 0);
    const rubric = calibrationRubric();
    editRubric(rubric);
    const review = noReview ? undefined : calibrationReview(trial, rubric);
    if (review) editReview(review, trial);
    const comparison = calibrationComparison(trial);
    editComparison(comparison, trial);
    cases.push({ title, trial, rubric, review, comparison, expected: { successfulExecution: "passed", structurallyValidContext: "passed", supportedWording: "passed", usefulDownstreamResults: "passed", ...expected } });
  }
  add("Supported context and matched repeated benefit are accepted");
  for (const title of ["Parent mutation preconditions", "Repeated publication revision exception", "Import classification priority", "Restart evidence boundary", "Filtered view absence boundary", "Outcome terminology", "Owner acceptance boundary", "Test assertion strength"]) {
    add("Incorrect wording is rejected: " + title, { mutate(trial) { const statement = statements[title]; trial.attempt.writes[statement.document] = trial.attempt.writes[statement.document].replace(statement.good, statement.bad); }, expected: { supportedWording: "failed" } });
  }
  add("Supported sentence does not excuse a contradictory claim in the same document", { mutate(trial) { const statement = statements["Parent mutation preconditions"]; trial.attempt.writes[statement.document] += "\n" + statement.bad + "\n"; }, expected: { supportedWording: "failed" } });
  add("Correct child does not excuse an overbroad parent", { mutate(trial) { const statement = statements["Parent mutation preconditions"]; trial.attempt.writes[statement.document] = trial.attempt.writes[statement.document].replace(statement.good, statement.bad); }, editReview(review) { review.claims = review.claims.filter(claim => claim.title !== "Parent mutation preconditions"); }, expected: { supportedWording: "incomplete" } });
  add("Broken cross-category topic route is rejected", { mutate(trial) { trial.attempt.writes[".ctx/context/behavior/publication.md"] += "\nSee `../../architecture/index.md`.\n"; }, expected: { structurallyValidContext: "failed" } });
  add("Inline prose citation omitted from metadata is rejected", { mutate(trial) { trial.attempt.writes[".ctx/context/overview/index.md"] += "\nAdditional evidence: `docs/source-view.md`.\n"; }, expected: { structurallyValidContext: "failed" } });
  add("Zero explicit citations do not waive an inline citation gap", { mutate(trial) { for (const path of Object.keys(trial.attempt.writes)) trial.attempt.writes[path] = trial.attempt.writes[path].replaceAll("Evidence source:", "Source:"); trial.attempt.writes[".ctx/context/overview/index.md"] += "\nAdditional evidence: `docs/source-view.md`.\n"; }, expected: { structurallyValidContext: "failed" } });
  add("Unnamed test evidence identified by a grounded review requires owning metadata", { mutate(trial) {
    const path = ".ctx/context/caveats/index.md";
    trial.attempt.writes[path] = trial.attempt.writes[path].replace(/^Evidence source:.*\n/gm, "").replace(/<!-- ctx:doc (.*?) -->/, (_, json) => {
      const meta = JSON.parse(json); meta.sources = meta.sources.filter(source => source !== "test/client-reopen.test.mjs");
      return "<!-- ctx:doc " + JSON.stringify(meta) + " -->";
    });
  }, expected: { structurallyValidContext: "failed" } });
  add("A real directory source covers grounded review evidence in its files", { mutate(trial) {
    const path = ".ctx/context/caveats/index.md";
    trial.attempt.writes[path] = trial.attempt.writes[path].replace(/<!-- ctx:doc (.*?) -->/, (_, json) => {
      const meta = JSON.parse(json); meta.sources = meta.sources.map(source => source === "test/client-reopen.test.mjs" ? "test" : source);
      return "<!-- ctx:doc " + JSON.stringify(meta) + " -->";
    });
  } });
  add("Fact count excludes routing and local continuation", { mutate(trial) { trial.attempt.final = trial.result.final = "All 9 fact documents are verified."; }, editReview(review) { review.reportedFactDocuments = { value: 9, quote: "All 9 fact documents are verified." }; }, expected: { structurallyValidContext: "failed" } });
  add("Unfamiliar wording causes abstention rather than guessed semantic approval", { mutate(trial) { const statement = statements["Parent mutation preconditions"]; trial.attempt.writes[statement.document] = trial.attempt.writes[statement.document].replace(statement.good, "Revision requirements differ between operations."); }, expected: { supportedWording: "incomplete" } });
  add("Missing review remains unreviewed despite green structure and reader success", { noReview: true, expected: { supportedWording: "not-evaluated" } });
  add("Stale candidate review cannot certify changed context", { editReview(review) { review.candidateHash = "b".repeat(64); }, expected: { supportedWording: "incomplete" } });
  add("Missing material-claim inventory prevents blanket verification", { editReview(review) { review.documents = []; }, expected: { supportedWording: "incomplete" } });
  add("Evidence-reference access is observable but not semantic certification", { mutate(trial) { trial.attempt.fullReads = trial.attempt.fullReads.filter(path => !path.endsWith("references/evidence-review.md")); } });
  add("Honest drafts are structurally valid but not promotion-ready", { mutate(trial) { for (const [path, text] of Object.entries(trial.attempt.writes)) trial.attempt.writes[path] = text.replace(/<!-- ctx:doc (.*?) -->/, (_, json) => { const metadata = JSON.parse(json); metadata.status = "draft"; metadata.verifiedAt = ""; return "<!-- ctx:doc " + JSON.stringify(metadata) + " -->"; }); trial.attempt.final = trial.result.final = "Seven fact documents remain draft."; trial.result.checks[1].code = 1; }, editReview(review) { delete review.reportedFactDocuments; } });
  add("Source audit disagreement fails execution", { mutate(trial) { trial.result.sourceAfter.head = "b".repeat(40); }, expected: { successfulExecution: "failed" } });
  add("Exact-model qualification cannot be inferred from a final answer", { mutate(trial) { trial.result.runtime.modelQualification.exactModelAdvertised = false; }, expected: { successfulExecution: "incomplete" } });
  add("Late completion does not erase a supervisor timeout", { mutate(trial) { trial.result.runtime.interrupted = "time-budget"; }, expected: { successfulExecution: "failed" } });
  add("Tool budget exhaustion fails execution", { mutate(trial) { trial.input.budgets.maxCalls = 1; }, expected: { successfulExecution: "failed" } });
  add("One downstream observation is not repeatability evidence", { editComparison(comparison) { comparison.pairs = comparison.pairs.slice(0, 1); }, expected: { usefulDownstreamResults: "incomplete" } });
  add("Repeated observations must include both execution orders", { editComparison(comparison) { comparison.pairs.forEach(pair => { pair.order = "baseline-first"; }); }, expected: { usefulDownstreamResults: "incomplete" } });
  add("Duplicate downstream artifacts cannot masquerade as independent repeats", { editComparison(comparison) { comparison.pairs[1].context.artifactHash = comparison.pairs[0].context.artifactHash; }, expected: { usefulDownstreamResults: "incomplete" } });
  add("Correct answers with greater retrieval work do not demonstrate benefit", { editComparison(comparison, trial) { Object.assign(comparison, calibrationComparison(trial, { contextCost: 120 })); }, expected: { usefulDownstreamResults: "not-demonstrated" } });
  add("Fewer calls cannot compensate for an incorrect downstream answer", { editComparison(comparison) { comparison.pairs[0].context.outcomes[0].verdict = "unsupported"; }, expected: { usefulDownstreamResults: "failed" } });
  add("Correct behavior with a misleading source attribution fails downstream usefulness", { editComparison: misleadingReaderCitation, expected: { usefulDownstreamResults: "failed" } });
  add("Missing citation review does not receive full correctness credit", { editComparison(comparison) { delete comparison.pairs[0].context.citationReview; }, expected: { usefulDownstreamResults: "incomplete" } });
  add("A stale citation review cannot certify a changed reader answer", { editComparison(comparison) { comparison.pairs[0].context.final += " Additional factual assertion."; bindComparisonRecords(comparison); }, expected: { usefulDownstreamResults: "incomplete" } });
  add("An empty citation inventory cannot certify a factual reader answer", { editComparison(comparison) { comparison.pairs[0].context.citationReview.claims = []; }, expected: { usefulDownstreamResults: "incomplete" } });
  add("Unresolved citation accuracy is not quietly accepted", { editComparison(comparison) { const claim = comparison.pairs[0].context.citationReview.claims[0]; claim.verdict = "unresolved"; claim.missingEvidence = "The source attribution has not been adjudicated."; }, expected: { usefulDownstreamResults: "incomplete" } });
  add("Legacy protocol preserves its outcomes-only contract without claiming citation grading", { editComparison(comparison, trial) { Object.assign(comparison, calibrationComparison(trial, { legacy: true })); } });
  add("Read-only scope violations fail downstream usefulness", { editComparison(comparison) { comparison.pairs[0].context.writeAttempts = 1; bindComparisonRecords(comparison); }, expected: { usefulDownstreamResults: "failed" } });
  const editRubric = rubric => { rubric.conditionalClaims = [{ title: "Explicit owner-acceptance attribution", trigger: "An authored statement attributes owner acceptance of a limitation" }]; };
  const applicability = trial => ({ title: "Explicit owner-acceptance attribution", verdict: "not-triggered", rationale: "Inspected all fact documents, router, and handoff: no owner-acceptance attribution is made.", documents: reviewScopeDocuments(trial), claimTitles: [] });
  add("Conditional audit can be untriggered without inventing an authored claim", { editRubric, editReview(review, trial) { review.applicability = [applicability(trial)]; } });
  add("Missing conditional applicability is not silently treated as untriggered", { editRubric, expected: { supportedWording: "incomplete" } });
  add("An untriggered audit requires the complete bound document scope", { editRubric, editReview(review, trial) { const assessment = applicability(trial); assessment.documents.pop(); review.applicability = [assessment]; }, expected: { supportedWording: "incomplete" } });
  add("Malformed conditional document entries abstain without crashing", { editRubric, editReview(review, trial) { const assessment = applicability(trial); assessment.documents[0] = null; review.applicability = [assessment]; }, expected: { supportedWording: "incomplete" } });
  add("Triggered conditional audits link real independently reviewed occurrences", { editRubric,
    mutate(trial) { const statement = statements["Owner acceptance boundary"]; trial.attempt.writes[statement.document] = trial.attempt.writes[statement.document].replace(statement.good, statement.bad); },
    editReview(review, trial) { review.applicability = [{ ...applicability(trial), verdict: "triggered", rationale: "An owner-acceptance assertion is present and exceeds the implementation evidence.", claimTitles: ["Owner acceptance boundary"] }]; }, expected: { supportedWording: "failed" } });
  const requireParents = rubric => { rubric.parentSummaryAudit = true; };
  add("Parent summaries pass only with bound grounded case inventories", { editRubric: requireParents, editReview: (review, trial) => calibrationParentSummaries(trial, review) });
  add("An implicit generic parent flow is checked even without a universal quantifier", { editRubric: requireParents,
    mutate(trial) { const statement = statements["Parent mutation preconditions"]; trial.attempt.writes[statement.document] = trial.attempt.writes[statement.document].replace(statement.good, statement.badAlternatives[0]); },
    editReview: (review, trial) => calibrationParentSummaries(trial, review), expected: { supportedWording: "failed" } });
  add("A missing parent counterexample inventory cannot pass", { editRubric: requireParents, expected: { supportedWording: "incomplete" } });
  add("A reviewer cannot report support while recording a parent counterexample", { editRubric: requireParents,
    mutate(trial) { const statement = statements["Parent mutation preconditions"]; trial.attempt.writes[statement.document] = trial.attempt.writes[statement.document].replace(statement.good, statement.bad); },
    editReview(review, trial) { calibrationParentSummaries(trial, review); review.claims.find(claim => claim.title === "Parent mutation preconditions").verdict = "supported"; }, expected: { supportedWording: "failed" } });
  add("Ungrounded parent case inventories remain incomplete", { editRubric: requireParents, editReview(review, trial) { calibrationParentSummaries(trial, review); review.claims[0].counterexampleCheck.cases[0].evidence[0].quote = "Not in the source"; }, expected: { supportedWording: "incomplete" } });
  add("A supplied parent counterexample cannot be hidden by unlinking its inventory", { editRubric: requireParents,
    mutate(trial) { const statement = statements["Parent mutation preconditions"]; trial.attempt.writes[statement.document] = trial.attempt.writes[statement.document].replace(statement.good, statement.bad); },
    editReview(review, trial) { calibrationParentSummaries(trial, review); const claim = review.claims.find(claim => claim.title === "Parent mutation preconditions"); claim.verdict = "supported"; review.parentSummaries.find(item => item.path === claim.location.path).claimTitles = []; }, expected: { supportedWording: "incomplete" } });
  return cases;
}
