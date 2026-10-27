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
`digests/`, are not bindings.

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
  on_miss: load-required        # or digest (needs action.digest)
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

- It parses as a paws rule.
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

A pack lists at least one binding. A binding appears once per pack. A
`pick_one` group has at least two options, each with at least one binding,
and its `default` is one of them.

## Check it

```
paws validate .
```
