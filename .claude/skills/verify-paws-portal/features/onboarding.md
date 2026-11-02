# Onboarding and connect

A new user opens the portal, connects a laptop with a one-time key, sees skills suggested from what Claude
already ran on that laptop, turns some on, and the laptop gets them after Claude's next turn: paws then
stops a `git commit` once until Claude reads the skill.

## Sub-features

- `setup-redirect` a first visit to `/portal` lands on `/portal/setup`.
- `one-time-key` the setup page shows `npx unbound-paws init && paws login --key pk_…`; the key works once.
- `connect-status` the page moves from `Waiting for paws to connect…` to `Connected.` to `Found 1 skill.`.
- `picks-from-counts` `Picked from what Claude ran` cards with reasons like `Claude ran git commit 12 times`; skills already on the laptop show under `On from your laptop`.
- `turn-on` `Turn on N skills` saves them; `N skills saved` and `See my skills`.
- `waiting-banner` home says `Waiting for sync` until the laptop reports the new config, then stops saying it.
- `laptop-pull` the next `Stop` hook writes rule files under `PAWS_HOME/portal/` and installs full skills into `CLAUDE_CONFIG_DIR/skills`.
- `first-gate` `git commit` is denied naming the skill; allowed once the transcript shows the Skill tool loaded it.
- `home-counts` the headline and the skill rows count the run.

## How to get to it (user POV)

- Open `/portal` signed in with no skills (redirects to `/portal/setup`).
- Run the command from the setup page in a terminal.
- `Next`, pick cards, `Turn on N skills`, `See my skills`.

## Driving it with portal.sh

Preconditions:

- `portal.sh up` and `doctor` ok. The script deletes the account (`DELETE /account`) and recreates the scratch laptop with git history and a laptop skill `team-notes`.

- **History.** Seed action records with `PostToolUse` events (12 × `git commit`, 4 × `gh pr create`, a `curl` with a bearer token, an Edit under `db/migrations/`). Run `portal.sh run onboard`. `PAWS_HOME/audit.log` exists.
- **Key.** Read `paws login --key pk_…` from the setup page; run it. Stdout has `Logged in as verify@…`; a second login with the same key fails.
- **First sync.** `paws sync`. The page shows `Connected.` and `Found 1 skill.`
- **Picks.** Click link `Next`. Heading `Pick your house rules`, section `Picked from what Claude ran`, chip `team-notes`.
- **Turn on.** Click button `Turn on 2 skills`. `2 skills saved` and `Waiting for sync`.
- **Pull.** Feed a `Stop` event. `PAWS_HOME/portal/*.md` exist; `CLAUDE_CONFIG_DIR/skills/git` and `/humanizer` exist.
- **Gate.** `PreToolUse` `git commit -m "real change"` → `permissionDecision: deny` naming `git`; with a transcript where `git` and `humanizer` were loaded → no decision (allowed).
- **Counts.** `Stop` again; `GET /skills` shows `git` `state: ok`, `paws_gave ≥ 1`; reload home: no `Waiting for sync`. Screenshots `01-setup-connect` … `06-home-synced`.

## Gotchas

- `onboard` deletes the `mock-verify` account: run it first, never against an account you care about.
- The `humanizer` skill also runs before a commit message (`pre-commit`), so an allowed commit needs both skills in the transcript.
- `?saved=` stays in the URL after `Turn on`; the banner must still disappear once the laptop syncs (a regression fixed in the website's portal PR).
- Counts are per Claude session: an allow after a deny in the same session is "paws gave it", not "Claude read it".
