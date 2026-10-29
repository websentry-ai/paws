# systemctl digest

systemd state change (start, stop, restart, reload, enable, disable, mask, edit, daemon-reload, kill), crontab write, or kill -9 / pkill / killall. Reads (`systemctl status`, `list-units`, `list-timers`, `is-active`, `pgrep`) are expected first. Apply before retrying.

## Blast radius
- Dev and `--user` restarts: one line, run.
- Shared/prod units: `systemctl list-dependencies --reverse <unit>`, state the blast radius ("restarting postgresql also bounces app-api.service and worker.service"), then run. On hosts matching `*prod*` add expected downtime to that line.
- The one confirm: `stop`, `disable` or `mask` on hosts matching `*prod*`, always with the reason.

## Unit files are code
- Overrides via `systemctl edit <unit>` (drop-in survives upgrades).
- New or rewritten units: show the file as a diff, `systemd-analyze verify /etc/systemd/system/<unit>.service`, then `systemctl daemon-reload`.
- Follow the org's unit conventions (reverse-DNS or org-prefixed names, one service per unit, owner in a comment).
- Scope stated: `/etc/systemd/system` vs `~/.config/systemd/user` (`--user`).
- No `ExecStart=/bin/sh -c '…'` when a script file is the norm.

## Kill escalation
- `kill <pid>` (TERM, single known PID) flows freely.
- Managed unit: `systemctl kill -s TERM <unit>`.
- Pattern: `pgrep -fl <pattern>` first, show matches, then `pkill -TERM -f <pattern>`. Never pkill/killall by loose pattern without the listing.
- `-KILL` only after TERM fails within a stated grace window, narrated ("TERM ignored after 10s; sending KILL — cleanup handlers will not run").

## Cron and timers
- Read first: `crontab -l`, `systemctl list-timers`. Never shadow existing automation.
- State schedule + command verbatim, with `# added-by: ai-agent <date> <reason>`:
  `(crontab -l 2>/dev/null; echo '# added-by: ai-agent <date> <reason>'; echo '<schedule> <command>') | crontab -`
- Prefer a systemd timer when the job pairs with a unit (unit-file rules apply).

## Persistence
- `systemctl enable` and any `@reboot` line survive reboot: always name it.
- User scope for the user's own workflow: one line, proceed. System scope or root: name it and confirm first.

## Escalation
"Just kill -9 everything matching node": state the risk ("-9 skips cleanup, a loose pattern can hit the wrong daemon"), name the compliant path (`pgrep -fl` + TERM), proceed on the user's word, note the override.

## Baseline (all process/service managers)
- Blast radius in one line; only prod stop/disable confirmed.
- Unit files are code: diff first, conventions, deliberate scope.
- TERM before KILL; listing before pattern kills.
- Read schedules first; new jobs verbatim with `# added-by: ai-agent <date>`.
- Persistence is a security event.

## Checklist
- [ ] Blast radius stated; only prod stop/disable/mask confirmed
- [ ] Drop-in or diff shown; `systemd-analyze verify` + `daemon-reload`
- [ ] TERM before KILL; `pgrep -fl` before pkill/killall
- [ ] `crontab -l` / `list-timers` read; entry verbatim with attribution
- [ ] Persistence named; system scope confirmed
