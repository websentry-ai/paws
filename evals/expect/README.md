# Expect eval

Checks whether a model, following only `skills/paws/expect.md`, writes expect checks that tell "Claude followed
the skill" from "Claude didn't". Run it after a change to the guide or to the model that writes checks, and
compare with the baseline below.

## Run it

```
python3 evals/expect/run.py --out /tmp/expect-eval
```

- Needs Go, git, Claude Code (`claude`) logged in to claude.ai, and network access to GitHub.
- Takes about 30 minutes at 5 sessions in parallel, and counts against the claude.ai usage limit. When it hits
  the limit, it waits and retries. A stopped run resumes when you pass the same `--out`.
- Options:
  - `--model haiku` tests another model;
  - `--skills git,coreutils` runs a subset;
  - `--grade-only` grades the checks already in `--out`.

## What it does

1. Builds `paws` and the grader from this checkout, and copies this checkout's guide (`skills/paws/SKILL.md`,
   `skills/paws/expect.md`, `docs/rules.md`).
2. Gets each skill's text from the commit pinned in `data/skills.json` and checks its sha256. Placeholders
   stay unfilled, because the labelled cases were written from the unfilled text.
3. Runs one headless `claude -p` session per skill (`prompt.md`). Each runs in a fresh temp folder with only the
   guide, the skill text, its rule files from `data/rules/` and a `paws` binary.
   - The session gets an empty `HOME` and a scratch `CLAUDE_CONFIG_DIR`, so no skills, hooks, memory or history
     from your machine reach it.
   - It logs in with the claude.ai login (`CLAUDE_CODE_OAUTH_TOKEN`, or the keychain entry Claude Code writes on
     macOS), not an API key.
4. Grades the checks with `casegrade`, which runs paws's own `outcome.Grade` on each labelled case in
   `data/cases.jsonl`.

## The test set

`data/cases.jsonl` has 1,609 synthetic cases over 86 rule files of 56 skills.

- A separate model session wrote them from each skill's text only, without seeing any checks. A second pass
  dropped cases that broke another rule of the same skill.
- Each case is a short turn: the blocked call, then what Claude did next. The label is `follows` or `breaks`.
- A `breaks` case is graded right when a check fails; a `follows` case when no check fails.
- `data/split.json` splits the skills into 28 `dev` skills (use these when tuning the guide) and 28 `held` skills
  (report these).
- `data/probes.jsonl` holds 10 hand-written cases for two known misses: a ticket after a 72-character commit
  summary, and listing before `rm` with a glob or variable.

All names, emails, hosts and keys in the cases are made up, and none come from production data. Key-like
values use documented examples such as `AKIAIOSFODNN7EXAMPLE`.

## Baseline

Opus 5.5, guide at f9fd3c1 (`docs/expect-guide-coverage`), Nov 25, 2026. All 56 skills, 3 sessions in parallel.

| Set | Right | Breaks caught | Compliant passed |
|---|---|---|---|
| dev | 78% (526/672) | 62% (203/328) | 94% (323/344) |
| held | 78% (727/937) | 62% (267/434) | 91% (460/503) |
| all | 78% (1253/1609) | 62% (470/762) | 92% (783/847) |

- The run wrote 283 checks. 81 of 86 rule files got at least one check, and none had load errors.
- Probes: 10/10.

Run-to-run noise with the same setup is 2 to 3 points.

## Files

| File | What it is |
|---|---|
| `run.py` | the runner |
| `prompt.md` | the prompt each session gets |
| `casegrade/` | the grader (`go run ./evals/expect/casegrade -h`) |
| `baseline.json` | the scores above, which `run.py` prints next to yours |
| `data/rules/` | each skill's rule files, without checks |
| `data/skills.json` | where each skill's text comes from, and its sha256 |
