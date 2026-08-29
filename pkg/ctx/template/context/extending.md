# Extending — {{PROJECT}}

> **Prompts — answer against the actual code, then delete this block.**
> - What are the extension seams (interfaces a new thing implements)?
> - For each kind of extension (language / provider / plugin / query / tool /
>   analyzer / … — pick what applies), document: the interface, the reference
>   implementation, the wiring points (where to register/dispatch), the test
>   convention, and any gotchas.
> - Is there an owner-authored authoritative guide for any of these (e.g. a
>   `.claude/skills/...SKILL.md`)? If so, **defer to it** and slim this section
>   to a pointer — don't duplicate.
> - What project-specific conventions must new code respect (custom analyzers,
>   constructor patterns, naming)?

## Extension seams

> Fill in (the interfaces).

## Adding a `<kind>` (per extension kind)

> Fill in (one subsection per kind that applies; interface + reference impl +
> wiring points + tests + gotchas).

## Conventions new code must respect

> Fill in (analyzers/linters, constructor patterns, naming, hashing rules,
> …). Defer to the owner's canonical instructions where they cover this.