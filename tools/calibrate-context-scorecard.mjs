#!/usr/bin/env node
// Executes only fixed public fixtures, never a model or subject-repository code.
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { calibrationCases } from "../evals/ctx-skill/fixtures/scorecard/cases.mjs";
import { evaluateScorecard, scorecardVersion } from "./lib/context-scorecard.mjs";
import { writeScorecardJson } from "./lib/scorecard-io.mjs";

export async function calibrateScorecard() {
  const results = [];
  for (const item of await calibrationCases()) {
    const report = evaluateScorecard(item.trial, item);
    const actual = Object.fromEntries(Object.entries(report.stages).map(([name, stage]) => [name, stage.verdict]));
    const matched = Object.entries(item.expected).every(([name, value]) => actual[name] === value);
    results.push({ title: item.title, matched, expected: item.expected, actual, promotion: report.promotion, issues: Object.fromEntries(Object.entries(report.stages).map(([name, stage]) => [name, [...stage.issues, ...stage.incomplete]])) });
  }
  return { schemaVersion: 1, scorecardVersion, modelFree: true, subject: "Public synthetic entry-store fixtures; no private project source", passed: results.filter(result => result.matched).length, failed: results.filter(result => !result.matched).length, results,
    limitation: "Known positive, negative, and unknown controls calibrate this scorecard and finite reference oracle. They do not prove general semantic detection or real downstream efficiency." };
}

async function main() {
  const args = process.argv.slice(2);
  if (args.length && (args.length !== 2 || args[0] !== "--output")) throw new Error("Usage: [--output /absolute/new-report.json]");
  const report = await calibrateScorecard();
  if (args.length) await writeScorecardJson(args[1], report);
  console.log(JSON.stringify(report, null, 2));
  if (report.failed) process.exitCode = 1;
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main().catch(() => { console.error("Scorecard calibration failed; no model was invoked"); process.exitCode = 2; });
