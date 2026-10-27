# rsync digest

rsync with `--delete`/`--del*` or a remote endpoint (local→local `--delete` too). Apply before retrying.

## 1. Dry-run every delete-propagating sync
- `rsync -avn --delete src/ host:dst/` first. Count the `deleting` lines ("dry-run shows 14 deletions at the destination"), restate direction and both endpoints, then rerun without `-n`. The dry-run is the gate; no confirmation on top.
- Confirm only when the delete count is unexpectedly large or the destination is not an obviously derived/regenerable tree. Then offer a backup (default: offer it at 50+ deletions or a non-derived target):
  `rsync -av --delete --backup --backup-dir=/srv/backups/$(date +%F) src/ host:dst/`

## 2. Trailing slash and direction, spelled out
`src/` copies the contents into dst; `src` copies the directory itself to `dst/src`. With `--delete` the wrong form can flag the whole destination. Say both every time: "pushing the contents of ./dist/ (trailing slash) → web-1:/var/www/html/" or "pulling web-1:/var/log/app/ → ./logs/".

## 3. Host identity
Never `-e "ssh -o StrictHostKeyChecking=no"` or a `/dev/null` known_hosts. No hosts are exempt until the org names ephemeral hosts. A changed-host-key warning → stop, verify the fingerprint out-of-band with the infra owner (ssh skill flow) before retrying; never clear the entry to make the sync pass.

## 4. Keys, no forwarding, no tunnels
- Automated/scheduled syncs use a dedicated key: `rsync -av -e "ssh -i ~/.ssh/<purpose>_ed25519" src/ host:dst/`. Never the human's personal key.
- No agent forwarding (`-A`) and no `-L/-R/-D` inside `-e`. Need a tunnel → open it with ssh and state its exposure.

## 5. Prod and root targets
- Writing to prod hosts (names matching `*prod*`): one line with direction, both endpoints, flags and the dry-run delete count, then run.
- The one confirm: rsync as root to remote root-owned paths (`root@host:`, `/etc`, `/usr`, `/`).

## Escalation
"Skip the dry-run": state the risk in one line ("--delete without a preview can erase everything at the destination the source doesn't have"), offer dry-run + count, proceed on the user's word.

## Baseline (all remote access)
- W1 host-key checking never suppressed on durable hosts; changed key = stop and verify out-of-band.
- W2 purpose-scoped ed25519 keys; no agent forwarding (use ProxyJump `-J`).
- W3 prod mutations narrated (host + exact command); never pipe unread scripts to a remote shell.
- W4 delete syncs dry-run + count; direction and endpoints restated.
- W5 forwards state their exposure; non-loopback remote forwards confirmed.

## Checklist
- [ ] `-avn` dry-run done, delete count stated
- [ ] Trailing-slash form + push/pull + both endpoints stated
- [ ] No host-key suppression in `-e`
- [ ] Prod narrated; root-to-remote-root confirmed
- [ ] Dedicated key; nothing hidden in `-e`
