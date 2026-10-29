#!/usr/bin/env node
// Read-only evaluation of frozen artifacts. Never starts or resumes an agent.
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { evaluateScorecard } from "./lib/context-scorecard.mjs";
import { readFrozenTrial, readScorecardJson, writeScorecardJson } from "./lib/scorecard-io.mjs";

export async function main(args = process.argv.slice(2)) {
  const options = new Map(); let requireReviewedContext = false;
  for (let index = 0; index < args.length; index++) {
    if (args[index] === "--require-reviewed-context" && !requireReviewedContext) requireReviewedContext = true;
    else if (["--trial", "--rubric", "--review", "--comparison", "--output"].includes(args[index]) && args[index + 1] && !options.has(args[index])) options.set(args[index], args[++index]);
    else throw new Error("Invalid scorecard arguments");
  }
  const trial = await readFrozenTrial(options.get("--trial"));
  const supplements = {};
  for (const name of ["rubric", "review", "comparison"]) if (options.has("--" + name)) supplements[name] = await readScorecardJson(options.get("--" + name));
  const report = evaluateScorecard(trial, supplements);
  if (options.has("--output")) await writeScorecardJson(options.get("--output"), report);
  console.log(JSON.stringify(report, null, 2));
  if (requireReviewedContext && !report.promotion.eligibleForReviewedUse) process.exitCode = 1;
  return report;
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main().catch(() => { console.error("Scorecard data could not be evaluated; inputs were not modified and private error details are omitted"); process.exitCode = 2; });
