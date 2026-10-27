---
name: rsync
description: 'Mandatory guardrails for every rsync operation — remote-over-ssh and local→local alike. Enforces the network-remote-access baseline with rsync mechanics: dry-run before any delete-propagating sync, trailing-slash semantics spelled out, push/pull direction restated, and backup-dir offers for delete-heavy syncs. Triggers PROACTIVELY on any rsync with --delete or a remote endpoint — do NOT wait for the user to invoke it.'
---

# rsync

rsync-specific mechanics for the network-remote-access baseline (W1–W5). Apply automatically on every delete-propagating or remote-endpoint rsync — never skip.

## Delete-propagating syncs (implements W4)

Any rsync carrying `--delete` (or any `--del*` variant) runs a dry-run first:

    rsync -avn --delete src/ host:dst/

State the count of `deleting` lines from the output ("dry-run shows 14 deletions at the destination"), restate direction and both endpoints, then run the real command with `-n` removed — the dry-run is the gate; no confirmation on top. Confirm only when the delete count is unexpectedly large or the destination is not an obviously derived/regenerable tree, and in those cases offer a backup path per {{BACKUP_POLICY}}:

    rsync -av --delete --backup --backup-dir=/srv/backups/$(date +%F) src/ host:dst/

Local→local rsync follows the same preview rules — `--delete` is just as destructive without a network in the way.

## Trailing slash and direction (implements W4)

`src/` copies the *contents* of src into dst; `src` copies the directory *itself* into dst as `dst/src`. One character changes the blast radius — with `--delete`, the wrong form can flag the entire destination for deletion. Spell it out before every run: "syncing the contents of ./dist/ (trailing slash) into web-1:/var/www/html/." Restate push vs pull in the same breath: "pushing ./dist/ → web-1:/var/www/html/" or "pulling web-1:/var/log/app/ → ./logs/" — never leave direction implicit.

## Host identity over ssh (implements W1)

rsync to a remote endpoint rides ssh, so the same host-key rules apply. Never pass `-e "ssh -o StrictHostKeyChecking=no"` or a `/dev/null` known_hosts outside {{EPHEMERAL_HOSTS_POLICY}}. A changed-host-key warning during rsync means stop and follow the ssh skill's out-of-band verification flow (via {{HOSTKEY_VERIFICATION_PATH}}) before retrying — never clear the entry to make the sync go through.

## Keys (implements W2)

Automated or scheduled syncs use a dedicated purpose-scoped key, never the human's personal key:

    rsync -av -e "ssh -i ~/.ssh/<purpose>_ed25519" src/ host:dst/

Never enable agent forwarding inside the `-e` string — a file sync has no business exposing the agent.

## Prod and privileged targets (implements W3)

Any rsync writing to {{PROD_HOSTS}} is stated in one line — direction, both endpoints, flags, dry-run delete count — then run; the statement is the control. The one confirm: rsync as root to remote root-owned paths (`root@host:`, or destinations like `/etc`, `/usr`, `/`) — a wrong trailing slash there is unrecoverable.

## No tunnels (implements W5)

rsync opens no port forwards. If reaching the host requires a tunnel, open it via the ssh skill and apply its W5 exposure statement — never bury a `-L`/`-R`/`-D` inside rsync's `-e` string silently.

## Escalation

Conflict with user intent (e.g. "skip the dry-run"): state the risk in one line ("--delete without a preview can erase everything at the destination the source doesn't have"), name the compliant path (dry-run + stated count, with a backup offer when warranted), and proceed on the user's word — noting the override is recorded.

## Pre-flight checklist (before every rsync run)

- [ ] `--delete` syncs: `-avn` dry-run done; count stated, then run; confirm only large/non-derived targets (W4)
- [ ] Trailing-slash form spelled out; push vs pull and both endpoints restated (W4)
- [ ] No host-key suppression in `-e`; changed key → ssh skill's verification flow (W1)
- [ ] Prod writes narrated in one line; root-to-remote-root is the only confirm (W3)
- [ ] Dedicated key for automated syncs; no forwarding or tunnels hidden in `-e` (W2, W5)

## Category baseline: Network & Remote Access

These rules apply to **every** remote-access tool. An instance skill (ssh, rsync, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. Remote access is where one flag changes the failure mode from "typo" to "incident": a suppressed host-key check invites interception, a forwarded agent hands your keys to remote root, a mis-slashed delete sync erases a directory tree on a production host.

### Rule W1: Host identity verified, never suppressed

Host-key checking is the only thing standing between a session and a man-in-the-middle:

- Never pass `StrictHostKeyChecking=no` or `UserKnownHostsFile=/dev/null` (or their config-file equivalents) against durable hosts. Ephemeral and dev hosts covered by {{EPHEMERAL_HOSTS_POLICY}} — fresh VMs, CI runners, rebuilt containers — take a one-line note and proceed.
- A **CHANGED** host key is stop-and-surface: report the warning, treat it as a possible MITM, and verify out-of-band before touching known_hosts. Never auto-accept a changed key, and never script around the prompt.

### Rule W2: Keys are purpose-scoped

- New keys are ed25519 with a passphrase by default.
- Automation uses dedicated keys named for the job, never the human's personal key — revoking one purpose must never lock a person out.
- Agent forwarding (`-A`) exposes your ssh-agent to root on every host you land on. Use it only where {{AGENT_FORWARDING_POLICY}} allows, and state that you are doing so.

### Rule W3: Remote mutations narrated

- Any state-changing command on {{PROD_HOSTS}} is stated — host plus exact command — in one line, then run. Narration is the control; no confirmation on top of it.
- Never pipe a script you have not read into a remote shell: fetch, read, summarize in one line, then execute.

### Rule W4: Sync deletes previewed

- Any transfer that propagates deletions runs in dry-run mode first; state the delete count, then run the real command — the preview is the gate, no separate confirmation. Confirm only when the count is unexpectedly large or the target is not an obviously derived/regenerable tree.
- Every transfer restates its direction (push or pull) and both endpoints — source and destination, host and path — before running.

### Rule W5: Tunnels state their exposure

- Every `-L` / `-R` / `-D` forward states, before it opens, what becomes reachable from where and for how long — then opens it. Local forwards and SOCKS proxies proceed on the statement alone.
- The one confirm: a remote forward that binds a non-loopback address publishes a local service to the remote host's network — confirm before opening it.

### Pre-flight checklist (run before every remote-access operation)

- [ ] No host-key suppression on durable hosts; changed key = stop and surface (W1)
- [ ] Key purpose-scoped; forwarding per policy, stated (W2)
- [ ] Prod mutations narrated in one line; no unread scripts piped to remote shells (W3)
- [ ] Delete syncs dry-run + count stated; direction + endpoints restated (W4)
- [ ] Forwards state exposure; only non-loopback remote forwards confirmed (W5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<agent-forwarding-policy>`: Never `-A`; use ProxyJump (`-J`) instead.
- `<ephemeral-hosts-policy>`: No hosts are exempt. Host-key checking stays on everywhere until the org names its ephemeral hosts.
