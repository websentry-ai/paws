# coreutils digest

Recursive/forced rm, find -delete, dd/mkfs/diskutil erase, chmod/chown -R, 777/setuid, shred, tar extraction. These commands report success whether they hit the right target or not. Apply before retrying.

## 1. Enumerate before you destroy
- `find … -delete` / `-exec rm`: run the identical expression as a listing first: `find <target> -name '<pat>' | head -20` and `| wc -l`. State "N files match, first 20 shown", then run the same expression with `-delete`. The listing is the gate; confirm only when matches look like irreplaceable data outside the project tree.
- `rm` with a glob or variable: resolve it visibly first (`ls -d <glob>`, `echo "$DIR"`). A variable not echoed this session is unverified. `rm -rf "$DIR/"` with an empty `$DIR` is `rm -rf /`.
- The delete uses exactly the expression that was listed.

## 2. Refused outright (no confirmation, no override)
- Recursive deletion of `/`, `$HOME`, `.`, `..`, or any mount point.
- Any deletion path containing an unresolved or unverified variable.
Refusing means stating the risk and proposing a scoped alternative.
Everything else: list-then-delete. Prefer a trash command (`trash` on macOS, `trash-put` from trash-cli on Linux) when installed; else `rm -I` after the listing, and say so.

## 3. Block devices: read back + one confirmation
Before `dd`, `mkfs`, `diskutil erase`: `diskutil info <disk>` (macOS) or `lsblk -o NAME,SIZE,LABEL,MOUNTPOINT` (Linux). Restate: "writing to /dev/disk4: 32 GB, 'SANDISK USB', not mounted — confirm?" One confirmation, then e.g. `dd if=<image> of=<device> bs=4M status=progress conv=fsync`. Mounted target or size mismatch → stop.

## 4. Permission changes
- Single files / small in-project changes flow freely.
- `-R` over ~100+ files or outside the project tree: `find <scope> | wc -l` and current mode (`stat -f '%Sp %N'` macOS, `stat -c '%A %n'` Linux); state "chmod -R 755 on <scope>: 312 files, 700 → 755", then run.
- `777`, `+s` (setuid/setgid), ownership changes outside the tree: one-line security callout first.
- Narrowest scope wins: one file over a directory, a directory over `-R`.

## 5. Archives are untrusted
- List first (`tar -tzf <a>.tgz | head -20`, `unzip -l <a>.zip`); state entry count and top-level paths.
- Any absolute path or `..` entry is zip-slip: stop and report, do not extract.
- Extract into a fresh dir: `mkdir extract-<name> && tar -xzf <a>.tgz -C extract-<name>`. Never `-P` / `--absolute-names`; never into `.`, `$HOME` or a populated tree.

## Escalation
Root scopes (2) are the only hard refusals. Anything else the user insists on: state the risk in one line, name the compliant path, proceed on their word.

## Checklist
- [ ] Listed first (`head -20` + `wc -l`), count stated, same expression deletes
- [ ] Variables echoed resolved; target not `/`, `$HOME`, `.`, `..`, a mount point
- [ ] Device identity read back; one confirm before dd/mkfs/erase
- [ ] Big/out-of-tree chmod/chown -R: count + before→after; 777/+s called out
- [ ] Archive listed; no absolute/`..` entries; extracted into `extract-<name>/`
