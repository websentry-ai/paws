---
name: pip
description: 'Mandatory guardrails for every Python dependency operation (pip install, pip download, uv add, uv pip install). Enforces the dependency-management baseline with Python mechanics: venv-first, hash-checked requirements installs, a one-line osv check before adopting, sdist inspection for unfamiliar packages, and no system or sudo pip. Triggers PROACTIVELY on any pip or uv install — do NOT wait for the user to invoke it.'
---

# pip

pip/uv-specific mechanics for the dependency-management baseline (L1–L5). Apply automatically on every Python dependency operation — never skip.

## Venv first, never system pip (implements L5)

Before any install, confirm an active virtualenv (`$VIRTUAL_ENV` set, or `which pip` resolving inside the project). If none:

    python3 -m venv .venv
    source .venv/bin/activate

- **Never `sudo pip`; never install into the system interpreter.** A permission error means no venv is active — create one, don't escalate.
- Global tools (pipx-style): one-line note (package, why global) per {{GLOBAL_INSTALL_POLICY}}, then proceed.

## Lockfile discipline (implements L1)

Installs in existing projects consume the pinned artifact unchanged:

    # requirements with hashes (pip-compile --generate-hashes output)
    pip install --require-hashes -r requirements.txt

    # uv-managed project
    uv sync --frozen

Where hashes exist, `--require-hashes` is mandatory — it turns a registry-side package swap into a hard failure. Pinned files (requirements.txt, uv.lock, poetry.lock) change only through their compile step (`pip-compile`, `uv lock`) with the diff reviewed and committed alongside the manifest change — never hand-edited, never regenerated as a side effect. Lock conflicts: merge the source manifest (requirements.in / pyproject.toml), re-compile — **never delete the lock**.

## Audit before adopt (implements L2)

Scales with familiarity. Well-known package (high downloads, a name you recognize — requests, numpy, flask): quick osv check, one line ("requests: no advisories — installing"), proceed. Unfamiliar package:

    pip index versions <pkg>
    curl -s https://api.osv.dev/v1/query -d '{"package":{"name":"<pkg>","ecosystem":"PyPI"}}'

State the finding in one line before installing: osv advisories, release recency from `pip index versions` (stale >18 months is a flag), and license (`info.license` in `https://pypi.org/pypi/<pkg>/json`) against {{LICENSE_ALLOWLIST}}.

## sdists run code (implements L3)

Wheels install without executing code; sdists run `setup.py`/build backends at install time — arbitrary code execution. Well-known packages install normally. Per {{INSTALL_SCRIPTS_POLICY}}, first contact with an **unfamiliar** package prefers wheels only:

    pip install <pkg> --only-binary :all:

If only an sdist exists, download and inspect before installing:

    pip download <pkg> --no-deps --no-binary :all: -d ./.sdist-inspect

Read `setup.py` and any build hooks it references, then install normally and remove the inspect directory.

## Registry and pinning (implements L4)

Before the first install of a session:

    pip config list

`global.index-url` / `install.index-url` (unset = pypi.org) must match {{SANCTIONED_REGISTRY}}; a surprise mirror → stop and ask. Same check for uv (`UV_INDEX_URL`, `[tool.uv.index]`). No `git+http`/direct-URL requirements from unsanctioned hosts. Manifest ranges follow {{PINNING_POLICY}}; the compiled lock carries exact pins and hashes.

## Escalation

Rule conflicts with user intent: state the risk in one line, then proceed on the user's say-so, noting the override is recorded. `sudo pip` has no override — it mutates the interpreter every tool on the machine shares.

## Pre-flight checklist (before every pip/uv operation)

- [ ] Virtualenv active; no sudo; globals get a one-line note
- [ ] Existing project → `--require-hashes` / `uv sync --frozen`; lock changed only via compile step, diff reviewed
- [ ] New package: one-line osv check; unfamiliar → recency + license workup, wheels only or sdist inspected
- [ ] `pip config list` index is sanctioned; no git/direct-URL deps; ranges per pinning policy

## Category baseline: Dependency Management

These rules apply to **every** dependency-installing tool. An instance skill (npm-deps, pip, cargo, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. This is the USAGE-shaped side of the supply chain — what the agent pulls **in**; what the org ships **out** is governed by the `package-publishing` category. Every install is code execution on an org machine: a regenerated lockfile hides a dependency swap, an unvetted postinstall script is an attacker's shell, a typosquatted package is one transposed letter away.

### Rule L1: Lockfile is law

The lockfile is the reviewed, committed truth of what runs:

1. Installs in existing projects use the clean-install mode (`npm ci`, `--frozen-lockfile`, `pip-sync`) that **errors on drift** instead of silently mutating the lockfile.
2. Lockfile changes are deliberate: made only alongside the manifest change that caused them, with the diff reviewed (new transitive packages, changed resolved URLs) before committing — never regenerated as a side effect of an unrelated task.
3. Lockfile merge conflicts are resolved by re-resolving from the merged manifest — **never** by deleting the lockfile and regenerating from scratch.

### Rule L2: Audit before adopt

Adding a NEW dependency (one not already in the manifest or lockfile) scales with familiarity — state the finding in **one line** and proceed:

1. **Well-known package** (high downloads, a name you recognize): a quick advisories check — the ecosystem's audit tool or an osv.dev query — is enough. "lodash: no advisories — installing."
2. **Unfamiliar package**: add the maintenance signal (last release date, download volume — years untouched is a risk statement, not a stability statement) and license against {{LICENSE_ALLOWLIST}}; copyleft or missing license → flag before proceeding.

### Rule L3: Install scripts are code execution

Lifecycle and build scripts (postinstall, setup.py, build.rs) run arbitrary code as the developer's user at install or build time. First contact with an **unfamiliar** package is a supply-chain event: follow {{INSTALL_SCRIPTS_POLICY}} — install with scripts disabled (`--ignore-scripts` or the ecosystem equivalent), inspect what the scripts do, then enable them explicitly. Well-known packages install normally — no first-contact ritual.

### Rule L4: Pin and provenance

- Dependencies resolve from {{SANCTIONED_REGISTRY}} only; verify the configured registry before the first install of a session — a surprise mirror is an incident, not a config quirk.
- No `git+http`/`git+https` or tarball/direct-URL dependencies from unsanctioned hosts.
- Version ranges in the manifest follow {{PINNING_POLICY}}; the lockfile carries the exact resolution.

### Rule L5: Scope discipline

- **Never sudo the package manager.** A permission error means the wrong environment (missing venv, root-owned prefix), not missing privileges — fix the environment.
- Project-local over global by default. A global install takes a one-line note (package, why global) per {{GLOBAL_INSTALL_POLICY}} — state it and proceed, no further ceremony.

### Pre-flight checklist (run before every dependency operation)

- [ ] Existing project → clean-install mode; lockfile changes deliberate, diff reviewed (L1)
- [ ] New dependency → one-line advisories check; unfamiliar → maintenance + license too, scripts disabled on first contact (L2, L3)
- [ ] Registry sanctioned; no unsanctioned URLs; ranges per pinning policy (L4)
- [ ] No sudo; project-local default; globals get a one-line note (L5)
