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

## Gotchas

- Repo conditions resolve from the hook's `cwd`, not the rule file's location.
- `no-edit-matching` and `agent-count` read the last 1 MB of the transcript.
- Parallel tool calls from one assistant message do not see each other: the
  hooks run before the message reaches the transcript. A test and a source
  Write sent together both see "no test edited yet".
- `new-file` and `repo-marker` paths are stat'ed at call time; `paws explain` evaluates them from the shell's cwd.
