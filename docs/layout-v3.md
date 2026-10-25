# Folder-first layout v3

New scaffolds put each fact category in a directory with an `index.md`. This
provides room to grow from the first initialization, without generating topic
stubs or forcing a separate file for every definition. Sharing modes, selected
default add-ons, owner authority, and local continuation are unchanged.

```text
context/
├── overview/
│   └── index.md
├── architecture/
│   └── index.md
├── caveats/
│   └── index.md
├── behavior/           # default-selected add-on
│   └── index.md
└── glossary/           # default-selected add-on
    └── index.md
```

Only indexes are generated. Selecting contracts or extending likewise creates
`context/contracts/index.md` or `context/extending/index.md`. Operating policy
remains `OPERATING.md`, review remains `workflows/review.md`, and clone-local
state remains `local/CONTINUE.md`. `ctx add --list` displays current default
paths; installation into an existing scaffold follows that scaffold's layout.

## Populate for selective retrieval

Keep indexes useful for orientation and choosing a route. During the first
population pass, put independently useful explanations in sibling topic files:
for example, `architecture/evidence-checking.md` or `glossary/hydration.md`.
Use descriptive filenames, useful aliases, and links saying when a topic is
relevant. These are examples, not generated filenames or required topics.

Keep short, closely related definitions together. A two-sentence definition
does not need its own file. A concept needing examples, substantial explanation,
or different supporting evidence may deserve one. Maintain one canonical
explanation and link it, rather than copying full detail into indexes.

INDEX routes to a relevant category; its index provides shared orientation and
routes to needed topics. Direct topic routes and targeted search can avoid
repeated navigation once scope is understood. Do not load all children by
default. Fewer irrelevant bytes is a design aim, not a guarantee of faster or
more accurate retrieval: extra navigation also costs tool calls.

The first [retrieval comparison](../evals/ctx-skill/retrieval-comparison-results.md)
did not establish a speed or accuracy advantage. Treat selective loading as a
design aim, not a measured performance promise.

## Evidence and size guidance

Every fact index and topic has its own `ctx:doc` metadata. A verified index
certifies only its own claims and sources, never all linked children. Do not use
another context document as source evidence. Draft children remain independently
not ready even when the parent is ready. Doctor checks structure and status
checks metadata/freshness; neither proves prose accuracy.

Writing targets are about 250 words for root INDEX, 300 for continuation, 500
for overview, 400 for other category indexes, and 800 for topic files. Status
warns non-fatally around twice these values. Organize by coherent topic during
population, not solely in response to a word-count warning.

## Existing scaffolds

Layout v3 uses config schema 2, layout version 3, template revision `3.0.0`.
The config schema and named managed-marker grammar have not changed. The saved
layout selects required paths, add-on outputs, routing, and update templates.

- Layout v2 retains flat `context/<category>.md` parents, authored children,
  selected add-ons, and its `2.0.1` template revision. Init hydration, add,
  update, doctor, and status continue to work without moving files.
- Layout v1 and config-less legacy scaffolds retain frozen templates and their
  original continuation and sharing behavior. Status still cannot certify v1
  readiness because that layout predates fact metadata.
- Update refreshes managed guidance for the installed layout. It does not move
  facts, install new defaults, or convert layout/mode. No conversion command is
  included: do not change layoutVersion by hand or move established facts just
  to match these examples.

The pinned [populated ctx example](examples/ctx/README.md) deliberately remains
a historical v2 snapshot, not a claim that current init generates flat parents.

## Go integration

`InitWithOptions` creates v3 and hydrates supported existing team scaffolds in
place. The deprecated `Init` and `TemplateFS` retain their legacy-v1 behavior.
`TemplateFSForLayout` accepts versions 1, 2, and 3. `AddonTemplateFS` retains its
original flat-v2 API; use `AddonTemplateFSForLayout(layoutVersion, id)` to select
explicitly. `ListAddons` and `LookupAddon` describe the current default layout.
