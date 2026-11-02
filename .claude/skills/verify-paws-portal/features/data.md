# Data: counts, uploads, insights, suggestions

What paws did reaches the portal as counts, commands and tool actions (never prompts, replies or file
contents), redacted before upload. The user sees who loaded each skill, the last upload, an insights chart
over time, and suggestions for skills that would cover what Claude often runs.

## Sub-features

- `claude-read` a skill Claude loads before paws has to ask counts as `Claude read it` (per session).
- `redaction` secrets in commands are redacted on the laptop; file paths are relative to the repo.
- `last-upload` Settings → `Last upload` shows the stored records.
- `insights` `/portal/insights` draws the daily chart, recent loads and per-skill rows; Free sees 30 days.
- `suggestions` frequent uncovered actions (e.g. `docker build`) show a suggestion with `Dismiss`.

## How to get to it (user POV)

- Skills home headline and the selected-skill panel; `/portal/settings`; `/portal/insights`; the `Suggestions` panel on home.

## Driving it with portal.sh

Preconditions:

- `onboard` and `manage` passed.

- **Claude read it.** Run `portal.sh run data`. New session id, transcript with git loaded, `git commit` allowed, `Stop`. API `claude_read` +1; panel shows `Claude read it`.
- **Redaction.** `PostToolUse` a `curl -H "Authorization: Bearer sk-live-…" "…?api_key=…"` and an Edit of `src/app.ts`. `paws sync --dry-run` shows neither secret nor the absolute home path; after `Stop`, `GET /uploads/last` has the records with no secret, `src/app.ts` relative, no `content`/`old_string` keys; Settings `Last upload` shows the redacted curl.
- **History.** POST `/sync` with the laptop's device token and 42 rule records over 7 days. `GET /insights?days=30` has 7+ days; the page draws the chart (no `The chart shows after two days`); `days=90` is clamped on Free.
- **Suggestion.** 5 × `docker build` + `Stop`. `GET /suggestions` has `skill_name: docker`; home shows it; click `Dismiss`; it stays gone. Screenshots `20-settings`, `21-insights`, `22-suggestion`.

## Gotchas

- `already_loaded` is decided per Claude session on the laptop; reuse of one session id turns every later allow into "paws gave it".
- The insights chart needs runs on at least two days; seed history through `/sync`, not the DB.
