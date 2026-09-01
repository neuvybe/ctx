# Releasing ctx

`ctx` uses semantic-release on pushes to `main`. Release commits, version files,
the changelog, archives, GitHub releases, and npm publishing are automated by
the release workflow.

## Pre-1.0 version policy

Until the project deliberately declares a stable `1.0.0` contract:

- backward-compatible fixes increment the patch version;
- features increment the minor version;
- incompatible CLI, API, layout, or visibility changes also increment the minor
  version and must include explicit migration guidance.

The commit analyzer therefore maps a conventional-commit
`BREAKING CHANGE:` footer to a minor release during `0.x`. The footer is still
required so generated release notes identify the incompatibility. A `!` in the
commit header alone is not sufficient with the repository's current parser.

Before releasing `1.0.0`, remove the pre-1.0 breaking-change override from
`.releaserc.json`; after that point, breaking changes must increment the major
version.

## Release checklist

1. Use conventional commits. Put user-facing migration text in an exact
   `BREAKING CHANGE:` footer for an incompatible change.
2. Run the full Go race suite, vet, smoke tests, release-script syntax checks,
   and package dry run.
3. Push the reviewed commit to `main`. Do not edit package versions or
   `CHANGELOG.md` manually; semantic-release owns them.
4. Verify the GitHub release assets and published npm package after the release
   workflow completes.

The migration guide for the team-mode default is
[`migrations/0.2-team-mode.md`](migrations/0.2-team-mode.md).
The migration guide for the incompatible layout-v2/API changes is
[`migrations/0.3-layout-v2.md`](migrations/0.3-layout-v2.md).
