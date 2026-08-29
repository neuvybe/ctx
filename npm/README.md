# @neuvybe/ctx

`ctx` — a reusable agent-context platform. Scaffolds a **private `.ctx/` folder** (gitignored via `.git/info/exclude`, not `.gitignore`) into a repo, holding a constitution-vs-log split of governing files plus a reference-context schema for the project. Survives context compaction; makes resuming a session cheap.

## Install

```bash
npm install -g @neuvybe/ctx
ctx --version
```

The postinstall step fetches the matching OS/arch `ctx` binary from the [GitHub release](https://github.com/neuvybe/ctx/releases) for this package's version and extracts it next to the package. If the fetch fails it warns and exits 0 (your `npm install` won't break); fall back to:

```bash
go install github.com/neuvybe/ctx/cmd/ctx@latest
```

## Commands

- `ctx init [target] [-f .ctx]` — scaffold `.ctx/` from embedded templates (`{{PROJECT}}`/`{{DATE}}` substituted, `.ctx-version` stamp, `.git/info/exclude` wired).
- `ctx --version`
- `ctx update` — refresh the platform-managed files (`README.md`, `REVIEW.md`) by swapping `<!-- ctx:managed -->` block content, preserving all your content; bumps `.ctx-version`.
- `ctx doctor` — validate `.ctx/` (folder, version stamp, exclude entry, no leftover placeholders, balanced markers, expected files).
- `ctx upgrade` — self-replace the binary from the latest GitHub release (matching OS/arch); prints the right command for npm/brew/go-install.

## Why `.ctx/` and not the repo's `.gitignore`

`.ctx/` is the **collaborator's private** working context — ignored via the target repo's `.git/info/exclude` (repo-local, non-tracked), never the repo's `.gitignore`, and never the project owner's tracked instruction namespace (root `CLAUDE.md`, `AGENTS.md`, `.claude/skills/`). It complements the owner's rules, never overrides them.

See the [project README](https://github.com/neuvybe/ctx#readme) for the full design + principles.