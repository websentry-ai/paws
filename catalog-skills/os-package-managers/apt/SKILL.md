---
name: apt
description: 'Mandatory guardrails for every apt/apt-get/dpkg mutation (install, upgrade, dist-upgrade, remove, purge, autoremove, repository add) on Debian/Ubuntu dev machines and servers. Enforces the os-package-managers baseline with apt mechanics: update before install, -s simulation before bulk upgrades and removals, apt-mark holds, key-verified sources, and change windows on prod hosts. Triggers PROACTIVELY on any mutating apt command — do NOT wait for the user to invoke it.'
---

# apt / apt-get

apt-specific mechanics for the os-package-managers baseline (H1–H5). Apply automatically on every mutating `apt`, `apt-get`, `add-apt-repository`, or `dpkg -i` command — never skip. Simulations (`-s`) and reads (`apt list`, `apt-cache`, `apt-mark showhold`) run freely.

## Privilege model (implements H1)

apt mutations require sudo — that IS its model. Use `sudo apt-get …` as designed; never contort around it with `dpkg --force-*` flags or manual file drops into /var or /usr. Never bypass the manager with `curl <url> | sh` installers for software apt can install. **Never `--force-yes` or `--allow-unauthenticated`** — an unauthenticated package defeats the entire signing chain; fix the key problem instead (see H3).

## Preview before mutate (implements H2)

Always refresh the index first — a stale index installs old CVEs:

    sudo apt-get update

A single-package install of a well-known tool then flows freely — `sudo apt-get install jq` needs one line ("installing jq"), no simulation ritual. For unfamiliar packages, bulk upgrades, and removals, simulate and state the change set (including dependencies) first:

    apt-get install -s <pkg>

Bare upgrade moves everything — list first and state it:

    apt list --upgradable
    sudo apt-get upgrade

`dist-upgrade` can remove packages: `apt-get dist-upgrade -s` is mandatory, and any line marked for removal is called out explicitly before the real run.

## Sources are supply chain (implements H3)

PPAs and third-party repos only from {{SANCTIONED_SOURCES}}. Adding one takes a single line — the sources.list.d entry it creates plus the signing key's provenance (URL, keyring path) — then proceed:

    curl -fsSL <key-url> | sudo gpg --dearmor -o /usr/share/keyrings/<name>.gpg
    echo "deb [signed-by=/usr/share/keyrings/<name>.gpg] <repo-url> <suite> <component>" | sudo tee /etc/apt/sources.list.d/<name>.list

`add-apt-repository ppa:<owner>/<name>` states the Launchpad owner. A source not in {{SANCTIONED_SOURCES}} → stop and ask.

## Holds (implements H4)

Packages that services or workflows depend on are held per {{PIN_POLICY}}:

    sudo apt-mark hold <pkg>

Before any `upgrade` or `dist-upgrade`, check and state existing holds:

    apt-mark showhold

A held package the user asks to upgrade → unhold explicitly, state why, re-hold after. Collateral upgrades — dependencies pulled forward by another package — show in the `-s` output; relay them, never silent.

## Removal previews the orphans (implements H5)

    apt-get autoremove --dry-run
    apt-cache rdepends --installed <pkg>

Run both before removing: dry-run states the autoremove list, then the real command proceeds — no extra confirmation; rdepends shows what still depends on the package — dependents exist → stop and show them. `purge` deletes config files: simulate first (`apt-get purge -s <pkg>`), name the configs that go, and never purge silently — default to `remove` unless the user explicitly wants configs gone.

## Production hosts

On hosts matching {{PROD_HOSTS}}: mutate only inside a change window, and restart affected services per the org's process-system-admin skill — never as an unstated side effect.

## Escalation

Rule conflicts with user intent: state the risk in one line, then proceed on the user's say-so, noting the override is recorded. `--allow-unauthenticated` / `--force-yes` have no override — fix the keyring instead.

## Pre-flight checklist (before every mutating apt command)

- [ ] sudo as designed; no --force-yes / --allow-unauthenticated; no curl-pipe-sh bypass (H1)
- [ ] `apt-get update` run this session; known single-package install → one line and go; bulk upgrades/removals → `-s` stated (H2)
- [ ] New source in {{SANCTIONED_SOURCES}}, one-line entry + key provenance (H3); `apt-mark showhold` checked (H4)
- [ ] autoremove dry-run stated, then proceed; rdepends checked; purged configs named (H5); hosts matching {{PROD_HOSTS}} → change window

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
- `<sanctioned-sources>`: The distro's official repositories (homebrew/core and homebrew/cask for brew). Any other PPA, repo or key: stop and ask.
