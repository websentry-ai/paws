# Catalog and packs

The catalog is every binding paws ships. A pack is a named set of bindings and
is what a user installs (`paws add writing`).

## Layout

```
catalog/<category>/<binding>.md              one binding
catalog/<category>/digests/<name>.md         digest a binding's action.digest points at
catalog/<category>/skills/<skill>/SKILL.md   a skill we own (org tag)
packs/<pack>.yaml                            a pack manifest
```

Files named `README.md` or `SKILL.md`, and anything under `skills/` or
`digests/`, are not bindings. The runner skips them in `.paws/` too, so a
category copied into `.paws/` loads cleanly.

## Binding

A binding is a paws rule file with extra frontmatter. The runner reads
`name`, `event`, `matcher` and `action`. The installer and the catalog page
read the rest. Unknown keys are ignored.

```markdown
---
name: humanizer-pr
event: pre-tool
matcher:
  tool: Bash
  command_regex: (^|[;&|\n])\s*gh\s+pr\s+(create|edit)\b
action:
  type: require-skill
  skill: humanizer
  on_miss: load-required        # or digest (needs action.digest); see docs/rules.md
upstream:                       # third-party skill, installed at this ref
  repo: github.com/blader/humanizer
  path: SKILL.md
  ref: 8c307f367c91bd974002f10f2be2965d642c13df
  content_hash: sha256:<64 hex> # optional until the installer pins it
provenance:
  stars: 61292
  license: MIT
  last_push: 2026-10-16
  fetched: 2026-10-27
tags: [practice, writing]
---
What the binding does, in a sentence or two.
```

Rules for a valid binding (`paws validate` checks each one):

- It parses as a paws rule, and every frontmatter key is a known one. The
  runner itself ignores unknown keys; the validator catches typos.
- A `require-skill` skill name is lowercase letters, digits and dashes.
- `upstream.path` is relative and stays inside the upstream repo.
- Its `name` is unique across the catalog.
- It has at least one tag.
- `upstream.repo`, `path` and `ref` are set together or not at all.
- `upstream.content_hash`, when set, is `sha256:` and 64 hex digits.
- A `require-skill` binding has an `upstream`, or a local skill at
  `catalog/<category>/skills/<skill>/SKILL.md`.
- `action.digest` and an `exec-script` script exist relative to the binding.

## Pack manifest

```yaml
name: writing                   # must equal the file name
description: One line for the catalog page.
tags: [writing]
bindings:                       # paths relative to catalog/
  - writing/humanizer-pr.md
pick_one:                       # optional: install exactly one option
  - name: prose
    description: Which de-slop skill to use.
    default: humanizer
    options:
      humanizer: [writing/humanizer-pr.md]
      stop-slop: [writing/stop-slop-pr.md]
```

A pack lists at least one binding. A `pick_one` group has a unique name, at
least two options, each with at least one binding, and its `default` is one
of them. A binding appears once in what installs together: the top-level
`bindings` plus one option. Two options of one group may share a binding.

## Check it

```
paws validate .
```
