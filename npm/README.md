# @neuvybe/ctx

`ctx` gives coding agents durable project context that survives compaction. By
default, it creates a **team scaffold**: stable guidance and reference docs are
available to track, while the living session log at `.ctx/local/CONTINUE.md` is
ignored through `.ctx/.gitignore`.

Use `--mode local` when the entire scaffold should remain private through the
target repository's `.git/info/exclude`. In either mode, the CLI never stages or
commits files.

## Install

```bash
npm install -g @neuvybe/ctx
ctx --version
```

The postinstall step fetches the matching OS/architecture `ctx` binary from the
[GitHub release](https://github.com/neuvybe/ctx/releases) for this package's
version and extracts it next to the package. If the fetch fails, it warns without
breaking `npm install`; fall back to:

```bash
go install github.com/neuvybe/ctx/cmd/ctx@latest
```

## Initialize

```bash
# default: shareable durable docs + ignored local session state
ctx init /path/to/repo

# opt in to a wholly private context folder
ctx init /path/to/repo --mode local

# custom folder name
ctx init /path/to/repo --folder .agent
```

For `ctx init`, `--folder` must be one top-level directory name containing only
letters, digits, `.`, `_`, or `-`. Nested paths such as `docs/ctx` and names
containing spaces are not supported.

Team mode writes the selected folder's `.gitignore` so its
`local/CONTINUE.md` remains local. All other scaffold files are available for
the user to review, stage, and commit. `ctx` itself does not run `git add` or
`git commit`.

After cloning committed team context, run `ctx init` again (repeating
`--folder`, if customized). It creates only the missing ignored continuation;
durable files are left unchanged. Custom-folder users must also repeat
`--folder <name>` with `update` and `doctor`.

Local mode adds the whole selected folder to `.git/info/exclude`; it never edits
the repository's root `.gitignore`. That exclude is shared by linked worktrees,
so `ctx` rejects local initialization when a sibling worktree has tracked
content or a team scaffold for the same folder.

## Commands

- `ctx init [target] [--folder .ctx] [--mode team|local]` — scaffold context;
  team mode is the default.
- `ctx update [target] [--folder .ctx]` — refresh managed blocks in `README.md` and `REVIEW.md`,
  preserve user content and sharing mode, and leave `local/CONTINUE.md` alone.
- `ctx doctor [target] [--folder .ctx]` — validate the scaffold, version stamp, ignore policy,
  placeholders, managed markers, and expected files.
- `ctx upgrade` — self-replace a direct binary from the latest GitHub release or
  print the right command for npm, Homebrew, or Go installs.
- `ctx --version` — print the installed version.

## Legacy scaffolds

Existing pre-team-mode scaffolds remain wholly local. `ctx update` does not
silently move their `<folder>/CONTINUE.md` to the new
`<folder>/local/CONTINUE.md` path or make their durable docs trackable; an
explicit future conversion flow will be required to opt them into team mode.

See the [ctx 0.2 migration guide](https://github.com/neuvybe/ctx/blob/main/docs/migrations/0.2-team-mode.md)
for the default change, compatibility behavior, and opt-out command.

See the [project README](https://github.com/neuvybe/ctx#readme) for the full
design and principles.
