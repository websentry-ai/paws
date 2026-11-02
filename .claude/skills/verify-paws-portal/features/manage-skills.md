# Manage skills

The user manages skills only in the browser, and every laptop follows after Claude's next turn: adding a
catalog skill (as the full SKILL.md or as a short rule), changing the actions it runs before, turning it
off and back on, and removing it.

## Sub-features

- `add-full` `Add skill` → search → `Add <name>` → `Turn on <name>`.
- `add-rule` the `Rule · about 40 tokens` option with editable `Rule text`; the laptop gets a digest, no SKILL.md.
- `edit-actions` `Edit actions` → `Add a command, file pattern or MCP tool` → `Save`; the laptop gates the new command.
- `recent-runs` the selected-skill panel lists recent commands with `paws gave it` / `Claude read it`.
- `turn-off-undo` `Turn off` → `<name> is off.` banner with `Undo`; the laptop drops and restores the rule files.
- `remove` `Remove` → `Remove <name>` on the edit page; the laptop deletes the rule files.

## How to get to it (user POV)

- Skills home `/portal`: button `Add skill`, a row link to select a skill, the panel's `Edit actions` / `Turn off`.
- `/portal/skills/new?skill=<name>` and `/portal/skills/<id>`.

## Driving it with portal.sh

Preconditions:

- `onboard` passed (git and humanizer on, laptop logged in).

- **Add as rule.** Run `portal.sh run manage`. Fill searchbox `Search skills, rules, tools` with `secret-scan`, click link `Add secret-scan`, click the `about 40 tokens` option, fill `Rule text` with a gitleaks line, click `Turn on secret-scan`. Banner `secret-scan is saved.`; API `load: rule`, `state: waiting_for_sync`.
- **Laptop.** `Stop`. `PAWS_HOME/portal/secret-scan--*.md` exists; no `CLAUDE_CONFIG_DIR/skills/secret-scan`; `git commit` (git and humanizer loaded) is denied with the gitleaks text.
- **Edit.** On `/portal/skills/<git id>` add `npm publish`, click `Save`. After `Stop`, `npm publish` is denied naming `git` and allowed once `git` is loaded.
- **Recent runs.** `/portal?selected=<git id>` shows `Recent runs` with the command and `paws gave it`.
- **Off and undo.** Click `Turn off` on humanizer → `humanizer is off.`; after `Stop` no `humanizer--*.md` and `gh pr create` is allowed; click `Undo`, `Stop`, the files are back and `gh pr create` is denied.
- **Remove.** On secret-scan's edit page click `Remove`, then `Remove secret-scan`. API no longer lists it; after `Stop` its rule file is gone. Screenshots `10-add-rule` … `14-off`.

## Gotchas

- A rule-mode skill must not install a SKILL.md; check the skills dir, not only the rule files.
- `Undo` only exists while the `?off=` banner is shown; reloading another URL loses it.
- `Remove` asks for confirmation in place (a second button), not a browser dialog.
