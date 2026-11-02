# Skill updates

A daily backend job compares each catalog skill's pinned commit with its upstream on GitHub. When the skill
changed, the portal shows the diff; nothing changes until the user clicks `Update <skill>`. Then laptops
install the new commit from the same repo and folder, only if the download hashes to the approved content;
if it doesn't, the installed version keeps serving and the laptop doesn't report the config as applied.

## Sub-features

- `check` `manage.py check_skill_updates` records an update with a diff.
- `banner` home shows `Review updates`.
- `review` `/portal/updates` shows the diff (scrolling inside its box) and `Update <skill>` / `Keep current version`.
- `approve` the skill's pin moves to the new commit and hash.
- `laptop-install` `paws sync` installs `github.com/<repo>@<new commit>` (see the skill's `.paws-skill.json`).
- `mismatch` a wrong hash: sync exits non-zero with `keeping the installed version`, the applied hash stays, the portal shows `waiting_for_sync`.

## How to get to it (user POV)

- The `Review updates` banner on home, the `Updates` nav tab, or `Review update` in the selected-skill panel.

## Driving it with portal.sh

Preconditions:

- `onboard` passed (humanizer on, laptop logged in). Network access to api.github.com.

- **Check.** Run `portal.sh run updates`. The script moves humanizer's pin back to an older commit in the DB and runs the job. Output has `'created': 1`; `GET /updates` has a diff.
- **Review.** Home link `Review updates`; `/portal/updates` button `Update humanizer` is visible; click it. Text `humanizer updates to …`; API version = the update's commit.
- **Install.** `paws sync`. `CLAUDE_CONFIG_DIR/skills/humanizer/.paws-skill.json` names `github.com/blader/humanizer@<new commit>` and the approved hash; portal not `waiting_for_sync`.
- **Mismatch.** Set the skill's `content_hash` to a wrong value in the DB, `paws sync`. Non-zero exit with `keeping the installed version`; marker unchanged; `state.json` applied hash unchanged; API `waiting_for_sync`. Restore the hash; sync is clean. Screenshot `updates-diff`.

## Gotchas

- Uses live GitHub without a token (60 requests an hour). Set `PAWS_GITHUB_TOKEN` for the backend if you hit the limit.
- If humanizer's upstream head ever equals the older commit's skill files, the job creates nothing; pick another older commit.
- Moving the pin in the DB is the only setter used; everything after it goes through the job, the UI and the CLI.
