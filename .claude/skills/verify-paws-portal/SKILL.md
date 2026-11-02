---
name: verify-paws-portal
description: Verify the paws portal end to end — the paws CLI on a scratch laptop, paws-backend, and the paws-website portal in a real browser, plus real Claude Code gated by skills turned on in the portal. Use after changes to `paws login`/`sync`/portal config in this repo, to paws-backend, or to the website's portal pages, and before a release that touches the portal.
---

# Verify the paws portal

The portal spans three checkouts: this repo (the CLI), `paws-backend` (Django API) and `paws-website`
(Next.js portal). `scripts/portal.sh` starts a private copy of the backend and website, builds paws from
this checkout into a scratch laptop (`HOME`, `PAWS_HOME`, `CLAUDE_CONFIG_DIR` all scratch), and drives
features with a headless Chrome (playwright-core) and the paws CLI. `verify-paws` covers the hook on its
own; this skill covers everything that goes through the portal.

## Launch

Prerequisites: `go`, `node`/`npm`, `psql` and a Postgres you can create databases on, Google Chrome, a
`paws-backend` checkout with its Python deps installed, and a `paws-website` checkout after `npm ci`.

```bash
S=.claude/skills/verify-paws-portal/scripts
export PAWS_BACKEND_DIR=../paws-backend PAWS_WEBSITE_DIR=../paws-website      # defaults: siblings of this repo
export VERIFY_PG_ADMIN_URL=postgres://postgres:postgres@localhost:5432/postgres
export VERIFY_PYTHON=../paws-backend/.venv/bin/python                          # default: that path, else python3
$S/portal.sh up          # fresh DB "paws_verify", backend :8013 (test mode), website :3135 (next dev)
```

Ready when `up` ends with the doctor table all `ok`. Other knobs: `VERIFY_API_PORT`, `VERIFY_SITE_PORT`,
`VERIFY_DB`, `VERIFY_CHROME`, `VERIFY_STATE` (instance dir; one instance per value), `VERIFY_PW_DIR`
(playwright-core cache, kept across runs).

Test mode means `ENVIRONMENT=test DEBUG=true` on the backend (any bearer token `X` is user `mock-X`) and
`NEXT_PUBLIC_PAWS_DEV_TOKEN=verify` on the website (signed in as `mock-verify`, a "Dev token" chip in the
nav). `paws login` opens the browser through `open`; `up` puts a recorder named `open` first on the
scratch laptop's PATH, so no real browser tab ever opens.

Teardown: `$S/portal.sh down`.

## Doctor

```bash
$S/portal.sh doctor
```

Read-only. Checks the recorded backend and website processes are alive and are the listeners on their
ports (not someone else's server), `/healthz` reports the database, the website serves `/portal`, the
paws binary runs, the database answers, playwright-core is installed, and whether the Claude token for
`claude-gate` exists. Run it first whenever a feature fails oddly.

## Drive

```bash
$S/portal.sh run                     # onboard → manage → data → updates → account → billing-jobs
$S/portal.sh run onboard manage      # a prefix of the suite (later features build on earlier state)
$S/portal.sh run claude-gate         # real Claude Code (independent; costs a few model calls)
$S/portal.sh run all                 # the suite, then claude-gate
```

`run` stops at the first failing feature. `onboard` resets the portal account and the scratch laptop, so
any run starts with it; `claude-gate` and `billing-jobs` need nothing before them except `onboard` for
`billing-jobs`. Each feature is one script under `scripts/` built on `scripts/lib.mjs`:

- the laptop side is the real paws binary: `paws login --key`, `paws sync`, and `paws gate` fed the exact
  hook JSON Claude Code sends (`PreToolUse`, `PostToolUse`, `Stop`);
- the browser side uses ARIA roles and visible names (`getByRole("button", { name: "Turn on 2 skills" })`);
- assertions read both the UI and the stored state (API with the dev token, `psql` on the run's DB, files
  under the scratch laptop).

What each feature drives, step by step, is in [features/](features/README.md).

### Real Claude Code (`claude-gate`)

Starts its own portal user (`verify-claude`) and scratch repo, runs `paws init` there (project scope),
turns on skills through the API, logs the scratch laptop in with a one-time key, then runs

```
claude -p "Commit the staged change…" --output-format stream-json --verbose \
  --allowedTools "Bash(git:*)" Skill --disallowedTools "Edit(~/.paws/**)" "Edit(~/.claude/**)"
```

and asserts on behaviour: paws denied the first `git commit`, the deny carried a random token from the
portal rule text, the commit subject starts with that token (only possible if Claude read the rule), and
the portal counted the run.

- **Isolated** (preferred): with a token at `~/.config/paws-verify/claude-token` (or
  `VERIFY_CLAUDE_TOKEN_FILE`), Claude runs with `CLAUDE_CONFIG_DIR=<scratch>` and
  `CLAUDE_CODE_OAUTH_TOKEN`, so the real `~/.claude` is never read or written. It also turns on `git` as a
  full skill and asserts Claude loaded it with the Skill tool and the retry passed on `skill-load`.
  Create the token once, in your own terminal (never in a chat):
  `claude setup-token`, copy the token, then
  `mkdir -p ~/.config/paws-verify && pbpaste > ~/.config/paws-verify/claude-token && chmod 600 ~/.config/paws-verify/claude-token`.
- **Fallback** (no token): the developer's normal login (macOS keeps it in the Keychain for the default
  config dir, so a scratch `CLAUDE_CONFIG_DIR` is logged out). Only the rule skill is turned on, so paws
  installs nothing into the real `~/.claude`; `--setting-sources project` keeps the user's own hooks out.
  That flag also hides user skills (where paws installs full skills), which is why the isolated mode, with no
  user settings to hide, runs without it.
  Claude Code itself still writes its session transcript under `~/.claude/projects/` in this mode.
- `VERIFY_SKIP_CLAUDE=1 portal.sh run claude-gate` checks everything up to the `claude` call (init, portal
  skills, login, sync) without spending model calls.

### Real mode (real PropelAuth sign-in)

The scripted features sign in with the dev token, so they run in test mode only. To prove the real
sign-in, run the stack in real mode and drive the user's own Chrome with playwriter (it is already
signed in to PropelAuth):

```bash
PAWS_PROPELAUTH_ENV=~/unbound/paws-propelauth.env VERIFY_SITE_PORT=3124 $S/portal.sh up --real
```

- The PropelAuth test project only accepts `http://localhost:3124` as the frontend, so real mode must use
  site port 3124 (stop whatever else serves it first; never someone else's process without asking).
- Never print the env file's values.
- Recipe (playwriter, one tab of the user's Chrome; `playwriter session new` first):
  1. `playwriter -s 1 -e 'await page.goto("http://localhost:3124/signin")'`, click `Continue with email`,
     complete PropelAuth. Proof: the URL is `/portal/setup` and the page says `Connect Claude Code`.
  2. Read the command shown on the setup page (`paws login --key pk_…`), run it with the scratch laptop's
     env (`. $VERIFY_STATE/env.sh`; `HOME=$VERIFY_DIR/home PAWS_HOME=$VERIFY_DIR/home/.paws
     CLAUDE_CONFIG_DIR=$VERIFY_DIR/home/.claude PAWS_PORTAL_API=$VERIFY_API $PAWS_BIN login --key …`).
     Proof: the CLI prints `Logged in as <your email>.`
  3. Device flow instead: run `$PAWS_BIN login` with the same env but the real `open`, click `Connect` on
     the `Connect this laptop?` card. Proof: the card and the CLI show the same code, the CLI logs in.
  4. Pick skills (`Turn on N skills`), `paws sync`, then run the `claude-gate` claude command by hand in a
     scratch repo with that laptop's env. Proof: as in `claude-gate`.

## Evidence

Everything lands in `<repo>/.verify-artifacts/portal-<run-id>/` (git-ignored) and survives `down`:

| Path | Holds |
|---|---|
| `results.txt` | every step and PASS/FAIL line, all features of the run |
| `shots/*.png` | full-page screenshots at each proof point (setup, pick, home, selected skill, updates, settings, insights, billing, catalog, connect card) |
| `backend.log`, `site.log`, `migrate.log` | the servers' output |
| `claude-gate/` | `stream.jsonl` (raw `claude -p`), `audit.jsonl` (what paws decided), `settings.json` (what `paws init` wrote), `git-log.txt` |

Proof standards: drive the user path (buttons, the CLI), not internal setters; check the side effect next
to the screen (rule files under `PAWS_HOME/portal`, installed skill markers, DB rows, the redacted upload);
the only stubs are the browser opener and, for Pro gating, a direct plan update in the DB, because Stripe
checkout needs real Stripe keys. `updates` talks to live GitHub (unauthenticated, 60 requests an hour).

## Cleanup

```bash
$S/portal.sh down
```

Kills only the recorded backend and website process trees, drops the run's database, deletes the
instance dir (scratch laptop included). Prints the evidence path and warns if it is missing. Run it after
every failed attempt too. The playwright-core cache in `VERIFY_PW_DIR` stays for the next run.

## Helpers

| Script | Use |
|---|---|
| `scripts/portal.sh up [--real] / doctor / run [features] / down` | the whole lifecycle; the only entry point |
| `scripts/run.mjs` | runs features in order, stops at the first failure (called by `portal.sh run`) |
| `scripts/<feature>.mjs` | one feature each: `onboard`, `manage`, `data`, `updates`, `account`, `billing-jobs`, `claude-gate` |
| `scripts/lib.mjs` | laptop (`paws`, `gate`, `stop`, `transcriptWith`), API (`api`, `skillRow`, `psql`, `manage`), browser (`browser`, `shot`, `text`), `check`/`step`/`done` |

The feature scripts read their config from the env `portal.sh up` writes to `$VERIFY_STATE/env.sh`; run
them through `portal.sh run`, not with `node` directly.
