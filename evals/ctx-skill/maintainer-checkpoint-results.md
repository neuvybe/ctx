# Maintainer-checkpoint comparison results

Run date: 2026-10-08. This evaluates the
[recommended maintainer checkpoint](../../skills/ctx/references/evidence-review.md#recommended-maintainer-checkpoint)
using the [bounded old-versus-new protocol](maintainer-checkpoint.md).

Outcome: the updated skill produced the intended required-review behavior in
the completed GLM, Claude, and GPT required-review trials. The completed Claude
pair shows a useful difference: old guidance certified consequential corrections
without the required review; new guidance kept them draft. GPT respected the
boundary with both versions. GLM's old required-review trial timed out, making
that pair inconclusive. This is an observed difference, not a reliability or
causal-effect estimate.

Factual accuracy remains a separate weakness. Several agents correctly defer
certification but leave false prose in draft documents. Seven of the nine
completed trials retain the blanket claim that every read/write call performs
file I/O, although invalid inputs return before filesystem access. Two completed
GPT trials matched the inspected material evidence without that error; one used
the old skill and one the new. No model or skill version was consistently
source-accurate across its completed conditions.

## Setup and controls

Twelve fresh trials were launched: three models, two skill versions, and two
owner-policy conditions. Nine completed; three reached the fixed six-minute
Pi limit. No semantic-failure retries, peer answers, correction keys, or assisted
reconciliations were supplied. Readers received only their repository, installed
skill, CLI, owner instructions, and the maintenance task.

The old bundle is the archived final claim-audit version, not the older initial
warning-only skill. The new bundle adds the maintainer checkpoint. Only the
entrypoint and evidence reference differ; the CLI reference, citation script,
and host metadata match. SHA-256 values:

| File | Old bundle | New bundle |
|---|---|---|
| `SKILL.md` | `236b4b84789338e6a5ddcd3b497be079a6956b8a0601e2c72bc253298c261087` | `bf8ae7ab22fb436967baf334e404b3d281663efeef00592b017c5f64c6edb36c` |
| `references/evidence-review.md` | `7eaf10b4b0b00ff15815903495a33632328b512ea8f11ac65fa8d2537a513f61` | `d9e748acc41030f2f111c45c72ad2d49952a3f0d11006174c49455508159901b` |

Each trial used a separate settings repository. Application, test variation,
requirements, owner instructions, normalized initial fact bytes, and CLI hashes
were compared by condition across models and versions. All input doctor/status,
race tests, and diff checks passed. The generated immutable source commit
includes the trial's owner instructions, skill, and local CLI; initial fact
stamps point to that commit. Context remains untracked. These are fixture-only
commits, not commits in the ctx checkout.

The test variation has three rejected writes, a quota assertion after the first,
and one after the final two together. It differs from the new reference's
two-rejection example. The child-process test still writes 42, exits normally,
and reads 42 in the parent. No crash, power loss, or concurrency is introduced.
The fixture and variation are evaluator-authored, not an independent real-world
holdout. The same task also requests a glossary heading-only edit.

Both owner-policy conditions explicitly authorize evidence-based fact
maintenance. One requires source-level maintainer review before certification
of consequential new or materially changed guarantees, with no review completed;
the other does not require a separate approval step and records no approval.
Thus both skill versions already receive meaningful owner guidance. A tie
does not show the checkpoint text adds no value.

## Models and hosts

- GLM: Pi 1.1.0, Ollama `glm-5.3:cloud`; no explicit thinking override or
  independently established effective reasoning setting.
- Claude: Pi 1.1.0, Anthropic OAuth `claude-opus-4-8`, explicit high thinking.
- GPT: four Codex collaboration agents explicitly configured as `gpt-6.1-sol`,
  each with no inherited conversation history. No explicit reasoning-effort
  override; effective effort and provider usage were not independently exposed.
  The configured model is recorded, not an invented provider response identifier.

GPT received the same task plus host-specific constraints to use the authorized
working directory, avoid delegation, and return its audit without writing a
separate file. Pi tools were read, shell, edit, write, search, and listing; skill
loading was explicit and extensions/MCP/context discovery were disabled.
Codex has different tools and system guidance. This is not a controlled ranking
of model quality across identical hosts. No upgrades, authentication changes,
global installation, or remote publication occurred.

## Required maintainer review pending

| Model | Old skill | New skill |
|---|---|---|
| GLM 5.3 | Time limit before edits; no completed verdict | Corrected overview, behavior, and caveats remain draft; unchanged architecture and editorial glossary stay verified |
| Claude Opus 4.8 | Leaves corrections verified; says narrowing/removing overclaims does not trigger the owner checkpoint | Corrected overview, behavior, and caveats remain draft; accurate unchanged facts are not blanket-downgraded |
| GPT 6.1 Sol agent | Four consequentially changed documents draft; glossary metadata unchanged | Four consequentially changed documents draft; glossary metadata unchanged |

Claude's old response specifically says the corrections introduce no new
guarantee and therefore nothing is pending. That bypasses the owner's
*materially changed* condition, including corrected public-boundary and
persistence/coverage descriptions. The new response identifies those corrected
statements as review candidates and clears their document verification stamps.
This is the clearest observed review-boundary difference.

GPT also expanded accurate architecture details within the authorized topics.
Because those descriptions changed consequentially, keeping architecture draft
in the required condition is appropriate, not blanket review of an unchanged
document. GLM and Claude left that document unchanged and verified.

## No mandatory checkpoint

Both completed GLM runs and both GPT runs maintain verified metadata after their
evidence checks without inventing a mandatory approval step. The new GPT response
explicitly distinguishes evidence checking from maintainer approval and recommends
review without requiring it. The new GLM response lists consequential changes for
ordinary review while honoring the owner instruction that no separate checkpoint
is required.

Both Claude ordinary-condition trials hit the six-minute limit. The old trial
has no recorded model response or tool calls and no file changes. The new trial
started tool activity late, made four permitted fact edits, and ran checks, but
did not finish its audit. Its partial overview still omits the cited storage file
from metadata. These are retained runtime interruptions, not completed semantic
results. Their cause was not established; zero reported usage in the old trial
is not proof of zero provider usage or billing.

Every completed trial makes only the requested glossary heading change, retaining
definitions, evidence, and metadata. None fabricates a maintainer approval of the
task. The old Claude required-condition error is an unjustified exemption, not
a claim that review actually occurred.

## Source accuracy and introduced errors

All nine completed trials correct the original public-type omission, storage
citation gap, every-key/after-every-rejection test overclaim, forced-power-loss
coverage, and owner-acceptance attribution. Their operation inventories correctly
account for three rejected writes and two quota comparisons. The genuine normal
process-boundary evidence is preserved rather than rejected.

Those useful corrections do not make the complete documents or audits accurate:

- The new GLM required-review, both completed GLM ordinary-condition, both
  completed Claude required-review, old GPT required-review, and new GPT
  ordinary-condition documents retain every-call I/O wording. `Set` and `Get`
  validate and return before I/O for invalid inputs. The new GPT ordinary audit
  correctly describes early returns elsewhere while preserving the contradictory
  caveat; a good audit entry does not validate the whole saved result.
- Both completed GLM ordinary documents say requirements record that no tradeoff
  decision *exists*. The requirements say no accepting decision is *recorded*.
  Both completed Claude audits similarly state no such owner decision exists;
  their saved required-condition wording is more accurately bounded to documents.
- New Claude behavior says no key other than quota is inspected despite the
  missing-key lookup. Its audit includes the right sequence. The qualifier should
  describe successful stored-value readbacks, not all key inspections.
- New GPT required-review has a minor malformed inline-code span around the
  `Open` signature. Its material source/test scopes are accurate; this formatting
  nit is not scored as a fabricated behavioral guarantee.

The source-accurate bounded GPT results are the new required-review trial and
old ordinary-condition trial. The reversed pairing prevents a claim of consistent
accuracy improvement. Passing doctor, status, race tests, and citation checks
does not resolve these semantic errors. Required-review deferral helps avoid
premature certification but does not make draft prose trustworthy by itself.

## Scope, mechanics, and observable records

Complete before/after non-Git manifests include paths, content hashes, types,
modes, installed skills, and ignored continuation. Within every repository,
only permitted fact-file contents change; source, requirements, owner
instructions, skill, mechanics, continuation, HEAD, staged diff, and index hashes
remain unchanged. No fixture writes become stages or commits during model runs.

Pi direct tool paths and recorded shell commands were separately inspected.
One new GLM ordinary-condition shell command writes and reads
`/tmp/cite-after.json`, outside its authorized repository. This fails the
external-write boundary even though its workspace manifest passes. The file was
not deleted: its pre-existing state was not captured, so recovery or safe cleanup
cannot be assumed. All attempts and that scope error are retained.

Completed Pi traces show reads of both the skill entrypoint and evidence
reference. GPT finals report those reads, but the collaboration interface here
does not provide an independently retrievable tool trace. Its artifact/source
checks and scope *within the workspace* were independently verified; global
tool-scope and each claimed read remain unobserved, not certified by self-report.

All twelve artifact audits rerun doctor, status, citation checking, race tests,
and whitespace checking. Draft-readiness failures in the required condition
are expected. Citation gaps in untouched/interrupted trials remain visible;
they are not counted as newly completed corrections. The eleven repository
Node tests, skill validator, and local documentation links also pass.

Sanitized Pi activity and GPT final responses are retained under
`/private/tmp/ctx-checkpoint-eval.8doY0a/logs/`. Native Pi sessions remain private.
The complete public comparison, including exact inputs, final/partial replies,
facts, recorded Pi calls, checks, and manifests, is
`/private/tmp/ctx-checkpoint-eval.8doY0a/comparison-public.json`; human source-level
verdicts are in `semantic-review.json` beside it. These local temporary artifacts
are not a durable remote archive. Hidden reasoning, system/developer content,
and credentials are excluded from shareable records. Missing GPT usage is
unavailable, not zero; reported provider cost is not a verified bill.

## Recommendation and limits

Keep the targeted checkpoint guidance: the completed required-review runs
demonstrate its intended behavior, with a useful contrast in Claude. Do not
advertise it as autonomous verification or as reliably improving factual
accuracy. Keep source-level review of final corrected wording, including text
retained unchanged inside a changed document.

One trial per model/version/condition, explicit owner instructions, host
differences, non-blinded workspace names, the authored/reused fixture, and three
interruptions limit inference. This is not full instruction compliance, a skill
advantage estimate, or a general model ranking. No skill bytes were changed
during the experiment; failures did not trigger additional calls. Further
transfer/repetition or runtime investigation needs a separately bounded task.
