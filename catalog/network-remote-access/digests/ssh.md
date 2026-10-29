# ssh digest

ssh with host-key overrides, `-A`, `-L/-R/-D`; any scp/sftp; `ssh-keygen -R`. Apply before retrying.

## 1. Host keys
- Never `-o StrictHostKeyChecking=no` or `-o UserKnownHostsFile=/dev/null` against durable hosts. No hosts are exempt until the org names ephemeral hosts; for those: "host <h> is ephemeral per policy — skipping host-key persistence for this connection only."
- `WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!` → stop. It is a rebuild or a MITM. Verify the new fingerprint out-of-band with the infra owner. Only after that: `ssh-keygen -R <host>` and reconnect.
- Never run `ssh-keygen -R` as a reflex to the warning.

## 2. Keys and forwarding
- New keys: `ssh-keygen -t ed25519 -f ~/.ssh/<purpose>_ed25519 -C "<purpose>"`, passphrase, one per purpose. Automation never reuses a personal key.
- `-A` exposes your agent to remote root. Default policy: never `-A`. Use ProxyJump: `ssh -J bastion.example.com target-host`. If the org allows `-A` somewhere, state "forwarding agent to <host> per policy" before connecting.

## 3. Prod commands
State-changing commands on prod hosts (names matching `*prod*`): one line with host + exact command, then run. Dev/ephemeral hosts: no narration. Never pipe an unread script into a remote shell (`curl … | ssh host 'bash -s'`): fetch, read, summarize, then run.

## 4. scp / sftp
Restate direction and both endpoints: "pushing ./build/app.tgz → prod-web-1:/opt/app/app.tgz (overwrites the existing file)". scp overwrites silently: say if the destination exists. `scp -r`: state the tree. No manual remote deletes to emulate a sync (that's rsync with a dry-run).

## 5. Tunnels state exposure
- Before `-L/-R/-D`: what becomes reachable from where, for how long. E.g. `ssh -L 5432:db:5432 bastion` → "remote db:5432 reachable at localhost:5432 while the session lives". Then open.
- `-D`: also state that proxied traffic egresses from the remote host.
- The one confirm: `-R` binding a non-loopback address (`-R 0.0.0.0:8080:…`, `-R *:8080:…`).

## Escalation
User wants the check suppressed: one line ("a MITM is indistinguishable from a rebuild"), offer out-of-band verify then `ssh-keygen -R`, proceed on their word.

## Baseline (all remote access)
- W1 host identity verified, never suppressed on durable hosts.
- W2 purpose-scoped ed25519 keys; forwarding only per policy, stated.
- W3 prod mutations narrated; no unread scripts into remote shells.
- W4 transfers restate direction + endpoints; delete syncs dry-run first.
- W5 forwards state exposure; non-loopback remote forwards confirmed.

## Checklist
- [ ] No suppression flags; changed key → verify before `ssh-keygen -R`
- [ ] Purpose-scoped key; no `-A` (use `-J`)
- [ ] Prod command stated in one line
- [ ] Transfer direction + endpoints restated
- [ ] Forward exposure stated; non-loopback `-R` confirmed
