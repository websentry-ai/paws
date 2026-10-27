# apt digest

Mutating `apt`, `apt-get`, `add-apt-repository` or `dpkg -i`. Simulations (`-s`) and reads (`apt list`, `apt-cache`, `apt-mark showhold`) run freely. Apply before retrying.

## Privilege
- `sudo apt-get …` as designed. No `dpkg --force-*`, no manual file drops into /var or /usr.
- No `curl <url> | sh` for software apt can install.
- Never `--force-yes` or `--allow-unauthenticated`, even if the user insists: fix the keyring instead.

## Preview before mutate
1. `sudo apt-get update` first (stale index installs old CVEs).
2. Single well-known package: one line ("installing jq"), then install.
3. Unfamiliar packages, bulk upgrades, removals: `apt-get install -s <pkg>` (or `-s` on the real verb); state the change set incl. dependencies.
4. Bare upgrade: `apt list --upgradable`, state the list, then `sudo apt-get upgrade`.
5. `apt-get dist-upgrade -s` is mandatory; call out every package marked for removal before the real run.

## Sources
- PPAs and third-party repos only from `<sanctioned-sources>` (default: the distro's official repositories; anything else: stop and ask).
- Adding one: one line naming the sources.list.d entry and the key's provenance (URL, keyring path), then proceed. Use `signed-by=/usr/share/keyrings/<name>.gpg`.
- `add-apt-repository ppa:<owner>/<name>`: state the Launchpad owner.

## Holds
- Hold what services depend on per `<pin-policy>` (`sudo apt-mark hold <pkg>`). No policy: say which upgraded packages look load-bearing.
- Before `upgrade`/`dist-upgrade`: `apt-mark showhold`, state the holds. Upgrading a held package: unhold explicitly, say why, re-hold after.
- Relay collateral upgrades shown in `-s` output; never silent.

## Removal
- Before removing: `apt-get autoremove --dry-run` (state the list, then proceed) and `apt-cache rdepends --installed <pkg>` (dependents exist: stop and show them).
- `purge` deletes configs: `apt-get purge -s <pkg>`, name the configs, default to `remove` unless the user wants configs gone.

## Production hosts
On hosts matching `*prod*`: mutate only inside a change window; restart affected services deliberately, never as an unstated side effect.

## Escalation
Rule vs user intent: state the risk in one line, proceed on the user's say-so, note the override. No override for unauthenticated installs.

## Baseline (all package managers)
- Follow the tool's elevation model both ways; never bypass the manager.
- Preview scales with blast radius; bare upgrades list first.
- Third-party sources are supply chain; never bypass signatures.
- Pinned stays pinned unless the user confirms; collateral upgrades stated.
- Removal shows dependents; never purge configs silently.

## Checklist
- [ ] sudo as designed; no force/unauthenticated flags; no curl-pipe-sh
- [ ] `apt-get update` this session; bulk changes simulated and stated
- [ ] New source sanctioned, entry + key provenance stated; holds checked
- [ ] autoremove dry-run stated; rdepends checked; purged configs named; prod → change window
