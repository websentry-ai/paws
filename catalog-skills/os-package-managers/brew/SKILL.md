---
name: brew
description: 'Mandatory guardrails for every Homebrew mutation (install, upgrade, uninstall, reinstall, tap, pin, autoremove, cleanup, link) on macOS and Linux dev machines. Enforces the os-package-managers baseline with brew mechanics: no sudo ever, known single-formula installs flow freely, outdated before bulk upgrades, sanctioned taps only, pin checks, and dry-run removal. Triggers PROACTIVELY on any mutating brew command — do NOT wait for the user to invoke it.'
---

# Homebrew

Homebrew-specific mechanics for the os-package-managers baseline (H1–H5). Apply automatically on every mutating `brew` command — never skip. Reads (`brew info`, `brew list`, `brew outdated`, `brew deps`, `brew doctor`) run freely.

## Privilege model (implements H1)

**Never `sudo brew` — for any subcommand.** Homebrew refuses root by design; a root-owned prefix breaks all later use. If a formula's post-install step needs elevation, brew requests it itself — that is the only sanctioned elevation. Never bypass brew with `curl <url> | sh` installers for software brew can install; the manager's file ownership and audit trail are the point. Permission errors in the prefix are fixed by ownership repair, not sudo:

    brew doctor

## Preview before mutate (implements H2)

A single-formula install of a well-known tool flows freely — `brew install jq` needs one line ("installing jq"), no deps-tree ritual. For an unfamiliar formula, preview and state what lands:

    brew info <f>
    brew deps --tree <f>

Bare `brew upgrade` moves everything — run `brew outdated` first and state the list:

    brew outdated
    # then upgrade explicitly, or confirm the full list before a bare upgrade
    brew upgrade <f1> <f2>

## Taps and casks (implements H3)

`brew tap <user/repo>` adds a third-party source — only from {{SANCTIONED_SOURCES}}; one line stating the GitHub org it resolves to, then proceed. Casks install unsigned or vendor-signed apps outside the App Store: they are supply chain. Before `brew install --cask <c>`:

    brew info --cask <c>

State the source URL in one line. A cask whose download URL is not the vendor's own domain → stop and ask.

## Pins (implements H4)

Tools that services or workflows depend on are pinned per {{PIN_POLICY}}. Before any upgrade:

    brew list --pinned

A pinned formula in the upgrade set → leave it pinned and state why, or get explicit confirmation to `brew unpin` first (and re-pin after). Collateral upgrades — dependencies pulled forward by another formula's upgrade — are stated, never silent: `brew upgrade` output names them; relay it.

## Removal previews the orphans (implements H5)

Before `brew uninstall <f>`, check what still depends on it:

    brew uses --installed <f>

Dependents exist → stop and show them. Before autoremove or cleanup, dry-run and state the list:

    brew autoremove --dry-run
    brew cleanup --dry-run

Then run the real command — the stated dry-run is the gate, no extra confirmation. Never `brew uninstall --zap` (removes configs, caches, app data) without stating the paths it deletes — the cask's zap stanza in `brew info --cask <c>` lists them.

## Escalation

Rule conflicts with user intent: state the risk in one line, then proceed on the user's say-so, noting the override is recorded. `sudo brew` has no override: it does not work and corrupts the prefix.

## Pre-flight checklist (before every mutating brew command)

- [ ] No sudo; no curl-pipe-sh bypass of brew (H1)
- [ ] Known single-formula install → one line and go; unfamiliar → `brew info` + `brew deps --tree`; `brew outdated` stated before bulk upgrades (H2)
- [ ] Tap/cask source in {{SANCTIONED_SOURCES}}, one-line org/URL statement (H3); `brew list --pinned` checked (H4)
- [ ] `brew uses --installed` for uninstalls; `--dry-run` stated before autoremove/cleanup, then proceed; zap paths named (H5)

## Category baseline: OS Package Managers

These rules apply to **every** package-manager mutation. An instance skill (brew, apt, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. A package manager mutates the machine itself: one wrong elevation corrupts the install for every later call, one silent upgrade breaks every workflow that depended on the old version, one unsanctioned repo is a standing supply-chain hole. On hosts matching {{PROD_HOSTS}}, every mutation additionally respects the org's change windows.

### Rule H1: Respect the privilege model

Every manager has an intended elevation model — follow it in both directions:

- Never elevate where the tool forbids it: `sudo brew` corrupts the prefix; rootful installs of user-scoped managers break the same way.
- Never skip required elevation: apt, dnf, and pacman need sudo for mutations — that IS their model; do not contort around it with force flags or manual file drops.
- Never bypass the manager: no piping installers to shells (`curl <url> | sh`) for software the manager can install. The manager's ownership of the file set and its audit trail are the point.

### Rule H2: Know what changes before it changes

Preview scales with blast radius:

1. Single-package install of a well-known tool: state the one-liner ("installing jq") and proceed — no preview ritual.
2. Bulk upgrades and removals: use the tool's preview mechanics (simulate / dry-run / `outdated` / `list --upgradable`) and state the change set — including dependencies — before executing.
3. Bare `upgrade` (no package named) moves everything: list what moves first and state it before running.

### Rule H3: Third-party sources are supply chain

Taps, PPAs, repos, casks, and community buckets execute code with the machine's trust:

- Add sources only from {{SANCTIONED_SOURCES}}; anything else → stop and ask.
- Adding a source takes one line — origin (org/owner, URL) and signing-key provenance — then proceed.
- Never install unauthenticated or signature-bypassed packages to work around a key problem — fix the key.

### Rule H4: Pin what other things depend on

Tools that services, CI, or workflows depend on are pinned/held per {{PIN_POLICY}}:

1. Check existing pins/holds before any upgrade; a pinned package stays pinned unless the user explicitly confirms otherwise.
2. Collateral upgrades — dependencies pulled forward by another package's upgrade — are stated, never silent.
3. No pin policy configured → skip-and-state: say which upgraded packages look load-bearing.

### Rule H5: Removal previews the orphans

Uninstall, autoremove, and cleanup change more than the named package:

1. Show dependents (what still uses the package) and configs affected before running.
2. Autoremove/cleanup runs its dry-run mode first; state the list, then run the real command — the dry-run IS the gate, no extra confirmation.
3. Never purge or zap config files silently — name what gets deleted; default to keeping configs.

### Pre-flight checklist (run before every package-manager mutation)

- [ ] Elevation matches the tool's model; no forced or manager-bypassing installs (H1)
- [ ] Bulk upgrades/removals previewed and stated; single known-tool installs just narrated (H2)
- [ ] New sources in {{SANCTIONED_SOURCES}}, one-line origin + key provenance (H3); pins/holds per {{PIN_POLICY}} (H4)
- [ ] Dependents checked; dry-run before autoremove/cleanup, then proceed (H5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<pin-policy>`: Pin or hold what services and workflows depend on. With no policy configured, skip and state: say which upgraded packages look load-bearing.
- `<sanctioned-sources>`: homebrew/core and homebrew/cask (distro official repos for apt). Any other tap, or a cask whose download is not on the vendor's own domain: stop and ask.
