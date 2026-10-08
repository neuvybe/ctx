# ctx CLI contract

Use the installed CLI's help and persisted scaffold configuration if they
disagree with this reference. These examples describe layout v2 in ctx 0.4.0
and later compatible releases; CLI release, config schema, layout version,
and managed template revision are separate values.

## Select the target explicitly

Commands accept a target repository and `--folder`; otherwise they use the
current directory and `.ctx`. Repeat a custom folder on every relevant command.

```bash
ctx --version
ctx init /path/to/repo --folder .agent
ctx doctor /path/to/repo --folder .agent
ctx status /path/to/repo --folder .agent
```

New initialization requires a Git repository and a single top-level folder name
containing letters, digits, dots, underscores, or hyphens; not `.` or `..`.
Maintenance tolerates safe existing nested paths for compatibility. That does
not authorize creating new nested scaffolds.

## Creation, sharing, and hydration

```bash
ctx init /path/to/repo
ctx init /path/to/repo --mode local
ctx init /path/to/repo --without behavior,glossary
ctx init /path/to/repo --with contracts
```

- Team mode is the default. Durable files are visible to normal Git tracking;
  `local/CONTINUE.md` stays ignored through the scaffold's `.gitignore`.
- Local mode excludes the whole folder through the repository's common
  `.git/info/exclude`. Ignored is not encrypted or access-controlled.
- Behavior and glossary are default-selected add-ons. `--with` and `--without`
  are repeatable and comma-friendly; do not override explicit selections.
- A valid existing team scaffold with missing ignored continuation can be
  hydrated by init. Shared files and configured add-ons remain unchanged.
  Existing continuation, incomplete scaffolds, and mode conflicts are refused.
- Init does not convert existing local/legacy scaffolds, reset their contents,
  or stage/commit files. Report a refusal; do not delete context or change ignore
  rules to force success.

Inspect `config.json` for `mode`, `layoutVersion`, `templateRevision`,
`project`, and `addons`. Do not hand-edit these to install capabilities or
convert modes.

## Extend deliberately

```bash
ctx add --list
ctx add /path/to/repo contracts --folder .agent
```

Add installs a missing selected document set, refreshes INDEX's managed routing
and config, and refuses existing outputs. The new facts are draft and need
authoring. An existing scaffold does not adopt newly introduced defaults
through update or hydration. An older template revision needs update before
add; a newer one needs a compatible CLI, not a forced downgrade.

## Refresh managed guidance

```bash
ctx update /path/to/repo --folder .agent
```

Update uses the persisted layout/add-ons. Layout-v2 README and INDEX have strict
named managed blocks; do not remove/rename those markers. Put project routes
outside the managed routing block. Updates preserve project-owned sections,
facts, and authored children and advance managed template metadata.

Update does not inspect the application to rewrite facts, reverify metadata,
install new defaults, or convert a mode/layout. CLI installation/upgrade is a
separate operation and needs its own authorization.

## Check structure and readiness separately

```bash
ctx doctor /path/to/repo --folder .agent
ctx status /path/to/repo --folder .agent
git -C /path/to/repo status --short
```

Doctor validates required files, configuration, template compatibility,
managed-marker grammar, and effective sharing/privacy boundaries. Missing
owner-pointer placeholders are not an invitation to invent an owner policy.

Status checks required facts and every nested Markdown document under
`context/`. Draft/invalid metadata, missing evidence, and changed listed
sources cause non-zero readiness. Word-count guidance alone does not.
Inspect reported paths, reconcile within scope, and state unresolved gaps.
Neither checker reviews the truth or sufficiency of the prose.

## Legacy scaffolds

Schema-v1 and config-less legacy scaffolds use their existing file set and
frozen templates. Config-less legacy scaffolds are whole-folder local and
have root `CONTINUE.md`. Their compatibility path is not layout-v2 metadata:
status reports that it cannot certify their readiness.

Use compatible doctor/update operations only when requested. Do not move
legacy files, invent a v2 config, install v2 add-ons into them, or convert
sharing mode as a side effect of ordinary work.
