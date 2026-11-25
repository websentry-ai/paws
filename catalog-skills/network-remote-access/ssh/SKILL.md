---
name: ssh
description: 'Mandatory guardrails for every remote-access operation via ssh, scp, sftp, and ssh-keygen (key and known_hosts management). Enforces the network-remote-access baseline with OpenSSH mechanics: changed-host-key handling, purpose-scoped ed25519 keys, narrated prod commands, and tunnel exposure statements. Triggers PROACTIVELY on any ssh invocation with host-key overrides, forwarding, or agent flags, and on any scp/sftp transfer — do NOT wait for the user to invoke it.'
---

# SSH (ssh, scp, sftp, ssh-keygen)

OpenSSH-specific mechanics for the network-remote-access baseline (W1–W5). Apply automatically on every ssh-family invocation — never skip.

## Host keys (implements W1)

Never add `-o StrictHostKeyChecking=no` or `-o UserKnownHostsFile=/dev/null` against durable hosts. Ephemeral and dev hosts covered by {{EPHEMERAL_HOSTS_POLICY}} — fresh VMs, CI runners, rebuilt containers — take one line and proceed: "host <h> is ephemeral per policy — skipping host-key persistence for this connection only."

When ssh prints `WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!`:

1. **Stop.** The block means the host's key no longer matches known_hosts — either the host was legitimately rebuilt, or the connection is being intercepted (MITM).
2. Verify out-of-band via {{HOSTKEY_VERIFICATION_PATH}}: confirm the new fingerprint matches what the host actually serves.
3. Only after confirmation, clear the stale entry and reconnect:

       ssh-keygen -R <host>

Never run `ssh-keygen -R` as an automatic reaction to the warning — that is exactly the reflex a MITM relies on.

## Keys and agent forwarding (implements W2)

New keys are ed25519 with a passphrase, one key per purpose — automation never reuses the human's personal key:

    ssh-keygen -t ed25519 -f ~/.ssh/<purpose>_ed25519 -C "<purpose>"

`-A` (agent forwarding) exposes your ssh-agent to root on the remote host. Use it only where {{AGENT_FORWARDING_POLICY}} allows, and state it before connecting: "forwarding agent to <host> per policy." For reaching hosts through a bastion, prefer `-J` (ProxyJump) — it authenticates end-to-end without exposing the agent:

    ssh -J bastion.example.com target-host

## Remote commands on prod (implements W3)

Before any state-changing command on {{PROD_HOSTS}} — service restarts, package installs, file edits, anything not read-only — state the host and the exact command in one line, then run — the narration is the control, not a request for permission. Dev and ephemeral hosts need no narration at all. Never pipe a script you have not read into a remote shell:

    # forbidden until the script has been fetched, read, and summarized
    curl -s https://example.com/install.sh | ssh prod-host 'bash -s'

## File transfers (implements W4)

Every scp/sftp transfer restates direction and both endpoints before running: "pushing ./build/app.tgz → prod-web-1:/opt/app/app.tgz (overwrites the existing file)." scp overwrites silently — say when the destination already exists, if known. Recursive transfers (`scp -r`) state the tree being copied. scp/sftp never emulate delete-propagating syncs with manual remote deletes — that is the rsync skill's job, with its dry-run rules.

## Tunnels (implements W5)

Every `-L` / `-R` / `-D` forward states what becomes reachable from where before it opens:

    ssh -L 5432:db:5432 bastion.example.com

State: "this makes the remote `db:5432` (Postgres) reachable at localhost:5432 on this machine for as long as the session lives" — then open it; local forwards and `-D` proceed on the statement alone. `-D` (SOCKS proxy) additionally states that all proxied traffic egresses from the remote host. The one confirm: a remote forward binding a non-loopback address (`-R 0.0.0.0:8080:...` or `-R *:8080:...`) publishes a local service to the remote host's network — confirm before opening it.

## Escalation

Conflict with user intent: state the risk in one line (e.g. "suppressing the check accepts whatever key the network serves — a MITM is indistinguishable from a rebuild"), name the compliant path (out-of-band verify via {{HOSTKEY_VERIFICATION_PATH}}, then `ssh-keygen -R`), and proceed on the user's word — noting the override is recorded.

## Pre-flight checklist (before every ssh/scp/sftp invocation)

- [ ] No suppression flags on durable hosts; changed key → out-of-band verify before `ssh-keygen -R` (W1)
- [ ] Purpose-scoped ed25519 key; `-A` only per policy and stated; `-J` preferred (W2)
- [ ] Prod mutations stated host + command in one line; no unread scripts piped to remote shells (W3)
- [ ] Transfers restate direction and both endpoints (W4)
- [ ] Forwards state exposure; only non-loopback `-R` binds confirmed (W5)

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
