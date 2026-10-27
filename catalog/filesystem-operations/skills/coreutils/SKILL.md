---
name: coreutils
description: 'Mandatory guardrails for every destructive coreutils invocation (rm, find -delete, dd, mkfs, diskutil, chmod/chown -R, shred, tar/unzip extraction) in a terminal. Enforces the filesystem-operations baseline with shell mechanics: list-before-delete flows, resolved-variable echoes, device read-back, and archive inspection. Triggers PROACTIVELY on any destructive shell command — do NOT wait for the user to invoke it.'
---

# Coreutils

Shell mechanics for the filesystem-operations baseline (Z1–Z5). Apply automatically on every destructive command — never skip. These commands report success whether they deleted the right thing or the wrong one.

## Enumerate before you destroy (implements Z1)

Before any `find … -delete` or `find … -exec rm`, run the identical expression in list form and state count + first paths:

    find <target> -name '<pat>' | head -20
    find <target> -name '<pat>' | wc -l

State "N files match, first 20 shown" → then run the same expression with `-delete` — the listing is the gate, no confirmation on top. Confirm only when the matches look like irreplaceable data outside the project tree:

    find <target> -name '<pat>' -delete

Before any `rm` with a glob or variable, resolve it visibly first:

    ls -d <glob>        # or: echo <glob>
    echo "$DIR"         # a variable you have not echoed this session is unverified

`rm -rf "$DIR/"` with an empty `$DIR` is `rm -rf /`. The delete must use the exact expression that was listed.

## Refused scopes and the paved road (implements Z2)

Refuse outright — no confirmation flow, no override:

- Recursive deletion of `/`, `$HOME`, `.`, `..`, or any mount point.
- Any deletion path containing an unresolved or unverified variable.

Everything else follows {{DELETION_POLICY}}. For interactive deletions the paved road is {{TRASH_COMMAND}} — recoverable beats gone. For bulk deletes without trash, use `rm -I` (prompt once per batch) after the Z1 listing.

## Block device writes (implements Z3)

Before `dd`, `mkfs`, or `diskutil erase`, read back the device identity:

    diskutil info <disk>          # macOS: size, name, mounted state
    lsblk -o NAME,SIZE,LABEL,MOUNTPOINT   # Linux

Restate to the user — "writing to /dev/disk4: 32 GB, 'SANDISK USB', not mounted — confirm?" — one confirmation, then write; the read-back plus restatement replaces any second round:

    dd if=<image> of=<device> bs=4M status=progress conv=fsync

Mounted target, or a size that doesn't match what the user described → stop. There is no recovery from the wrong `of=`.

## Recursive permission changes (implements Z4)

Single-file and small in-project chmod flows freely — no ceremony. For `-R` over roughly 100+ files or anything outside the project tree, state scope and before→after first:

    find <scope> | wc -l          # how many files this touches
    stat -f '%Sp %N' <scope>      # macOS current mode (Linux: stat -c '%A %n')

State "chmod -R 755 on <scope>: 312 files, 700 → 755" → then execute:

    chmod -R <mode> <scope>

Mode `777`, `+s` (setuid/setgid), or ownership changes outside the project tree: one-line callout of the security implication, then run. Prefer the narrowest scope — one file over `-R`.

## Archive extraction (implements Z5)

List before extracting; state entry count and top-level paths:

    tar -tzf <a>.tgz | head -20
    unzip -l <a>.zip

Any entry with an absolute path or a `..` component is zip-slip — **stop and report; do not extract.** Otherwise extract into a fresh directory:

    mkdir extract-<name> && tar -xzf <a>.tgz -C extract-<name>

Never pass `-P` / `--absolute-names` to tar, and never extract into `.`, `$HOME`, or a populated tree.

## Escalation

Z2 root scopes are the only hard refusals — no override exists for them. Everything else: state the risk in one line ("this glob is unverified — one typo deletes the parent tree"), name the compliant path, and proceed on the user's word — noting the override is recorded.

## Pre-flight checklist (before every destructive command)

- [ ] `find`/glob listed first (`head -20` + `wc -l`); count stated; delete uses the same expression
- [ ] Variables echoed resolved; target is not `/`, `$HOME`, `.`, `..`, or a mount point
- [ ] Device identity read back (`diskutil info` / `lsblk`); one confirm before dd/mkfs/erase
- [ ] Big/out-of-tree chmod/chown -R: count + before→after stated; 777/+s called out
- [ ] Archive listed; no absolute/`..` entries; extracted into `extract-<name>/`

## Category baseline: Filesystem Operations

These rules apply to **every** destructive filesystem tool. An instance skill (coreutils, PowerShell, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. This category exists because these commands have no undo: a mistyped glob deletes a year of work, a wrong-disk `dd` destroys a machine, and no error message warns you first — the command succeeds.

### Rule Z1: Enumerate before you destroy

Never run a recursive or bulk deletion without first listing what will match:

1. Run the same match expression in list-only form; state the total count and the first N paths — then delete. The listing is the gate; no confirmation on top of it. Confirm only when the matches look like irreplaceable data outside the project tree.
2. Wildcards and shell variables are echoed **resolved** before use — `rm -rf "$DIR/"` with an empty `$DIR` is `rm -rf /`. If you have not seen the expanded value this session, you do not know what you are deleting.
3. The deletion command must use the exact same expression that was listed — never "list one thing, delete another".

### Rule Z2: Root scopes are refused

Some targets are refused outright, no confirmation flow, no override:

- Recursive deletion of `/`, `$HOME`, `.`, `..`, or any mount point.
- Any deletion path containing an unresolved or unverified variable.
- Everything else follows {{DELETION_POLICY}} — trash where available, `-I` interactive batching for bulk deletes.

Refusal means stating the risk and proposing a scoped alternative, not silently doing nothing.

### Rule Z3: Block devices get read-back plus one confirmation

`dd`, `mkfs`, `diskutil erase`, and anything else that writes raw to a device — block devices are the definition of irreversible, so this keeps exactly one confirmation:

1. Read back the target device's identity first — size, label/name, mounted state — using the instance skill's exact probe.
2. Restate it to the user ("writing 4.2 GB image to /dev/disk4, 32 GB, 'SANDISK USB', not mounted") and get one confirmation before the write — the read-back plus restatement replaces any second round.
3. A mounted device or a size that doesn't match expectations → stop. Wrong-disk writes are unrecoverable by design.

### Rule Z4: Permission changes are surgical

Single-file and small in-project chmod/chown flow freely — the ceremony is reserved for the shapes that can't be undone at scale:

- `-R` over roughly 100+ files, or any recursive change outside the project tree, states the scope (file count) and before→after mode/owner first, then runs.
- Mode `777`, setuid/setgid bits, and ownership changes outside the project tree get a one-line callout naming the security implication before running — these turn a permissions fix into an incident when unstated.
- Prefer the narrowest scope that solves the problem: one file over one directory, one directory over `-R`.

### Rule Z5: Archives are inspected before extraction

An archive is untrusted input until its listing says otherwise:

1. List the archive's contents before extracting; state the entry count and top-level paths.
2. Any entry with an absolute path or a `..` component is an attack (zip-slip) — **stop and report**, do not extract.
3. Extract into a fresh, dedicated directory — never directly into `.`, `$HOME`, or an existing populated tree where entries can overwrite files.

### Pre-flight checklist (run before every destructive filesystem operation)

- [ ] Matches listed, count stated, then delete; variables/globs echoed resolved (Z1)
- [ ] Target is not `/`, `$HOME`, `.`, `..`, a mount point, or an unresolved variable (Z2)
- [ ] Block device identity read back and restated; one confirm before write (Z3)
- [ ] Big/out-of-tree chmod/chown -R: count + before→after stated; 777/setuid called out (Z4)
- [ ] Archive listed; no absolute/`..` entries; extraction into a fresh directory (Z5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<trash-command>`: No default. Use `trash` (macOS) or `trash-put` (Linux, trash-cli) when one is installed; otherwise fall back to `rm` with the list-first flow and say so.
