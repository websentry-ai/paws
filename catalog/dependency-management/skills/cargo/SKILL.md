---
name: cargo
description: 'Mandatory guardrails for every Rust dependency operation (cargo add, cargo install, cargo update). Enforces the dependency-management baseline with cargo mechanics: Cargo.lock committed and honored via --locked, a one-line advisories check before adopting, build.rs read for unfamiliar crates, and one-line notes on cargo install globals. Triggers PROACTIVELY on any cargo add/install/update — do NOT wait for the user to invoke it.'
---

# cargo

cargo-specific mechanics for the dependency-management baseline (L1–L5). Apply automatically on every Rust dependency operation — never skip.

## Lockfile discipline (implements L1)

Cargo.lock is committed for binaries (and for libraries per current cargo guidance). Builds and installs in existing projects honor it:

    cargo build --locked
    cargo install --locked <crate>

`cargo update` rewrites the lockfile wholesale — run it only when updating is the task, scoped when possible (`cargo update -p <crate>`), diff reviewed and committed deliberately. Lock conflicts: merge Cargo.toml first, then `cargo update -p` the conflicted crates to re-resolve — **never delete Cargo.lock**.

## Audit before adopt (implements L2)

Scales with familiarity. Well-known crate (high downloads, a name you recognize — serde, tokio, clap): quick advisories check, one line ("serde: no advisories — adding"), proceed. Unfamiliar crate:

    cargo audit
    cargo info <crate>

State the finding in one line before adding: RustSec advisories from `cargo audit` (or an osv.dev query with `"ecosystem":"crates.io"` if cargo-audit is unavailable), last-release recency and downloads from `cargo info`, license against {{LICENSE_ALLOWLIST}}.

## Deliberate adds (implements L1, L4)

    cargo add <crate>

Then review the Cargo.toml diff before building: intended crate name (typosquats differ by one letter), version requirement per {{PINNING_POLICY}}, no surprise feature flags. The Cargo.lock diff commits together with the Cargo.toml change.

## build.rs is code execution (implements L3)

`build.rs` and proc-macros run arbitrary code at **build** time — cargo has no `--ignore-scripts`, so the inspection happens between add and build. Well-known crates build normally. Per {{INSTALL_SCRIPTS_POLICY}}, first contact with an **unfamiliar** crate:

    cargo fetch
    find ~/.cargo/registry/src -maxdepth 2 -path "*<crate>-<version>*" -name build.rs

Read any `build.rs` (network calls, file writes outside OUT_DIR, env exfiltration are red flags) before the first `cargo build` pulls the trigger.

## Registry and provenance (implements L4)

Crates resolve from {{SANCTIONED_REGISTRY}} only. Before the first add of a session, check for source replacement:

    cat .cargo/config.toml ~/.cargo/config.toml 2>/dev/null

A `[source]` or `[registries]` redirect that isn't the sanctioned registry → stop and ask. No `git = "http://…"` or git dependencies from unsanctioned hosts in Cargo.toml.

## Scope discipline (implements L5)

- **Never `sudo cargo`.** A permission error means a root-owned `~/.cargo` or target dir — fix ownership, don't escalate.
- `cargo install <crate>` is a **global** install into `~/.cargo/bin`: one-line note (crate, why global) per {{GLOBAL_INSTALL_POLICY}}, then proceed. Prefer project-local paths — dev-dependencies, `cargo run -p`, or a pinned `cargo install --locked --root ./target/tools`.

## Escalation

Rule conflicts with user intent ("skip the audit"): state the risk in one line, then proceed on the user's say-so, noting the override is recorded. `sudo cargo` has no override — fix ownership instead.

## Pre-flight checklist (before every cargo dependency operation)

- [ ] Existing project → `--locked`; `cargo update` scoped and deliberate; lock diff reviewed
- [ ] New crate: one-line advisories check; unfamiliar → `cargo info` workup, build.rs read before first build
- [ ] Cargo.toml diff reviewed after `cargo add` (exact name, version range, features)
- [ ] Source replacement checked; no unsanctioned git deps; no sudo; `cargo install` gets a one-line note

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
