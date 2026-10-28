---
name: secret-scan
description: 'Check staged changes for secrets before every git commit, and keep .env and secrets/ files out of git. Use before any commit and before writing or editing a .env file or anything under secrets/.'
---

# Secret scan

A secret that reaches a commit is burned, pushed or not. Check before it lands.

## Before every commit

1. Run the repo's scanner on the staged changes when one exists (gitleaks, trufflehog, detect-secrets, a pre-commit hook).
2. Always also check the staged file names and added lines:

       git diff --cached --name-only | grep -E '(^|/)\.env($|\.)|(^|/)secrets/|\.(pem|key|p12|pfx)$|(^|/)id_(rsa|ed25519)' | grep -v '\.example$'
       git diff --cached -U0 | grep -E '^\+' | grep -nE 'AKIA[0-9A-Z]{16}|BEGIN [A-Z ]*PRIVATE KEY|gh[pousr]_[A-Za-z0-9]{36}|github_pat_|xox[abprs]-|sk-[A-Za-z0-9_-]{20,}|AIza[0-9A-Za-z_-]{35}|(password|passwd|secret|token|api_?key)[^:=]{0,3}[:=][[:space:]]*[^[:space:]]{8,}'

3. Say what you checked in one line, then commit.

On a hit:
- Unstage it: `git restore --staged <file>`. Do not commit it "for now".
- Move the value to an environment variable or the secret store, and read it from there.
- Add the file to `.gitignore`, and commit a `.env.example` with placeholder values if the project needs one.
- Tell the user what you found and where. If the value was ever committed or pushed, say it must be rotated; deleting it from history is not enough.
- A test fixture or a documented fake key is fine. Say why it is fake.

## Writing .env or secrets/ files

- Never write a real credential the user did not give you. Use placeholders such as `API_KEY=changeme`.
- Before writing, check the path is ignored: `git check-ignore -q <path>`. If it is not, add it to `.gitignore` first.
- Never print a secret value back into the chat, a log or a commit message.
