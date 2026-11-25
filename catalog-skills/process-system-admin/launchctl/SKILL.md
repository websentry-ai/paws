---
name: launchctl
description: 'Mandatory guardrails for every launchd mutation on macOS (bootstrap, bootout, load, unload, enable, disable, kickstart, kill) and every LaunchAgent or LaunchDaemon plist the agent writes. Enforces the process-system-admin baseline with launchd mechanics: full-plist diffs before bootstrap, reverse-DNS labels, user-vs-root scope choice, and explicit persistence callouts for RunAtLoad/KeepAlive. Triggers PROACTIVELY on any launchctl mutation or plist write — do NOT wait for the user to invoke it.'
---

# launchctl

macOS mechanics for the process-system-admin baseline (J1–J5). Apply automatically on every launchd mutation — never skip. Reads (`launchctl list`, `launchctl print`) are always allowed and expected **before** the mutation. Prefer modern `bootstrap`/`bootout` over legacy `load`/`unload`; both are covered below, since much existing tooling still emits `load`.

## Read the existing jobs first (implements J4)

Before adding any job, list what the org already runs — never silently shadow existing automation:

    launchctl list | grep {{ORG_LABEL_PREFIX}}

State the new job's schedule (`StartCalendarInterval`/`StartInterval`) and command (`ProgramArguments`) verbatim before installing, and carry {{AGENT_ATTRIBUTION_CONVENTION}} as a comment in the plist: `<!-- # added-by: ai-agent <date> <reason> -->`.

## Plists are code (implements J2)

Show the full plist (and a diff when editing an existing one) **before** installing it. Labels follow {{UNIT_CONVENTIONS}} — reverse-DNS, e.g. `com.<org>.<purpose>` — and the filename matches the label. `ProgramArguments` references a script file when that is the org norm — no inline `/bin/sh -c '…'` soup.

    # User-scope agent (default)
    launchctl bootstrap gui/$UID ~/Library/LaunchAgents/<label>.plist

    # Verify it landed
    launchctl print gui/$UID/<label>

Legacy equivalent (`launchctl load ~/Library/LaunchAgents/<label>.plist`) still works and appears in existing scripts; use bootstrap for new work.

## Scope choice and blast radius (implements J1)

Choose scope deliberately and state it: `~/Library/LaunchAgents` (user session, domain `gui/$UID`) vs `/Library/LaunchDaemons` (system-wide, domain `system/`, requires sudo — the one confirm; root daemons always confirm before install). User-scope `bootout`, `kickstart -k`, and `disable` take a one-line statement of what depends on the job (login workflows, watch paths, jobs it feeds) and run. On {{PROD_HOSTS}} (build machines, shared Macs), state blast radius and proceed; confirm only stop/disable of a shared service. Removal is narrated, not silent:

    launchctl bootout gui/$UID/<label>

## Kill escalation (implements J3)

    # TERM first, delivered via launchd
    launchctl kill SIGTERM gui/$UID/<label>

    # Only after a stated grace window
    launchctl kill SIGKILL gui/$UID/<label>

SIGKILL is narrated as last resort — cleanup handlers will not run. `KeepAlive` jobs respawn after a kill: to actually stop one, `bootout` or `launchctl disable` — say which you are doing. Never `pkill`/`killall` a launchd-managed process by pattern; go through the label. If a pattern kill is unavoidable, list the matches with `pgrep -fl <pattern>` first.

## RunAtLoad / KeepAlive is persistence (implements J5)

Any plist with `RunAtLoad`, `KeepAlive`, or LaunchDaemon scope survives reboot/login — always name it. A user-scope LaunchAgent for the user's own workflow: one line ("this plist sets RunAtLoad — it will run at every login") and proceed. LaunchDaemon scope, root ownership, or `launchctl enable` in the `system/` domain: name it and confirm first — silent system persistence is indistinguishable from malware.

## Escalation

Conflict with user intent (e.g. "just load it, skip showing the plist"): state the risk in one line ("an unreviewed RunAtLoad plist is silent login persistence"), name the compliant path (show the plist, state the persistence), and proceed on the user's word — noting the override is recorded.

## Pre-flight checklist (before every launchctl mutation)

- [ ] `launchctl list | grep {{ORG_LABEL_PREFIX}}` read first; schedule + command stated verbatim with attribution (J4)
- [ ] Full plist / diff shown; label reverse-DNS per {{UNIT_CONVENTIONS}}; script file over inline shell (J2)
- [ ] Scope stated; blast radius in one line; only root daemons and shared-service stop/disable confirmed (J1)
- [ ] SIGTERM before SIGKILL, via label; no pattern kills without `pgrep -fl` listing (J3)
- [ ] Persistence named; user-scope proceeds, system/root scope confirmed; install verified with `launchctl print` (J5)

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

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<org-label-prefix>`: The org's reverse-DNS label prefix (e.g. `com.acme`), implied by its unit conventions. Unknown: list all jobs (`launchctl list`) and say which prefix you assumed.
