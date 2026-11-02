# paws portal verification map

The maintained source for verifying what a paws user does through the portal: connect a laptop, manage
skills in the browser, see what paws did, approve skill updates, and manage the account. Read this index,
then use the matching feature file as the recipe. Every recipe has a script under `../scripts/`.

## Baseline preconditions

- `scripts/portal.sh up` started this instance (backend :8013 test mode, website :3135, fresh DB) and
  `scripts/portal.sh doctor` is all `ok`. Never drive a backend or website this run did not start.
- The browser is signed in as `mock-verify` through `NEXT_PUBLIC_PAWS_DEV_TOKEN=verify`; API calls use
  `Authorization: Bearer verify`.
- The laptop is the scratch `HOME` under `$VERIFY_DIR/home` with `PAWS_HOME` and `CLAUDE_CONFIG_DIR`
  inside it, the paws binary built from this checkout, and the `open` recorder first on PATH.

## Driving conventions

- Run features through `scripts/portal.sh run <feature>`; `onboard` first (it resets the account and
  laptop), then the others in the suite order: `manage`, `data`, `updates`, `account`, `billing-jobs`.
- Browser actions use ARIA roles and visible names; never coordinates.
- Laptop actions are the real CLI (`paws login`, `paws sync`) and `paws gate` fed Claude Code's hook JSON.
  `PAWS_SYNC_INTERVAL=0s` and `PAWS_CONFIG_INTERVAL=0s` make every `Stop` hook upload and pull.
- `claude-gate` is the only feature that runs real Claude Code.

## Proof and skip reporting

- Pair each user action with the UI state and the stored state: API with the dev token, `psql` on the
  run's DB, files under the scratch laptop.
- Every step and check is appended to `results.txt`; screenshots go to `shots/`.
- Report a feature you could not drive with the precondition that failed; never report it as verified
  through another path.

## Feature entry contract

Each feature file starts with an H1 and one paragraph, then four H2s in order: `Sub-features`,
`How to get to it (user POV)`, `Driving it with portal.sh`, `Gotchas`.

## Features

- [Onboarding and connect](./onboarding.md): setup page, one-time key, first sync, picks from what Claude ran, first gated commit.
- [Manage skills](./manage-skills.md): add from the catalog (full or short rule), edit actions, turn off and undo, remove; the laptop follows each change.
- [Data: counts, uploads, insights, suggestions](./data.md): who loaded the skill, redacted uploads, the insights chart, suggestions and dismiss.
- [Skill updates](./updates.md): the daily check, the diff, Update, the laptop installing the approved commit, a mismatched hash.
- [Account and laptops](./account.md): weekly email, key from Settings, the browser connect card, delete data, disconnect all.
- [Billing, plans, jobs, catalog](./billing-jobs.md): Billing page without Stripe, Pro limits, purge and weekly-email jobs, public catalog, auth refusals.
- [Gate with real Claude Code](./gate-real-claude.md): skills turned on in the portal stop a real `claude -p` commit until Claude reads them.
