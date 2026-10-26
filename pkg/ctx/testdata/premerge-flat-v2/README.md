# Pre-merge flat-layout regression fixtures

These are byte-for-byte scaffold outputs from ctx 0.4.0, built from commit
`57c719617a570a0f66cc560d380d1b480ab51b74`, the base of PR #4. They were
captured on 2026-10-25, before exercising them with the current implementation.
They are historical inputs, not golden files to refresh when templates change.

The binary was built from `git archive` of that commit, outside the working
tree. Each case started in a fresh Git repository named `premerge-project`:

```sh
# team-glossary-only: non-default selection in the default folder
ctx-premerge init /temporary/team/premerge-project --without behavior

# local-core-only: no add-ons in a custom folder
ctx-premerge init /temporary/local/premerge-project \
  --mode local --folder team_context-2.0 --without behavior,glossary
```

Each `scaffold/` tree contains the complete generated scaffold, including local
continuation state. `git-info-exclude.fixture` contains the corresponding
repository's generated `.git/info/exclude`, which is needed to reproduce local
mode without invoking today's initializer or ignore-rule generator.

Every captured filename has an additional `.fixture` suffix. The loader removes
only that suffix and copies the contents unchanged into a temporary repository.
In particular, `.gitignore.fixture` must not act as an ignore file here: otherwise
Git would silently omit the captured local continuation from this repository.

The permanent tests cover flat-path and saved-selection preservation, authored
facts and nested topics, add-on installation, doctor/status, visibility, and
fresh-clone team hydration. Authored content is added only to temporary test
copies; these captured files remain untouched. Tests need neither the historical
binary nor Git history, network access, or a model provider.
