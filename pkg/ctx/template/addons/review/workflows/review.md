# Review workflow — {{PROJECT}}

<!-- ctx:managed begin review-workflow -->
This optional workflow is tool-neutral. Project-owned instructions determine
when review is required, which base or artifact to review, and who may approve
changes.

1. Finish the scoped change and run its required local verification.
2. Select an explicit review target according to the project workflow.
3. Examine correctness, data loss, security, compatibility, failure handling,
   concurrency where relevant, and meaningful test gaps.
4. Report findings by priority with concrete evidence and tight source ranges.
5. Separate confirmed defects from questions and environment limitations.
6. Treat findings as advisory: do not broaden or mutate the change without the
   authorization required by the owner.

Do not hard-code a model, service, branch name, or hosting provider here unless
the project has deliberately standardized it below.
<!-- ctx:managed end review-workflow -->

## Project review profile

- **When required:** [fill]
- **Canonical target/base:** [fill]
- **Tool or command:** [fill]
- **Required checks first:** [fill]
- **High-risk project areas:** [fill]
- **Known environment constraints:** [fill with evidence, or none]
