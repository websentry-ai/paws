# brew digest

Mutating `brew` command (install, upgrade, uninstall, reinstall, tap, pin, autoremove, cleanup, link). Reads (`info`, `list`, `outdated`, `deps`, `doctor`) run freely. Apply before retrying.

## Privilege
- Never `sudo brew`, for any subcommand; no override. A root-owned prefix breaks all later use.
- Prefix permission errors: `brew doctor` and ownership repair, not sudo.
- No `curl <url> | sh` for software brew can install.

## Preview before mutate
- Known single formula: one line ("installing jq"), then install.
- Unfamiliar formula: `brew info <f>` and `brew deps --tree <f>`; state what lands.
- Bare `brew upgrade`: run `brew outdated`, state the list, then upgrade named formulae (or confirm the full list).

## Taps and casks
- `brew tap <user/repo>` only from `<sanctioned-sources>` (default: homebrew/core and homebrew/cask; anything else: stop and ask). One line naming the GitHub org it resolves to.
- Before `brew install --cask <c>`: `brew info --cask <c>`, state the source URL. Download not on the vendor's own domain: stop and ask.

## Pins
- Pin what services depend on per `<pin-policy>`; no policy: say which upgraded formulae look load-bearing.
- Before any upgrade: `brew list --pinned`. A pinned formula stays pinned (say why) unless the user confirms `brew unpin`; re-pin after.
- Relay collateral upgrades from `brew upgrade` output.

## Removal
- Before `brew uninstall <f>`: `brew uses --installed <f>`; dependents exist: stop and show them.
- `brew autoremove --dry-run` / `brew cleanup --dry-run`: state the list, then run the real command.
- Never `brew uninstall --zap` without naming the paths it deletes (zap stanza in `brew info --cask <c>`).

## Escalation
Rule vs user intent: state the risk in one line, proceed on the user's say-so, note the override.

## Baseline (all package managers)
- Follow the tool's elevation model; never bypass the manager.
- Preview scales with blast radius; bare upgrades list first.
- Third-party sources are supply chain: state origin and key provenance; never bypass signatures.
- Pinned stays pinned unless confirmed; collateral upgrades stated.
- Removal shows dependents; configs kept by default.
- On hosts matching `*prod*`, respect change windows.

## Checklist
- [ ] No sudo; no curl-pipe-sh
- [ ] Unfamiliar → info + deps tree; `brew outdated` stated before bulk upgrades
- [ ] Tap/cask source sanctioned and stated; `brew list --pinned` checked
- [ ] `brew uses --installed` for uninstalls; dry-run stated; zap paths named
