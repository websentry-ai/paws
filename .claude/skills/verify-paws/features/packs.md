# Packs and skill install

## Sub-features

- `paws add [--project] [--pick group=option] <pack|category/binding>`: bindings from the
  catalog embedded in the binary, plus the skills they require.
- Upstream skills: fetched at the pinned ref, refused on a content-hash mismatch.
- Our own skills: copied from `catalog/<category>/skills/`, `{{PLACEHOLDER}}`s filled from `config.yaml`.
- A skill paws did not install is never replaced or removed (`.paws-skill.json` marks paws's own).
- `paws remove` drops a pack and the skills only it needed; `paws list` shows fire counts;
  `paws search`, `paws index`; `paws pin` (maintainers) pins upstream hashes.

## How to get to it (user POV)

`paws add writing` after `paws init`. `--project` installs into the repo
(`.paws/packs/`, `.claude/skills/`) so a team can commit it.

## Driving it with drive.py

`run launch-journey` (also with `--npm`): init, `paws add --project writing`
fetches humanizer from GitHub, then the agent's commit is denied once,
humanizer is invoked, and the commit lands. Proof: the installed skill has
paws's marker, the audit shows one deny then an allow via skill-load.

## Gotchas

- Cases always use `--project`: user mode writes to the developer's real `~/.claude/skills`.
- The case needs network access to GitHub for the upstream skill.
