# cargo digest

`cargo add`, `cargo install`, `cargo update`. Apply before retrying.

## Lockfile discipline
- Cargo.lock is committed and honored: `cargo build --locked`, `cargo install --locked <crate>`.
- `cargo update` rewrites the lockfile: only when updating is the task, scoped (`cargo update -p <crate>`), diff reviewed and committed deliberately.
- Lock conflict: merge Cargo.toml, then `cargo update -p` the conflicted crates. Never delete Cargo.lock.

## Before adopting a new crate
- Well-known (serde, tokio, clap): quick advisories check, one line ("serde: no advisories — adding").
- Unfamiliar: `cargo audit` (or an osv.dev query with `"ecosystem":"crates.io"`) and `cargo info <crate>`. One line: RustSec advisories, last-release recency and downloads, license (permissive MIT/Apache-2.0/BSD/ISC; copyleft or missing → flag).

## Deliberate adds
After `cargo add <crate>`, review the Cargo.toml diff before building: exact crate name (typosquats differ by one letter), caret version requirement, no surprise feature flags. Commit the Cargo.lock diff with the Cargo.toml change.

## build.rs is code execution
cargo has no `--ignore-scripts`; build.rs and proc-macros run at build time. Unfamiliar crate, first contact:
`cargo fetch`, then `find ~/.cargo/registry/src -maxdepth 2 -path "*<crate>-<version>*" -name build.rs`; read it (network calls, writes outside OUT_DIR, env exfiltration are red flags) before the first `cargo build`. Well-known crates build normally.

## Registry
Crates come from crates.io only. Before the first add of a session: `cat .cargo/config.toml ~/.cargo/config.toml 2>/dev/null`. A `[source]`/`[registries]` redirect that isn't sanctioned → stop and ask. No `git = "http://…"` or git deps from unsanctioned hosts.

## Scope
- Never `sudo cargo`. A permission error = root-owned `~/.cargo` or target dir; fix ownership.
- `cargo install <crate>` is a global install into ~/.cargo/bin: one-line note (crate, why global), then proceed. Prefer dev-dependencies, `cargo run -p`, or `cargo install --locked --root ./target/tools`.

## Baseline (all package managers)
- Lockfile is law: clean-install mode in existing projects; lockfile changes only with the manifest change that caused them, diff reviewed (new transitive packages, changed resolved URLs); conflicts re-resolved from the merged manifest, never by deleting the lockfile.
- Audit scales with familiarity: well-known → one-line advisories check; unfamiliar → also maintenance signal (years untouched is a risk) and license against MIT/Apache-2.0/BSD/ISC; copyleft or missing license → flag before proceeding.
- Install/build scripts are code execution: scripts off on first contact with an unfamiliar package, inspect, then enable. Well-known packages install normally.
- No git+http(s), tarball or direct-URL dependencies from unsanctioned hosts.
- Never sudo the package manager: a permission error means the wrong environment. Project-local by default; a global install gets a one-line note (developer CLIs only).
- User says skip a rule: state the risk in one line, proceed on their say-so, note the override is recorded. Sudo has no override.

## Checklist
- [ ] Existing project → `--locked`; `cargo update` scoped and deliberate
- [ ] New crate: advisories line; unfamiliar → `cargo info` workup, build.rs read before first build
- [ ] Cargo.toml diff reviewed after `cargo add`
- [ ] Source replacement checked; no unsanctioned git deps; no sudo; `cargo install` noted
