---
name: npm-deps
description: 'Mandatory guardrails for every npm dependency operation (npm install, npm ci, npm update, npm rebuild, npx execution). Enforces the dependency-management baseline with npm mechanics: npm ci in existing projects, a one-line audit before adopting, --ignore-scripts on first contact with unfamiliar packages, and no sudo. Triggers PROACTIVELY on any npm install/update or npx run — do NOT wait for the user to invoke it.'
---

# npm (dependency usage)

npm-specific mechanics for the dependency-management baseline (L1–L5). Apply automatically on every npm dependency operation — never skip. This skill covers the **install side** only; publishing (`npm publish`, versioning, dist-tags) is governed by the separate `package-publishing/npm` skill.

## npm ci vs npm install (implements L1)

Decision rule, checked before every install:

    # Existing project (package-lock.json present), no dependency change intended
    npm ci

    # Deliberately adding or changing a dependency
    npm install <pkg>    # then review the package-lock.json diff before committing

Bare `npm install` in an existing project is almost always wrong — it can rewrite the lockfile as a side effect; use `npm ci`. Lockfile changes are committed together with the package.json change that caused them, diff reviewed for new transitive packages and changed `resolved` URLs. Merge conflicts in package-lock.json: resolve package.json first, then `npm install` to re-resolve — **never delete the lockfile**.

## Audit before adopt (implements L2)

Scales with familiarity. Well-known package (high downloads, a name you recognize — react, lodash, express): quick advisories check, one line ("lodash: no advisories — installing"), proceed. Unfamiliar package:

    npm audit --omit=dev
    npm view <pkg> time.modified license

State the finding in one line before installing: advisories, last-modified date (stale >18 months is a flag), license against {{LICENSE_ALLOWLIST}} — copyleft or missing → flag first.

## Install scripts and npx (implements L3)

Lifecycle scripts (preinstall/install/postinstall) are arbitrary code execution. Well-known packages install normally. Per {{INSTALL_SCRIPTS_POLICY}}, first contact with an **unfamiliar** package:

    npm install <pkg> --ignore-scripts

Inspect the `scripts` block of `node_modules/<pkg>/package.json` and any files it invokes, then enable deliberately:

    npm rebuild <pkg>

`npx <pkg>` downloads and executes registry code in one step. Well-known runners (create-next-app, prettier, tsc via npx) flow freely. An unfamiliar npx target gets the one-line L2 check and a pinned version (`npx <pkg>@<version>`), never a floating tag.

## Registry and pinning (implements L4)

Before the first install of a session:

    npm config get registry

Must resolve to {{SANCTIONED_REGISTRY}}; anything else → stop and ask. Never add `git+http`/`git+https` dependencies from unsanctioned hosts or raw tarball URLs. Manifest ranges follow {{PINNING_POLICY}}; the lockfile carries the exact pin.

## Scope discipline (implements L5)

- **Never `sudo npm`.** EACCES means a root-owned prefix or wrong Node install — fix the environment (nvm/volta prefix), don't escalate.
- `npm install -g`: one-line note (package, why global) per {{GLOBAL_INSTALL_POLICY}}, then proceed. Default to project-local devDependencies plus `npx` from the local tree.

## Escalation

Rule conflicts with user intent ("skip the audit"): state the risk in one line, then proceed on the user's say-so, noting the override is recorded. `sudo npm` has no override — it breaks the install.

## Pre-flight checklist (before every npm operation)

- [ ] Existing project → `npm ci`; dependency change → lockfile diff reviewed, committed deliberately
- [ ] New package: one-line advisories check; unfamiliar → `npm view` workup, `--ignore-scripts` first, then `npm rebuild`
- [ ] `npm config get registry` = sanctioned registry; no git/tarball URLs; ranges per pinning policy
- [ ] No sudo; `-g` gets a one-line note; unfamiliar npx targets checked and version-pinned

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
