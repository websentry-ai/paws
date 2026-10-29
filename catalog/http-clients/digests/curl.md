# curl digest

curl/wget that authenticates, skips TLS checks, writes remote state, or pipes a download into a shell. Apply before retrying.

Start from: `curl -fsSL --proto '=https' --max-time 60 <url>` (fail on HTTP errors, HTTPS only incl. redirects, bounded time).

## 1. Credentials never ride argv or URLs
- Auth via a header whose value the shell expands from the environment: `curl -H "Authorization: Bearer $API_TOKEN" <url>`, or `~/.netrc` (mode 0600) via `-n`.
- Forbidden: `-u user:pass`, `https://user:pass@host/…`, literal tokens in flags. They land in shell history and `ps`.
- Token not in the environment → ask where the org keeps it. Never inline it "just this once".

## 2. TLS verification stays on
- No `-k` / `--insecure` / wget `--no-check-certificate` to make an error go away. A cert error on a public host is a finding: report it.
- Internal private-CA hosts: `curl --cacert <internal-ca-path> <url>`. No CA configured → report, don't bypass.
- User explicitly insists on `-k`: one line ("accepts any certificate, including an attacker's"), then proceed on their word.

## 3. No fetch-and-execute in one step
`curl … | bash` / `| sh` stays refused. Instead, with no approval ceremony:
1. `curl -fsSLo /tmp/install.sh <url>`
2. Read the whole script (`less /tmp/install.sh`), not just the first screen.
3. One-line summary (installs what where, modifies PATH, fetches what next; say if it pulls more remote scripts), then `bash /tmp/install.sh`.

## 4. Non-GET to third-party hosts is narrated
- localhost, dev, staging and the org's internal APIs flow freely.
- Before `-X POST/PUT/PATCH/DELETE` (or implicit POST via `-d`/`--data`/`-F`) to a third-party/external host: state method, host, one-line payload summary, then send.
- Explicit same-session confirmation only for DELETE and destructive POSTs against production data.
- A timed-out POST is never blindly re-run: GET the resource first to see whether it landed.

## 5. Downloads land deliberately
- Explicit output path always (`-o <path>`, or `-O` with the directory stated).
- Publisher posts a checksum → verify before use: `echo "<sha256>  <file>" | shasum -a 256 -c`. None published → one line saying verification was skipped.
- Inspect downloaded executables (`file`, read scripts) before any `chmod +x` or run.
- Large transfers: state size and destination up front.

## Siblings
wget, httpie, xh, aria2 follow the same rules (`--no-check-certificate` ≡ `-k`, `--post-data` ≡ `-d`).

## Checklist
- [ ] Auth via env-sourced header or .netrc; no `-u`, no user:pass@ URL
- [ ] No `-k`/`--insecure`; internal CAs via `--cacert <internal-ca-path>`
- [ ] No `| bash`/`| sh`; download → inspect → summary → run
- [ ] Non-GET to third-party hosts narrated; no blind POST retries
- [ ] Explicit `-o` path; checksum verified when published, one line when not
