// Maintainer-side grading of frozen data. No provider calls, repository code
// execution, host-path reads, or natural-language truth inference occur here.
import { digest, safePath, validateProposal } from "./authoring-boundary.mjs";
import { posix } from "node:path";

export const scorecardVersion = "1.4.0";
const factPrefix = ".ctx/context/";
const sourcePath = path => path !== ".ctx" && !path.startsWith(".ctx/") && path !== ".agents/skills/ctx" && !path.startsWith(".agents/skills/ctx/");
const nonempty = value => typeof value === "string" && value.trim().length > 0;
const natural = value => Number.isSafeInteger(value) && value >= 0;
const fullCommit = value => typeof value === "string" && /^(?:[a-f0-9]{40}|[a-f0-9]{64})$/.test(value);
const hash = value => typeof value === "string" && /^[a-f0-9]{64}$/.test(value);
const issue = (title, document) => ({ title, ...(document ? { document } : {}) });
const verdict = (issues, incomplete = []) => issues.length ? "failed" : incomplete.length ? "incomplete" : "passed";

export function fingerprint(value) {
  const sorted = item => Array.isArray(item) ? item.map(sorted) : item && typeof item === "object"
    ? Object.fromEntries(Object.keys(item).sort().map(key => [key, sorted(item[key])])) : item;
  return digest(JSON.stringify(sorted(value)));
}

export function candidateFiles(trial) {
  return validateProposal(trial.input, trial.attempt || { writes: {} });
}

export function candidateFingerprint(trial) {
  return fingerprint(Object.fromEntries(Object.entries(candidateFiles(trial)).filter(([path]) => path.startsWith(".ctx/"))));
}

function scopeDocuments(files) {
  return Object.entries(files).filter(([path]) => path === ".ctx/INDEX.md" || path === ".ctx/local/CONTINUE.md" || (path.startsWith(factPrefix) && path.endsWith(".md")))
    .sort(([a], [b]) => a.localeCompare(b)).map(([path, text]) => ({ path, hash: digest(text) }));
}

export function reviewScopeDocuments(trial) {
  return scopeDocuments(candidateFiles(trial));
}

function parentDocuments(files) {
  return scopeDocuments(files).filter(document => document.path === ".ctx/INDEX.md" || document.path === ".ctx/local/CONTINUE.md" || posix.basename(document.path) === "index.md");
}

export function parentSummaryDocuments(trial) {
  return parentDocuments(candidateFiles(trial));
}

function execution(trial) {
  const { input, manifest, result, attempt } = trial;
  const issues = [], incomplete = [];
  const runtime = result.runtime || {};
  const expectedExit = runtime.code === 0 || (runtime.code === null && ["SIGTERM", "SIGKILL"].includes(runtime.signal) && runtime.interrupted === "completed");
  if (runtime.mode !== "provider") incomplete.push(issue("No provider authoring attempt was recorded"));
  if (result.failure || runtime.incomplete || runtime.spawnError || (runtime.mode === "provider" && !expectedExit) ||
      (runtime.interrupted && runtime.interrupted !== "completed") || !nonempty(result.final)) issues.push(issue("Authoring did not complete successfully"));
  if (result.sourceUnchanged !== true || result.modelRegistryUnchanged !== true) issues.push(issue("Source or model-registry integrity was not preserved"));
  if (!manifest.sourceBefore || !result.sourceAfter) incomplete.push(issue("Before-and-after source audit evidence is missing"));
  else if (fingerprint(manifest.sourceBefore) !== fingerprint(result.sourceAfter)) issues.push(issue("Recorded source audits disagree"));
  if (manifest.sourceCommit !== input.sourceCommit || result.sourceCommit !== input.sourceCommit) issues.push(issue("Frozen source anchors disagree"));
  if (!manifest.budgets || fingerprint(manifest.budgets) !== fingerprint(input.budgets)) issues.push(issue("Frozen execution budgets disagree with the manifest"));
  if (runtime.mode === "provider" && !nonempty(runtime.version)) incomplete.push(issue("Provider runtime version was not recorded"));
  if (!manifest.sourceHashes || !Object.keys(manifest.sourceHashes).length) incomplete.push(issue("Frozen source hashes are missing"));
  else for (const [path, expected] of Object.entries(manifest.sourceHashes)) {
    if (!sourcePath(path)) issues.push(issue("Source manifest incorrectly includes scaffold or skill content", path));
    if (!hash(expected) || !Object.hasOwn(input.files, path) || digest(input.files[path]) !== expected) issues.push(issue("Frozen source content does not match its manifest", path));
  }
  if (Object.keys(input.files).filter(sourcePath).some(path => !Object.hasOwn(manifest.sourceHashes || {}, path))) issues.push(issue("Available source is missing from the frozen manifest"));
  if (!attempt) incomplete.push(issue("Frozen author attempt is missing"));
  else {
    for (const [field, ceiling] of [["calls", "maxCalls"], ["outputBytes", "maxOutputBytes"], ["writeBytes", "maxWriteBytes"]]) {
      if (!natural(attempt[field]) || !natural(input.budgets?.[ceiling]) || attempt[field] > input.budgets[ceiling]) issues.push(issue("Recorded authoring budget is invalid or exceeded"));
    }
    if (!Array.isArray(attempt.events) || attempt.events.length !== attempt.calls) issues.push(issue("Tool-call evidence is inconsistent"));
    else if (attempt.events.some(event => !natural(event.returnedBytes) || !["allowed", "denied"].includes(event.outcome)) ||
        attempt.events.reduce((sum, event) => sum + event.returnedBytes, 0) !== attempt.outputBytes) issues.push(issue("Returned-byte evidence is inconsistent"));
    if (attempt.writes && Object.values(attempt.writes).every(text => typeof text === "string") &&
        Object.values(attempt.writes).reduce((sum, text) => sum + Buffer.byteLength(text), 0) > attempt.writeBytes) issues.push(issue("Cumulative write-byte evidence is inconsistent"));
    if (attempt.final !== result.final) issues.push(issue("Frozen final text differs from the result"));
    const required = [".agents/skills/ctx/SKILL.md", ".ctx/INDEX.md", ".ctx/local/CONTINUE.md", ...["AGENTS.md", "CLAUDE.md"].filter(path => Object.hasOwn(input.files, path))];
    if (required.some(path => !attempt.fullReads?.includes(path) || !attempt.events?.some(event => event.tool === "ctx_read" && event.path === path && event.outcome === "allowed"))) issues.push(issue("Required entrypoint reads were not observed"));
    try { candidateFiles(trial); } catch { issues.push(issue("Frozen proposal violates the context-only write boundary")); }
  }
  const selected = manifest.runtime;
  if (!selected?.model || !selected.provider) incomplete.push(issue("Explicit runtime selection is missing"));
  else if (selected.kind === "codex") {
    if (runtime.modelQualification?.exactModelAdvertised !== true || (selected.thinking && runtime.modelQualification?.requestedEffortAdvertised !== true)) incomplete.push(issue("Native exact-model qualification was not recorded"));
  } else if (selected.kind === "pi-sdk") {
    if (selected.thinking !== null && selected.thinking !== runtime.effectiveThinking) issues.push(issue("Pi did not preserve the explicit thinking setting"));
    const messages = runtime.messages || [];
    if (!messages.some(message => message.model === selected.model && message.provider === selected.provider)) incomplete.push(issue("Pi exact-model observation is missing"));
    if (messages.some(message => (message.model && message.model !== selected.model) || (message.provider && message.provider !== selected.provider) || message.error || ["error", "aborted", "length"].includes(message.stopReason))) issues.push(issue("Pi recorded a selection mismatch or incomplete response"));
  } else incomplete.push(issue("This runtime is not qualified for provider authoring"));
  return { verdict: verdict(issues, incomplete), issues, incomplete, mode: runtime.mode || "unknown", fixture: manifest.fixture === true,
    deniedCalls: attempt?.events?.filter(event => event.outcome === "denied").length ?? null,
    evidenceReviewFullyRead: attempt?.fullReads?.includes(".agents/skills/ctx/references/evidence-review.md") ?? false,
    requestedModel: selected?.model ?? null, observedVersion: runtime.version ?? null,
    scope: "Trusted adapter observations, not independent wire attestation. Denied requests are recorded separately from actual boundary escapes." };
}

function proseLines(text) {
  let fence = null;
  return text.split(/\r?\n/).flatMap((text, index) => {
    const delimiter = text.match(/^\s{0,3}(`{3,}|~{3,})/);
    if (delimiter) {
      if (!fence) fence = delimiter[1];
      else if (delimiter[1][0] === fence[0] && delimiter[1].length >= fence.length) fence = null;
      return [];
    }
    return fence || /^\s*<!-- ctx:doc /.test(text) ? [] : [{ text, line: index + 1 }];
  });
}

function metadata(text) {
  const markers = [...text.matchAll(/^\s*<!-- ctx:doc (.*?) -->\s*$/gm)];
  if (markers.length !== 1) throw new Error("One metadata marker required");
  const value = JSON.parse(markers[0][1]);
  if (!["draft", "verified", "not-applicable"].includes(value.status) || typeof value.verifiedAt !== "string" || !Array.isArray(value.sources) || value.sources.some(path => typeof path !== "string")) throw new Error("Invalid fact metadata");
  for (const path of value.sources) { safePath(path); if (!sourcePath(path)) throw new Error("Context cannot support itself"); }
  if (value.status === "verified") {
    const match = value.verifiedAt.match(/^([a-f0-9]{40}|[a-f0-9]{64}) @ (\d{4}-\d{2}-\d{2})$/);
    if (!match || new Date(match[2]).toISOString().slice(0, 10) !== match[2] || !value.sources.length) throw new Error("Verified evidence anchor required");
    value.commit = match[1];
  } else if (value.verifiedAt !== "") throw new Error("Nonverified facts must clear verification time");
  return value;
}

function sourceCovered(sources, path, available) {
  return sources.some(source => source === path || (available.some(name => name.startsWith(source + "/")) && path.startsWith(source + "/")));
}

function structure(trial, files, review) {
  const issues = [], incomplete = [], warnings = [], facts = [];
  let config;
  try {
    config = JSON.parse(files[".ctx/config.json"]);
    if (!config || typeof config !== "object" || Array.isArray(config) || !["team", "local"].includes(config.mode) || !natural(config.layoutVersion) || config.layoutVersion < 1) throw new Error("Invalid scaffold descriptor");
  } catch { config = undefined; issues.push(issue("Scaffold configuration is invalid")); }
  if (config && config.layoutVersion !== 3) incomplete.push(issue("This scorecard currently evaluates folder-first guarded-authoring scaffolds only"));
  for (const path of [".ctx/INDEX.md", ".ctx/README.md", ".ctx/config.json", ".ctx/.gitignore", ".ctx/local/CONTINUE.md"]) if (!Object.hasOwn(files, path)) issues.push(issue("Required scaffold file is missing", path));
  if (config?.layoutVersion === 3) for (const category of ["overview", "architecture", "behavior", "caveats", "glossary"]) if (!Object.hasOwn(files, `${factPrefix}${category}/index.md`)) issues.push(issue("Default category index is missing", `${factPrefix}${category}/index.md`));
  const available = Object.keys(trial.manifest.sourceHashes || {});
  const exists = path => Object.hasOwn(files, path) || Object.keys(files).some(name => name.startsWith(path + "/"));
  let recognizedCitations = 0;
  // The router is not a fact document, but broken routes there are still failures.
  for (const [document, text] of Object.entries(files).filter(([path]) => path === ".ctx/INDEX.md" || (path.startsWith(factPrefix) && path.endsWith(".md"))).sort(([a], [b]) => a.localeCompare(b))) {
    let meta;
    if (document.startsWith(factPrefix)) {
      try { meta = metadata(text); } catch { issues.push(issue("Fact metadata is invalid", document)); }
      facts.push({ document, status: meta?.status || "invalid", hash: digest(text) });
    }
    if (meta?.status === "verified" && meta.commit !== trial.input.sourceCommit) issues.push(issue("Verified fact is anchored to a different source commit", document));
    for (const path of meta?.sources || []) if (!available.some(name => name === path || name.startsWith(path + "/"))) incomplete.push(issue("Listed evidence is unavailable in the filtered source view; repository absence is not established", document));
    const citations = new Set(), routes = new Set();
    const seedLines = new Set((trial.input.files[document] || "").split(/\r?\n/));
    let retainedGuidance = false;
    for (const { text: line } of proseLines(text)) {
      // A fresh draft contains conditional add-on instructions, not active
      // routes. Exempt only unchanged coordinator seed blockquotes in drafts;
      // newly authored blockquotes and all verified prose are still checked.
      if (meta?.status === "draft" && /^\s*>/.test(line) && seedLines.has(line)) { retainedGuidance = true; continue; }
      const explicit = line.match(/^\s*(?:[-*]\s+)?(?:\*\*)?Evidence source:(?:\*\*)?\s*`([^`]+)`/);
      if (explicit) { citations.add(explicit[1]); recognizedCitations++; }
      for (const match of line.matchAll(/`([^`]+)`/g)) {
        const token = match[1];
        if (available.includes(token)) citations.add(token);
        if (/^(?:\.\.?\/|\.ctx\/|context\/).*\.md(?:#.*)?$/.test(token) || (/^[a-z0-9-]+\.md$/.test(token) && !available.includes(token))) routes.add(token);
      }
      for (const match of line.matchAll(/\]\(([^\s)]+)(?:\s+"[^"]*")?\)/g)) {
        if (/^[a-z][a-z0-9+.-]*:/i.test(match[1])) {
          if (!/^(?:https?|mailto):/i.test(match[1])) issues.push(issue("Local or executable URL scheme is not a shareable context route", document));
        } else if (!match[1].startsWith("#")) routes.add(match[1]);
      }
    }
    if (retainedGuidance) warnings.push(issue("Unchanged draft template guidance is not an active topic route", document));
    for (const path of citations) {
      const covered = sourceCovered(meta?.sources || [], path, available);
      if (document.startsWith(factPrefix) && !covered) issues.push(issue("Prose citation is not covered by the owning fact's metadata", document));
      if (!available.includes(path)) incomplete.push(issue("Explicit citation is unavailable in the frozen source view", document));
    }
    for (const route of routes) {
      if (/<[a-z][a-z-]*>/i.test(route)) {
        (meta?.status === "verified" ? issues : warnings).push(issue("Unfilled template topic-route placeholder", document));
        continue;
      }
      const path = route.split("#")[0];
      const target = path.startsWith(".ctx/") ? posix.normalize(path) : path.startsWith("context/") ? ".ctx/" + path : available.includes(path) ? path : posix.normalize(posix.join(posix.dirname(document), path));
      if (path.startsWith("/") || target.startsWith("../")) issues.push(issue("Route leaves the repository-relative view", document));
      else if (!exists(target)) {
        if (target.startsWith(".ctx/")) issues.push({ ...issue("Context topic route does not resolve", document), route, target });
        else incomplete.push(issue("Source link is unavailable in the filtered view; repository absence is not established", document));
      }
    }
  }
  if (!recognizedCitations) warnings.push(issue("No explicit Evidence source lines were recognized; this is not citation-completeness certification"));
  if (files[".ctx/README.md"]?.includes("{{OWNER_INSTRUCTIONS_PATH}}")) warnings.push(issue("Managed seed README still has an owner-pointer placeholder"));
  const doctor = trial.result.checks?.find(check => check.command === "doctor");
  if (!doctor) incomplete.push(issue("Recorded doctor check is missing"));
  else if (doctor.code !== 0) issues.push(issue("Recorded doctor check did not pass"));
  if (review?.reportedFactDocuments) {
    const count = review.reportedFactDocuments;
    if (!natural(count.value) || !nonempty(count.quote) || !trial.result.final.includes(count.quote)) incomplete.push(issue("Reported fact-count observation is not anchored in the final text"));
    else if (count.value !== facts.length) issues.push(issue("Reported fact-document count includes nonfact files or otherwise disagrees"));
  }
  return { verdict: verdict(issues, incomplete), issues, incomplete, warnings, facts,
    readiness: { verified: facts.filter(fact => fact.status === "verified").length, draft: facts.filter(fact => fact.status === "draft").length, notApplicable: facts.filter(fact => fact.status === "not-applicable").length, reportedStatusCode: trial.result.checks?.find(check => check.command === "status")?.code ?? null },
    explicitCitations: recognizedCitations,
    citationScope: "Explicit Evidence source lines and inline-code tokens exactly matching available source paths, outside fenced examples. Not arbitrary prose or complete semantic citation coverage. Link anchors and external URLs are not checked." };
}

function anchored(files, location) {
  return location && nonempty(location.path) && natural(location.line) && location.line > 0 && nonempty(location.quote) && typeof files[location.path] === "string" &&
    files[location.path].split(/\r?\n/).slice(location.line - 1, location.line - 1 + location.quote.split(/\r?\n/).length).join("\n").includes(location.quote);
}

function sourceEvidence(trial, files, evidence) {
  return nonempty(evidence?.path) && sourcePath(evidence.path) && Object.hasOwn(trial.manifest.sourceHashes || {}, evidence.path) && anchored(files, evidence);
}

function conditionalReview(files, rules, review, claims, incomplete) {
  const documents = scopeDocuments(files);
  const result = { required: rules.length, reviewed: 0, triggered: 0, notTriggered: 0, unresolved: 0, assessments: [], scopeHash: fingerprint(documents) };
  const supplied = review.applicability ?? [];
  if (!Array.isArray(supplied) || supplied.some(item => !rules.some(rule => rule.title === item?.title))) {
    incomplete.push(issue("Conditional applicability contains invalid or unknown rule assessments"));
    return result;
  }
  for (const rule of rules) {
    const matches = supplied.filter(item => item?.title === rule.title), item = matches[0];
    const boundScope = Array.isArray(item?.documents) && item.documents.every(document => nonempty(document?.path) && hash(document?.hash)) &&
      item.documents.length === documents.length && new Set(item.documents.map(document => document.path)).size === documents.length &&
      documents.every(document => item.documents.some(observed => observed.path === document.path && observed.hash === document.hash));
    const valid = matches.length === 1 && boundScope && nonempty(item.rationale) && ["triggered", "not-triggered", "unresolved"].includes(item.verdict) && Array.isArray(item.claimTitles) &&
      new Set(item.claimTitles).size === item.claimTitles.length && item.claimTitles.every(nonempty) &&
      (item.verdict === "triggered" ? item.claimTitles.length > 0 && item.claimTitles.every(title => claims.some(claim => claim.title === title)) : item.claimTitles.length === 0) &&
      (item.verdict !== "not-triggered" || !claims.some(claim => claim.title === rule.title)) &&
      (item.verdict !== "unresolved" || nonempty(item.missingEvidence));
    if (!valid) { incomplete.push(issue("Conditional audit lacks a unique bound applicability inventory or real reviewed occurrences")); continue; }
    result.reviewed++;
    if (item.verdict === "triggered") result.triggered++;
    else if (item.verdict === "not-triggered") result.notTriggered++;
    else { result.unresolved++; incomplete.push(issue("Conditional audit applicability remains unresolved")); }
    result.assessments.push({ title: rule.title, verdict: item.verdict, rationale: item.rationale, claimTitles: item.claimTitles, inspectedDocuments: documents.length });
  }
  return result;
}

function parentSummaryReview(trial, files, required, review, claims, issues, incomplete) {
  const result = { required, inspectedDocuments: 0, reviewedClaims: 0, cases: 0, counterexamples: 0, unresolved: 0, assessments: [],
    limitation: "Bound case inventories and judgments, not automatic summary detection or proof that all counterexamples were found." };
  if (!required) return result;
  const documents = parentDocuments(files), supplied = review.parentSummaries;
  result.scopeHash = fingerprint(documents);
  if (!Array.isArray(supplied) || supplied.some(item => !documents.some(document => document.path === item?.path))) {
    incomplete.push(issue("Parent-summary document inventory is missing or contains unknown documents")); return result;
  }
  // An explicit case check cannot disappear merely because its owning
  // inventory omits the claim. Discovery of other summaries remains reviewed.
  for (const claim of claims) {
    const raw = review.claims.find(item => item?.title === claim.title);
    if (Object.hasOwn(raw, "counterexampleCheck") && !supplied.some(item => item?.path === claim.document && Array.isArray(item.claimTitles) && item.claimTitles.includes(claim.title))) {
      incomplete.push(issue("A supplied parent case check is not linked by its owning inventory", claim.document));
    }
  }
  for (const document of documents) {
    const inventories = supplied.filter(item => item?.path === document.path), item = inventories[0];
    const bound = inventories.length === 1 && item.hash === document.hash && nonempty(item.rationale) && Array.isArray(item.claimTitles) &&
      new Set(item.claimTitles).size === item.claimTitles.length && item.claimTitles.every(title => nonempty(title) && claims.some(claim => claim.title === title && claim.document === document.path));
    if (!bound) { incomplete.push(issue("Parent-summary inventory is not uniquely bound to real owning claims", document.path)); continue; }
    result.inspectedDocuments++;
    for (const title of item.claimTitles) {
      const claim = claims.find(claim => claim.title === title), raw = review.claims.find(claim => claim?.title === title), check = raw?.counterexampleCheck;
      const valid = nonempty(check?.coverage) && Array.isArray(check?.cases) && check.cases.length > 0 &&
        new Set(check.cases.map(item => item?.title)).size === check.cases.length && check.cases.every(item => nonempty(item?.title) && nonempty(item.rationale) &&
          ["supports", "outside-qualified-scope", "counterexample", "unresolved"].includes(item.relation) && Array.isArray(item.evidence) && item.evidence.length > 0 &&
          item.evidence.every(evidence => sourceEvidence(trial, files, evidence)) && (item.relation !== "unresolved" || nonempty(item.missingEvidence))) &&
        check.cases.some(item => item.relation !== "outside-qualified-scope");
      if (!valid) { incomplete.push(issue("Parent summary lacks a grounded operation/case counterexample check", document.path)); continue; }
      result.reviewedClaims++; result.cases += check.cases.length;
      const counterexamples = check.cases.filter(item => item.relation === "counterexample").length, unresolved = check.cases.filter(item => item.relation === "unresolved").length;
      result.counterexamples += counterexamples; result.unresolved += unresolved;
      if (counterexamples && claim.verdict !== "narrow") issues.push(issue("An observed counterexample is not reflected in the parent claim's narrow verdict", document.path));
      if (unresolved) incomplete.push(issue("Parent-summary counterexample check remains unresolved", document.path));
      claim.evidencePaths = [...new Set([...claim.evidencePaths, ...check.cases.flatMap(item => item.evidence.map(evidence => evidence.path))])];
      result.assessments.push({ title, document: document.path, coverage: check.coverage, cases: check.cases.map(({ title, relation, rationale, evidence }) => ({ title, relation, rationale, evidencePaths: evidence.map(item => item.path) })) });
    }
  }
  return result;
}

function wording(trial, files, structural, rubric, review) {
  const issues = [], incomplete = [], claims = [];
  const required = rubric?.requiredClaims || [];
  const conditional = Array.isArray(rubric?.conditionalClaims) ? rubric.conditionalClaims : [];
  const metrics = { required: required.length, reviewed: 0, supported: 0, narrow: 0, unresolved: 0, unsupportedCertified: 0, unreviewedFactDocuments: [], unreviewedVerifiedDocuments: [] };
  if (rubric?.schemaVersion !== 1 || !nonempty(rubric.title) || !required.length || required.some(claim => !nonempty(claim.title)) || new Set(required.map(claim => claim.title)).size !== required.length) incomplete.push(issue("A versioned, nonempty, coordinator-frozen claim scope is required"));
  if ((rubric?.conditionalClaims !== undefined && !Array.isArray(rubric.conditionalClaims)) || conditional.some(rule => !nonempty(rule?.title) || !nonempty(rule.trigger)) ||
      new Set([...required.map(claim => claim.title), ...conditional.map(rule => rule?.title)]).size !== required.length + conditional.length) incomplete.push(issue("Conditional audit rules need unique descriptive titles and explicit triggers"));
  if (rubric?.parentSummaryAudit !== undefined && typeof rubric.parentSummaryAudit !== "boolean") incomplete.push(issue("Parent-summary audit selection must be a boolean"));
  if (!review) return { verdict: "not-evaluated", issues, incomplete: [...incomplete, issue("No grounded wording review was supplied")], metrics, claims };
  const validHeader = review.schemaVersion === 1 && review.sourceCommit === trial.input.sourceCommit && review.candidateHash === candidateFingerprint(trial) && review.rubricHash === fingerprint(rubric) &&
    ["human", "independent-model", ...(trial.manifest.fixture === true ? ["fixture-reference"] : [])].includes(review.reviewer?.kind) && nonempty(review.reviewer?.name) && nonempty(review.scope) && Array.isArray(review.claims);
  if (!validHeader) return { verdict: "incomplete", issues, incomplete: [...incomplete, issue("Review identity or source/candidate fingerprint does not match this attempt")], metrics, claims };
  const seen = new Set();
  for (const claim of review.claims) {
    if (!claim || typeof claim !== "object" || Array.isArray(claim)) { incomplete.push(issue("Claim review is not an object")); continue; }
    const requirement = required.find(required => required.title === claim.title);
    const document = claim.location?.path;
    const fact = structural.facts.find(fact => fact.document === document);
    const validClaim = nonempty(claim.title) && !seen.has(claim.title) && (!requirement?.document || requirement.document === document) && document?.startsWith(".ctx/") && anchored(files, claim.location) &&
      ["supported", "narrow", "unresolved"].includes(claim.verdict) && nonempty(claim.limits) && ["implementation", "direct-test", "owner-requirement", "owner-decision"].includes(claim.basis) && Array.isArray(claim.evidence) &&
      (claim.verdict === "unresolved" ? nonempty(claim.missingEvidence) : claim.evidence.length > 0) &&
      (claim.verdict !== "narrow" || nonempty(claim.correction)) && claim.evidence.every(evidence => sourceEvidence(trial, files, evidence));
    seen.add(claim.title);
    if (!validClaim) { incomplete.push(issue("Claim review is duplicate, ungrounded, or missing an evidence limit", document)); continue; }
    if (requirement) metrics.reviewed++;
    metrics[claim.verdict]++;
    claims.push({ title: claim.title, document, verdict: claim.verdict, basis: claim.basis, limits: claim.limits, evidencePaths: claim.evidence.map(evidence => evidence.path) });
    if (claim.verdict === "narrow") {
      issues.push(issue("Reviewed wording exceeds its cited evidence", document));
      if (fact?.status === "verified") metrics.unsupportedCertified++;
    } else if (claim.verdict === "unresolved") incomplete.push(issue("Material wording remains unresolved", document));
  }
  for (const requiredClaim of required) if (!claims.some(claim => claim.title === requiredClaim.title)) incomplete.push(issue("Required material claim was not validly reviewed", requiredClaim.document));
  const applicability = conditionalReview(files, conditional, review, claims, incomplete);
  const parentSummaries = parentSummaryReview(trial, files, rubric?.parentSummaryAudit === true, review, claims, issues, incomplete);
  const applicabilityEvidence = [];
  for (const fact of structural.facts) {
    const inventory = Array.isArray(review.documents) ? review.documents.find(document => document?.path === fact.document && document.hash === fact.hash && document.materialClaimsInventoried === true) : null;
    const applicability = inventory?.notApplicable;
    const validApplicability = fact.status !== "not-applicable" || (nonempty(applicability?.rationale) && Array.isArray(applicability.evidence) && applicability.evidence.length > 0 &&
      applicability.evidence.every(evidence => sourceEvidence(trial, files, evidence)));
    if (!inventory || !validApplicability || (fact.status !== "not-applicable" && !claims.some(claim => claim.document === fact.document))) {
      metrics.unreviewedFactDocuments.push(fact.document);
      if (fact.status === "verified") metrics.unreviewedVerifiedDocuments.push(fact.document);
      incomplete.push(issue("Fact document lacks a bound material-claim or applicability review", fact.document));
    }
    if (inventory && fact.status === "not-applicable" && validApplicability) applicabilityEvidence.push({ document: fact.document, evidencePaths: applicability.evidence.map(evidence => evidence.path) });
  }
  return { verdict: verdict(issues, incomplete), issues, incomplete, metrics, claims, applicability, parentSummaries, applicabilityEvidence, reviewer: review.reviewer, scope: review.scope,
    limitation: "A grounded reviewer assessment, not machine proof of prose. Material-claim inventory and conditional applicability completeness are reviewer attestations; unlisted sentences are not automatically certified." };
}

function reconcileReviewedSources(trial, files, structural, wording) {
  const available = Object.keys(trial.manifest.sourceHashes || {}), inspected = new Map(), gaps = [];
  // Only wording's already-bound, source-grounded entries reach this check.
  // This catches unnamed evidence without trusting a stale review or guessing
  // which files arbitrary prose refers to. Semantic validity remains reviewed.
  for (const entry of [...wording.claims, ...(wording.applicabilityEvidence || [])]) {
    if (!structural.facts.some(fact => fact.document === entry.document)) continue;
    let meta;
    try { meta = metadata(files[entry.document]); } catch { continue; } // already a structural failure
    for (const path of new Set(entry.evidencePaths)) {
      const key = entry.document + "\0" + path;
      if (inspected.has(key)) continue;
      const covered = sourceCovered(meta.sources, path, available);
      inspected.set(key, { document: entry.document, evidencePath: path, covered });
      if (!covered) {
        gaps.push({ document: entry.document, evidencePath: path });
        structural.issues.push({ ...issue("Grounded review evidence is not covered by the owning fact's metadata", entry.document), evidencePath: path });
      }
    }
  }
  structural.reviewedEvidence = { checkedDocumentSourcePairs: inspected.size, gaps,
    scope: "Evidence paths from validly bound and grounded claim/applicability reviews, including unnamed prose references. Missing or partial reviews are not complete source inventories." };
  structural.verdict = verdict(structural.issues, structural.incomplete);
}

function readerCitations(trial, observation) {
  const audit = observation.citationReview;
  if (!audit || audit.answerHash !== digest(observation.final) || audit.materialClaimsInventoried !== true || !nonempty(audit.scope) ||
      !Array.isArray(audit.claims) || !audit.claims.length) return { verdict: "incomplete" };
  let unsupported = 0, unresolved = 0, reviewed = 0, invalid = false;
  const titles = new Set();
  for (const claim of audit.claims) {
    if (!claim || !nonempty(claim.title)) { invalid = true; continue; }
    if (titles.has(claim.title)) invalid = true;
    titles.add(claim.title);
    if (!["supported", "unsupported", "unresolved"].includes(claim.verdict) || !nonempty(claim.answerQuote) || !observation.final.includes(claim.answerQuote) ||
        !nonempty(claim.rationale) || !Array.isArray(claim.evidence) ||
        (claim.verdict === "unresolved" ? !nonempty(claim.missingEvidence) : !claim.evidence.length) ||
        claim.evidence.some(evidence => !sourceEvidence(trial, trial.input.files, evidence))) { invalid = true; continue; }
    reviewed++;
    if (claim.verdict === "unsupported") unsupported++;
    if (claim.verdict === "unresolved") unresolved++;
  }
  return { verdict: invalid || unresolved ? "incomplete" : unsupported ? "failed" : "passed", reviewed, unsupported, unresolved };
}

function downstream(trial, comparison) {
  const issues = [], incomplete = [];
  if (!comparison) return { verdict: "not-evaluated", issues, incomplete: [issue("No matched downstream observations were supplied")] };
  const { protocol, pairs } = comparison;
  const citationsRequired = protocol?.schemaVersion === 2;
  const versionedPolicy = citationsRequired ? protocol.citationPolicy === "all-material-claims" :
    (protocol?.schemaVersion === undefined || protocol.schemaVersion === 1) && !Object.hasOwn(protocol || {}, "citationPolicy");
  const validProtocol = comparison.schemaVersion === 1 && nonempty(comparison.reviewer?.name) && nonempty(comparison.scope) &&
    versionedPolicy &&
    ["human", "independent-model", ...(trial.manifest.fixture === true ? ["fixture-reference"] : [])].includes(comparison.reviewer?.kind) &&
    protocol?.sourceCommit === trial.input.sourceCommit && nonempty(protocol.reader?.model) && nonempty(protocol.reader?.provider) && nonempty(protocol.reader?.runtime) && nonempty(protocol.reader?.version) && Object.hasOwn(protocol.reader, "thinking") &&
    nonempty(protocol.budgetFingerprint) && Array.isArray(protocol.tasks) && protocol.tasks.length > 0 && protocol.tasks.every(task => task && typeof task === "object") && new Set(protocol.tasks.map(task => task.title)).size === protocol.tasks.length &&
    protocol.tasks.every(task => nonempty(task.title) && ["read-only", "change"].includes(task.kind) && Array.isArray(task.requiredOutcomes) && task.requiredOutcomes.length > 0 && task.requiredOutcomes.every(nonempty) && new Set(task.requiredOutcomes).size === task.requiredOutcomes.length) &&
    natural(protocol.repetitions) && protocol.repetitions >= 3 && ["calls", "returnedBytes", "totalTokens", "elapsedMs"].includes(protocol.efficiencyMetric) && Number.isFinite(protocol.minimumImprovementFraction) && protocol.minimumImprovementFraction > 0 && protocol.minimumImprovementFraction < 1 && Array.isArray(pairs);
  if (!validProtocol) return { verdict: "incomplete", issues, incomplete: [issue("Matched downstream protocol is missing, invalid, or below three repetitions") ] };
  const protocolHash = fingerprint(protocol), candidateHash = candidateFingerprint(trial), observations = [], identities = new Set(), cells = new Set();
  let baselinePassed = 0, contextPassed = 0, expected = 0, baselineMetric = 0, contextMetric = 0;
  let baselineCitationsPassed = 0, contextCitationsPassed = 0, citationExpected = 0;
  let baselineFullyCorrect = 0, contextFullyCorrect = 0;
  for (const pair of pairs) {
    if (!pair || typeof pair !== "object") { incomplete.push(issue("Downstream pair is malformed")); continue; }
    const task = protocol.tasks.find(task => task.title === pair.task);
    const cell = `${pair.task}\0${pair.repetition}`;
    if (!task || !natural(pair.repetition) || pair.repetition < 1 || pair.repetition > protocol.repetitions || cells.has(cell) || !["baseline-first", "context-first"].includes(pair.order)) { incomplete.push(issue("Downstream task, repetition, or ordering is invalid or duplicated")); continue; }
    cells.add(cell);
    let valid = true;
    const citationAssessments = {};
    for (const [condition, observation] of [["baseline", pair.baseline], ["context", pair.context]]) {
      // Review annotations do not belong to the reader's frozen runtime record.
      // Preserve the legacy hash contract for historical protocols.
      const { artifactHash, outcomes, ...record } = observation || {};
      if (citationsRequired) delete record.citationReview;
      if (!observation || observation.protocolHash !== protocolHash || !hash(observation.artifactHash) || identities.has(observation.artifactHash) || observation.candidateHash !== (condition === "context" ? candidateHash : null) ||
          !nonempty(observation.recordLabel) || !nonempty(observation.final) || artifactHash !== fingerprint(record)) {
        valid = false; incomplete.push(issue("Downstream observation lacks a matching frozen protocol, candidate, or reader record")); continue;
      }
      identities.add(observation.artifactHash);
      if (typeof observation.completed !== "boolean" || typeof observation.scopePreserved !== "boolean" || !natural(observation.writeAttempts) || !natural(observation.metrics?.[protocol.efficiencyMetric])) {
        valid = false; incomplete.push(issue("Downstream completion, scope, or metric evidence is malformed"));
      }
      // Grade independently bound annotations even if a neighboring inventory
      // is missing. Incompleteness must not hide a known grounded failure.
      const entries = Array.isArray(outcomes) ? outcomes : [];
      const groundedOutcomes = entries.map(outcome => outcome && task.requiredOutcomes.includes(outcome.title) && ["supported", "unsupported", "unresolved"].includes(outcome.verdict) &&
        nonempty(outcome.answerQuote) && observation.final.includes(outcome.answerQuote) && nonempty(outcome.rationale) && Array.isArray(outcome.evidence) &&
        (outcome.verdict === "unresolved" ? nonempty(outcome.missingEvidence) : outcome.evidence.length > 0) &&
        outcome.evidence.every(evidence => sourceEvidence(trial, trial.input.files, evidence)));
      if (entries.length !== task.requiredOutcomes.length || new Set(entries.map(outcome => outcome?.title)).size !== task.requiredOutcomes.length || groundedOutcomes.some(grounded => !grounded)) {
        valid = false; incomplete.push(issue("Downstream outcome review lacks an answer quote, source grounding, or reviewer rationale"));
      }
      if (condition === "context" && entries.some((outcome, index) => outcome?.verdict === "unsupported" && groundedOutcomes[index])) issues.push(issue("Context-assisted work failed required correctness outcomes"));
      if (observation.completed === false || observation.scopePreserved === false || (task.kind === "read-only" && natural(observation.writeAttempts) && observation.writeAttempts !== 0)) { valid = false; issues.push(issue("Downstream task did not complete within its authorized scope")); }
      if (entries.some(outcome => outcome?.verdict === "unresolved")) { valid = false; incomplete.push(issue("Downstream quality has unreviewed outcomes")); }
      if (citationsRequired) {
        citationAssessments[condition] = readerCitations(trial, observation);
        if (citationAssessments[condition].verdict === "incomplete") { valid = false; incomplete.push(issue("Downstream citation accuracy lacks a complete answer-bound grounded review")); }
        if (condition === "context" && citationAssessments[condition].unsupported > 0) issues.push(issue("Context-assisted work contains unsupported source attributions"));
      } else if (Object.hasOwn(observation, "citationReview")) {
        valid = false; incomplete.push(issue("A citation review requires the frozen citation-aware protocol"));
      }
    }
    if (!valid) continue;
    const baselineQuality = pair.baseline.outcomes.filter(outcome => outcome.verdict === "supported").length;
    const contextQuality = pair.context.outcomes.filter(outcome => outcome.verdict === "supported").length;
    baselinePassed += baselineQuality; contextPassed += contextQuality; expected += task.requiredOutcomes.length;
    if (citationsRequired) {
      citationExpected++;
      baselineCitationsPassed += Number(citationAssessments.baseline.verdict === "passed");
      contextCitationsPassed += Number(citationAssessments.context.verdict === "passed");
    }
    const baselineCorrect = baselineQuality === task.requiredOutcomes.length && (!citationsRequired || citationAssessments.baseline.verdict === "passed");
    const contextCorrect = contextQuality === task.requiredOutcomes.length && (!citationsRequired || citationAssessments.context.verdict === "passed");
    baselineFullyCorrect += Number(baselineCorrect); contextFullyCorrect += Number(contextCorrect);
    baselineMetric += pair.baseline.metrics[protocol.efficiencyMetric]; contextMetric += pair.context.metrics[protocol.efficiencyMetric];
    observations.push({ task: pair.task, repetition: pair.repetition, order: pair.order, baselineQuality, contextQuality,
      ...(citationsRequired ? { baselineCitations: citationAssessments.baseline, contextCitations: citationAssessments.context } : {}),
      baselineMetric: pair.baseline.metrics[protocol.efficiencyMetric], contextMetric: pair.context.metrics[protocol.efficiencyMetric] });
  }
  for (const task of protocol.tasks) {
    for (let repetition = 1; repetition <= protocol.repetitions; repetition++) if (!observations.some(observation => observation.task === task.title && observation.repetition === repetition)) incomplete.push(issue("Matched downstream task repetition is missing"));
    if (new Set(observations.filter(observation => observation.task === task.title).map(observation => observation.order)).size < 2) incomplete.push(issue("Downstream ordering was not counterbalanced for each task"));
  }
  const comparable = expected > 0 && baselinePassed === expected && contextPassed === expected &&
    (!citationsRequired || (baselineCitationsPassed === citationExpected && contextCitationsPassed === citationExpected)) && !issues.length && !incomplete.length;
  const improvementFraction = comparable && baselineMetric > 0 ? (baselineMetric - contextMetric) / baselineMetric : null;
  const qualityImproved = !issues.length && !incomplete.length && contextPassed === expected &&
    (!citationsRequired || contextCitationsPassed === citationExpected) && (contextPassed > baselinePassed || contextFullyCorrect > baselineFullyCorrect);
  const observedBenefit = qualityImproved || (improvementFraction !== null && improvementFraction >= protocol.minimumImprovementFraction);
  return { verdict: issues.length ? "failed" : incomplete.length ? "incomplete" : observedBenefit ? "passed" : "not-demonstrated", issues, incomplete,
    quality: { baselinePassed, contextPassed, expected, qualityImproved,
      citationAccuracy: { required: citationsRequired, baselinePassed: citationsRequired ? baselineCitationsPassed : null, contextPassed: citationsRequired ? contextCitationsPassed : null, expected: citationsRequired ? citationExpected : null },
      fullyCorrectAnswers: citationsRequired ? { baseline: baselineFullyCorrect, context: contextFullyCorrect, expected: observations.length } : null },
    efficiency: { metric: protocol.efficiencyMetric, baselineTotal: baselineMetric, contextTotal: contextMetric, comparableQuality: comparable, improvementFraction, requiredImprovementFraction: protocol.minimumImprovementFraction }, observations,
    reviewer: comparison.reviewer, scope: comparison.scope,
    limitation: "Trusted grounded outcome assessments and descriptive repeated pairs, not statistical significance, a model ranking, or causal attribution. Citation inventories and semantic verdicts are reviewer attestations, not automatic citation discovery. Legacy protocols did not grade citation accuracy and cannot establish fully correct answers. Record fingerprints detect mismatch, not fabricated independent runs. Upfront authoring and maintenance costs are not included." };
}

export function evaluateScorecard(trial, { rubric, review, comparison } = {}) {
  if (!trial?.input?.files || !fullCommit(trial.input.sourceCommit) || !trial.manifest || !trial.result) throw new Error("Frozen input, manifest, and result are required");
  for (const [path, text] of Object.entries(trial.input.files)) { safePath(path); if (typeof text !== "string") throw new Error("Frozen files must be text"); }
  const successfulExecution = execution(trial);
  let files;
  try { files = candidateFiles(trial); } catch {
    const unavailable = () => ({ verdict: "not-evaluated", issues: [], incomplete: [issue("Proposal is outside the authorized context view")] });
    return { schemaVersion: 1, scorecardVersion, sourceCommit: trial.input.sourceCommit, candidateHash: null,
      stages: { successfulExecution, structurallyValidContext: unavailable(), supportedWording: unavailable(), usefulDownstreamResults: unavailable() },
      promotion: { eligibleForReviewedUse: false, blockers: ["Invalid context proposal"], limitation: "Not merge permission or human approval." }, noCompositeScore: true };
  }
  const structurallyValidContext = structure(trial, files, review);
  const supportedWording = wording(trial, files, structurallyValidContext, rubric, review);
  reconcileReviewedSources(trial, files, structurallyValidContext, supportedWording);
  const usefulDownstreamResults = downstream(trial, comparison);
  const blockers = [["Successful execution", successfulExecution], ["Structurally valid context", structurallyValidContext], ["Supported wording", supportedWording]].filter(([, stage]) => stage.verdict !== "passed").map(([title, stage]) => `${title}: ${stage.verdict}`);
  if (structurallyValidContext.readiness.draft) blockers.push("Draft fact documents remain");
  return { schemaVersion: 1, scorecardVersion, sourceCommit: trial.input.sourceCommit, candidateHash: candidateFingerprint(trial), rubricHash: rubric ? fingerprint(rubric) : null,
    stages: { successfulExecution, structurallyValidContext, supportedWording, usefulDownstreamResults },
    promotion: { eligibleForReviewedUse: blockers.length === 0, blockers, limitation: "Not merge permission or human approval. Downstream success never waives an upstream failure; draft context may still be studied as explicitly untrusted experimental input." },
    noCompositeScore: true };
}
