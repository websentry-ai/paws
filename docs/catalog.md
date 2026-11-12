# Catalog and packs

The catalog is every binding paws offers. A pack is a named set of bindings and
is what a user installs (`unbound-paws add writing`).

The published catalog lives in the paws backend's database, not in this repo or
the binary. The website pages through it, `unbound-paws search` and `add` ask for
just what they need, and every skill is downloaded from the backend by content
hash.

## Layout

A catalog on disk (what `--catalog <dir>` reads):

```
catalog/<category>/<binding>.md              one binding
catalog/<category>/digests/<name>.md         digest a binding's action.digest points at
catalog/<category>/skills/<skill>/SKILL.md   a skill we own (org tag)
packs/<pack>.yaml                            a pack manifest
trigger_cases.json                           calls each binding must fire on and ignore
without_trigger_cases.txt                    older bindings without cases yet
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
  content_hash: sha256:<64 hex> # required; `unbound-paws pin .` writes it
provenance:
  stars: 61292
  license: MIT
  last_push: 2026-10-16
  fetched: 2026-10-27
  official: true # optional; see below
summary: 'Rewrites text that sounds machine-written so it reads like a person wrote it.' # first binding only
tags: [practice, writing]
---
What the binding does, in a sentence or two.
```

`summary` says what the skill is about, not who acts on it, in one plain sentence of at
most 140 characters: "Your team's git rules: commit message format, protected branches,
and a secret scan before every commit." The portal shows it on the add-skill page. A skill
bound by several files takes it from the first one by path (`writing/humanizer-commit.md`
before `writing/humanizer-pr.md`), so put it there. The website shows the skill's own
SKILL.md `description` instead; the backend reads it at push.

`provenance.official: true` marks a skill whose upstream repo belongs to the company that
makes the product the skill is about: Cloudflare's skills for Cloudflare, Stripe's for
Stripe. A third party writing about someone else's product is not official, and when in
doubt, leave it out. Decide per skill, not per repo. The website shows an Official chip.
Only an upstream skill can be official, and like `summary` the first binding by path is
the one read, so set it on every binding of the skill.

Rules for a valid binding (`unbound-paws validate` checks each one):

- It parses as a paws rule, and every frontmatter key is a known one. The
  runner itself ignores unknown keys; the validator catches typos.
- A `require-skill` skill name is lowercase letters, digits and dashes.
- `upstream.path` is relative and stays inside the upstream repo.
- Its `name` is unique across the catalog.
- It has at least one tag.
- `upstream.repo`, `path` and `ref` are set together or not at all.
- An upstream binding has `upstream.content_hash`: `sha256:` and 64 hex digits.
- A `require-skill` binding has an `upstream`, or a local skill at
  `catalog/<category>/skills/<skill>/SKILL.md`.
- `action.digest` exists relative to the binding.

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

## Install

`unbound-paws add <pack>` asks the backend for that pack's bindings, digests and
skills (`--catalog <dir>` reads one on disk instead), validates them like a
catalog on disk, and:

1. installs each required skill: upstream ones downloaded from the backend and
   refused unless the files hash to `upstream.content_hash`; ours from
   `catalog/<category>/skills/<skill>/` with `{{NAME}}` placeholders filled
   from that directory's `config.yaml` (`placeholders: {NAME: default}`);
2. copies the bindings (with their digests) to
   `<packs dir>/<pack>/<catalog-relative path>` (a single binding,
   `unbound-paws add writing/humanizer-pr`, goes under `<packs dir>/_bindings/`);
3. records the pack, its picks and the skills it installed in `installed.yaml`.

User mode: `~/.paws/packs`, `~/.claude/skills`, `~/.paws/installed.yaml`.
`--project`: `.paws/packs`, `.claude/skills`, `.paws/installed.yaml` in the repo.

A skill directory paws installed carries `.paws-skill.json` with the hash of
what it wrote. paws replaces or removes a skill directory only when that
marker is there and the content still matches it: your own skills and your
edits to installed ones are left alone.

`upstream.repo` is a host path (`github.com/owner/repo`, fetched over https)
and `upstream.ref` a full commit SHA; `unbound-paws validate` rejects anything else,
and `unbound-paws add` validates whatever catalog it reads before installing.

The content hash is sha256 over each file's path and bytes, in path order,
skipping any path segment that starts with `.`. `unbound-paws pin <dir>` fetches every
upstream skill and writes the hash into its bindings; `unbound-paws validate`
rejects an upstream binding without one.

## Check it

```
unbound-paws validate <dir>
```

With a `trigger_cases.json` in `<dir>`, `validate` also runs every trigger against its cases.

A trigger that fires when its skill doesn't apply blocks Claude until it loads a skill it doesn't need, so each
binding's trigger is tested against real calls. `trigger_cases.json` lists, per binding name, calls the trigger must
fire on and near misses it must ignore, as `[tool, target]` pairs; the target is what the hook matches (the command,
file path, URL, subagent prompt, MCP input as JSON, or session source). Add a few of each with a new binding: a key
word inside another word, the tool named in `echo` or a commit message, `--help`, a read-only subcommand when the
skill is about writes. The first fire case of each binding is also the example the website shows under its trigger.
