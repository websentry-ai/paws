# when: conditions

## Sub-features

- `dependency:PKG`: package.json (all four dependency maps), go.mod requires,
  Cargo.toml dependency tables, pyproject.toml / requirements.txt names.
- `repo-marker:PATH`: PATH exists under the repo root (nearest `.git` above cwd).
- `new-file`: the Read/Write/Edit target does not exist yet.
- `no-edit-matching:GLOB`: no main-thread Edit/Write on a matching path in the transcript tail.
- `agent-count>N`: more than N Agent/Task spawns since the last prompt the user typed.
- One string or a list; every condition must hold.

## How to get to it (user POV)

Add `when:` beside `matcher:` in a rule.

## Driving it with drive.py

- `run when-dependency-hit`: package.json has stripe, the commit is gated, the skill's `PAY-OK:` prefix lands.
- `run when-dependency-miss`: same rule, no stripe, no audit record, plain commit.
- `run when-no-edit-tdd`: first Write under `src/` with no test edited is denied; after the skill, the test file (with `# TDD-OK`) and the source exist.
- `run when-repo-marker-hit`: `infra/terraform/` exists, the commit is gated, the skill's `TFM-3K:` prefix lands.
- `run when-repo-marker-miss`: `infra/` without `terraform/`, no audit record, plain commit.
- `run when-new-file`: an Edit of an existing file is never gated; the Write that creates `app/stats.py` is denied once, and after the skill the file opens with `# NB-58X module`.
- `run when-agent-count`: with `agent-count>1`, three sequential subagent spawns: the first two run (their files exist), the third is denied naming the skill once; the agent loads it and the third spawn runs, following the skill.

## Gotchas

- Repo conditions resolve from the hook's `cwd`, not the rule file's location.
- `no-edit-matching` and `agent-count` read the last 1 MB of the transcript.
- `agent-count>N` counts spawns already in the transcript, not the one being
  gated: with `>1` the third spawn is the first denied.
- Parallel tool calls from one assistant message do not see each other: the
  hooks run before the message reaches the transcript. A test and a source
  Write sent together both see "no test edited yet".
- `new-file` and `repo-marker` paths are stat'ed at call time; `paws explain` evaluates them from the shell's cwd.
