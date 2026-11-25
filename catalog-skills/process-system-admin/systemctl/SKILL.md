---
name: systemctl
description: 'Mandatory guardrails for every systemd state change (start, stop, restart, reload, enable, disable, mask, unit edits, daemon-reload) plus crontab edits and kill/pkill/killall on Linux hosts. Enforces the process-system-admin baseline with systemd mechanics: reverse-dependency blast radius, drop-in unit edits with verify, TERM-before-KILL escalation, and attributed cron entries. Triggers PROACTIVELY on any systemctl state change, crontab write, or kill — do NOT wait for the user to invoke it.'
---

# systemctl

Linux mechanics for the process-system-admin baseline (J1–J5); also covers crontab and kill/pkill mechanics on Linux hosts. Apply automatically on every service, cron, or process mutation — never skip. Reads (`systemctl status`, `list-units`, `list-timers`, `is-active`, `pgrep`) are always allowed and expected **before** the mutation.

## Blast radius before restart (implements J1)

Dev and user-scope (`--user`) restarts take one line and run. For shared/prod units, list what depends on the unit first:

    systemctl list-dependencies --reverse <unit>

State the blast radius ("restarting postgresql also bounces app-api.service and worker.service"), then run — on {{PROD_HOSTS}} add expected downtime to the same line; the statement is the control:

    systemctl restart <unit>

The one confirm: `stop`, `disable`, or `mask` of a unit on {{PROD_HOSTS}} — and those always state why.

## Unit files are code (implements J2)

Never mutate a unit without showing the change first:

    # Preferred for overrides — drop-in file, survives package upgrades
    systemctl edit <unit>

    # New or rewritten units: write the file, show it as a diff, then
    systemd-analyze verify /etc/systemd/system/<unit>.service
    systemctl daemon-reload

Units follow {{UNIT_CONVENTIONS}}. System scope is `/etc/systemd/system`; user scope is `~/.config/systemd/user` (managed with `systemctl --user`) — choose deliberately and state which. No inline `ExecStart=/bin/sh -c '…'` shell soup when a script file is the org norm: write the script, reference it.

## Kill escalation (implements J3)

A TERM to a single, known PID (`kill <pid>`) flows freely — no narration. Otherwise:

    # Managed unit → let systemd deliver the signal
    systemctl kill -s TERM <unit>

    # Loose pattern → LIST the matches first, always
    pgrep -fl <pattern>
    pkill -TERM -f <pattern>

Only if TERM fails after a stated grace window: escalate to `-KILL`, narrated as last resort ("TERM ignored after 10s; sending KILL — cleanup handlers will not run"). Never `pkill`/`killall` by loose pattern without showing the `pgrep -fl` matches first — the listing is the gate.

## Cron and timers (implements J4)

Read the existing schedule before adding to it — never silently shadow org automation:

    crontab -l
    systemctl list-timers

State the new entry's schedule + command verbatim before installing, with the attribution comment `# added-by: ai-agent <date> <reason>`:

    (crontab -l 2>/dev/null; echo '# added-by: ai-agent <date> <reason>'; echo '<schedule> <command>') | crontab -

Prefer a systemd timer when the job pairs with a unit — a new timer is a new unit, so J2 applies in full, and `systemctl list-timers` is read first.

## Enable is persistence (implements J5)

`systemctl enable` — and any `@reboot` cron line — makes the change survive reboot. Always name it:

    systemctl enable --now <unit>

User-scope (`systemctl --user enable`, user crontab) for the user's own workflows: one line ("enabling <unit> — persists across reboots for your session") and proceed. System-scope or root persistence: name it and confirm first — silent system persistence is indistinguishable from malware.

## Escalation

Conflict with user intent (e.g. "just kill -9 everything matching node"): state the risk in one line ("-9 skips cleanup handlers, and a loose pattern can hit the wrong daemon"), name the compliant path (`pgrep -fl` + TERM first), and proceed on the user's word — noting the override is recorded.

## Pre-flight checklist (before every systemctl/cron/kill mutation)

- [ ] Blast radius stated in one line; only prod stop/disable/mask confirmed (J1)
- [ ] Unit edits via drop-in or file shown as diff per {{UNIT_CONVENTIONS}}; `systemd-analyze verify` + `daemon-reload` run (J2)
- [ ] TERM before KILL; `pgrep -fl` matches shown before any pkill/killall (J3)
- [ ] `crontab -l` / `list-timers` read first; schedule + command stated verbatim with attribution (J4)
- [ ] Persistence named; user-scope proceeds, system-scope confirmed (J5)

## Category baseline: Process & System Administration

These rules apply to **every** tool that starts, stops, kills, schedules, or persists processes on a host. An instance skill (systemctl, launchctl, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. Process management mutates the live machine: a careless restart takes its dependents down with it, a loose pkill hits the wrong daemon, and a silently installed @reboot entry is persistence — the same move malware makes.

### Rule J1: Service changes state their blast radius

Dev and user-scope services restart on a one-line statement — no ceremony. For anything shared or prod:

1. List the service's reverse dependencies / dependents using the instance skill's exact mechanics.
2. State the blast radius in one line ("restarting X also bounces Y and Z"), then run — the statement is the control, even on {{PROD_HOSTS}}.
3. The one confirm: `stop` or `disable` of a service on {{PROD_HOSTS}} — leaving prod without a running service is the shape that pages someone.
4. `stop` without a paired restart plan, `disable`, and mask-style operations state why in the same line.

### Rule J2: Persistent units are code

Unit files, plists, and service definitions are configuration-as-code, not throwaway strings:

- New or edited unit files/plists are shown as a diff (full file for new ones) **before** installation.
- Naming, labels, and structure follow {{UNIT_CONVENTIONS}}.
- They land in the right scope — system vs user — chosen deliberately and stated.
- No inline ExecStart/ProgramArguments shell soup when a script file is the org norm: write the script, reference it from the unit.

### Rule J3: Kill escalates gradually

- TERM before KILL, always. A TERM to a single, known PID flows freely — no narration needed.
- `-9`/SIGKILL is narrated as a last resort: say that cleanup handlers will not run.
- Never `pkill`/`killall` by loose pattern without listing the matching processes first — show the match list, then kill; the listing is the gate.

### Rule J4: Scheduled tasks are registered, not scattered

- Read the existing schedule first (crontab listing, timer list, launchd job list) — never silently shadow or duplicate existing org automation.
- New cron entries, timers, and launchd jobs state their schedule + command **verbatim** before installing.
- Every agent-installed job carries {{AGENT_ATTRIBUTION_CONVENTION}} in a comment so ops can distinguish agent writes from human ones.

### Rule J5: Persistence is a security event

Anything that survives reboot or login — enable, RunAtLoad, KeepAlive, @reboot — is explicitly called out as **adding persistence**. Persistence is the attacker's favorite move; naming it is what separates an agent from malware. User-scope persistence for the user's own workflows (LaunchAgents, user timers) is stated in one line and proceeds; system-scope or root persistence is confirmed before installation.

### Pre-flight checklist (run before every process/service mutation)

- [ ] Blast radius stated in one line; only prod stop/disable confirmed (J1)
- [ ] Unit/plist shown as diff, follows {{UNIT_CONVENTIONS}}, correct scope (J2)
- [ ] TERM before KILL; pattern kills preceded by a match listing (J3)
- [ ] Existing schedule read; new job stated verbatim with attribution (J4)
- [ ] Persistence named; user-scope proceeds, system/root scope confirmed (J5)
