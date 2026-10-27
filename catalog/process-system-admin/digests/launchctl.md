# launchctl digest

launchd mutation (bootstrap, bootout, load, unload, enable, disable, kickstart, kill) or a LaunchAgent/LaunchDaemon plist write. Reads (`launchctl list`, `launchctl print`) are expected first. Apply before retrying.

## Read existing jobs first
- `launchctl list | grep <org-label-prefix>` (the org's reverse-DNS prefix, e.g. `com.acme`; unknown: list all and say which prefix you assumed). Never shadow or duplicate existing automation.
- State the new job's schedule (`StartCalendarInterval`/`StartInterval`) and `ProgramArguments` verbatim before installing.
- Attribution comment in the plist: `<!-- # added-by: ai-agent <date> <reason> -->`.

## Plists are code
- Show the full plist (diff when editing) before installing.
- Label reverse-DNS (`com.<org>.<purpose>`), filename matches label; one service per unit, owner in a comment.
- `ProgramArguments` points at a script file, not inline `/bin/sh -c '…'`.
- Prefer `launchctl bootstrap gui/$UID ~/Library/LaunchAgents/<label>.plist` over legacy `load`; verify with `launchctl print gui/$UID/<label>`.

## Scope and blast radius
- State the scope: `~/Library/LaunchAgents` (domain `gui/$UID`) vs `/Library/LaunchDaemons` (domain `system/`, sudo). Root daemons always confirm before install.
- User-scope `bootout`, `kickstart -k`, `disable`: one line on what depends on the job, then run.
- On hosts matching `*prod*` (build machines, shared Macs): state blast radius and proceed; confirm only stop/disable of a shared service.
- Removal is narrated: `launchctl bootout gui/$UID/<label>`.

## Kill escalation
- `launchctl kill SIGTERM gui/$UID/<label>` first; `SIGKILL` only after a stated grace window, narrated as last resort (cleanup handlers will not run).
- `KeepAlive` jobs respawn: to stop one, `bootout` or `launchctl disable`, and say which.
- No `pkill`/`killall` on launchd-managed processes; go through the label. Unavoidable pattern kill: `pgrep -fl <pattern>` first.

## Persistence
- `RunAtLoad`, `KeepAlive` or LaunchDaemon scope survives login/reboot: always name it.
- User-scope agent for the user's own workflow: one line ("this plist sets RunAtLoad — it will run at every login"), proceed.
- LaunchDaemon, root ownership, or `enable` in `system/`: name it and confirm first.

## Escalation
User says skip showing the plist: state the risk ("an unreviewed RunAtLoad plist is silent login persistence"), name the compliant path, proceed on their word, note the override.

## Baseline (all process/service managers)
- Blast radius in one line; the one confirm is stop/disable of a prod service.
- Unit files are code: diff first, conventions, deliberate scope, script over shell soup.
- TERM before KILL; list matches before pattern kills.
- Read the schedule before adding; new jobs verbatim with `# added-by: ai-agent <date>`.
- Persistence is a security event: user scope stated, system/root scope confirmed.

## Checklist
- [ ] Existing jobs listed; schedule + command verbatim with attribution
- [ ] Full plist/diff shown; reverse-DNS label; script file
- [ ] Scope stated; blast radius one line; root daemons / shared stop confirmed
- [ ] SIGTERM before SIGKILL via label; `pgrep -fl` before pattern kills
- [ ] Persistence named; install verified with `launchctl print`
