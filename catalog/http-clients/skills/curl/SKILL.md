---
name: curl
description: 'Mandatory guardrails for every curl invocation that authenticates, disables verification, writes remote state, or fetches code — plus wget-shaped equivalents. Enforces the http-clients baseline with curl mechanics: env-var auth headers, --cacert for internal CAs, the download-inspect-execute flow, and checksum verification. Triggers PROACTIVELY on any gated curl/wget shape — do NOT wait for the user to invoke it.'
---

# curl

curl-specific mechanics for the http-clients baseline (Q1–Q5). Apply automatically on every gated curl invocation — never skip. Unless a rule below says otherwise, start every command from the sensible-defaults bundle:

    curl {{CURL_DEFAULT_FLAGS}} <url>

`-f` fails on HTTP errors instead of saving an error page, `--proto '=https'` refuses plaintext (including after redirects), `--max-time` bounds hangs.

## Credentials (implements Q1)

Auth rides a header whose value is expanded by the shell from the environment (or `~/.netrc` via `-n`, per {{CREDENTIALS_SOURCE}}):

    curl -H "Authorization: Bearer $API_TOKEN" https://api.example.com/v1/resource

`-u user:pass` and `https://user:pass@host/…` are forbidden: both put the literal credential into shell history and the process table, where any local process can read it. The header form keeps the secret in the environment; only the variable name appears in argv. Token not in the environment → ask where the org keeps it; never paste it inline.

## TLS (implements Q2)

`-k` / `--insecure` is forbidden. For internal services signed by the org's private CA, trust the CA instead:

    curl --cacert {{INTERNAL_CA_PATH}} https://internal.example.com/health

A certificate error against a public host is a finding — report it, do not bypass it. User insists on `-k` → state the risk in one line (accepts any certificate, including an attacker's) and proceed on their word — the override lands in the audit log.

## Fetch-and-execute (implements Q3)

Never `curl <url> | bash` (or `| sh`) in one step — that shape stays refused, per {{PIPE_TO_SHELL_POLICY}}. The replacement flow needs no approval gate:

    # 1. Download to an explicit path
    curl -fsSLo /tmp/install.sh https://example.com/install.sh

    # 2. Inspect the whole script
    less /tmp/install.sh

    # 3. Summarize in one line (installs X to Y, modifies PATH, fetches Z) — then run:
    bash /tmp/install.sh

If the script fetches further remote scripts, say so — that is the real payload. Never skip step 2 because the source "looks official".

## Writes (implements Q4)

Narration is for **third-party and external services only** — localhost, dev, staging, and the org's internal APIs flow freely; do not narrate every API call. Before any `-X POST/PUT/PATCH/DELETE` (or implicit POST via `-d`/`--data`/`-F`) to a third-party host, state method, host, and a payload summary, then send:

    # "POST to api.example.com/v1/webhooks — registering a webhook pointing at <target>"
    curl -fsSL -X POST https://api.example.com/v1/webhooks \
      -H "Authorization: Bearer $API_TOKEN" -H "Content-Type: application/json" \
      --data '{"url": "<target>", "events": ["push"]}'

The one confirm: DELETE and destructive POSTs against production data. A POST that times out is never blindly re-run — GET the resource (or list endpoint) to check whether the first attempt landed.

## Downloads (implements Q5)

Explicit output path always (`-o <path>` or `-O` with the directory stated). When the publisher posts a checksum, verify before use, per {{CHECKSUM_POLICY}}:

    curl -fsSLO https://example.com/tool-1.2.0.tar.gz && echo "<sha256>  tool-1.2.0.tar.gz" | shasum -a 256 -c

No published checksum → one line that verification was skipped, and move on. Downloaded executables are inspected (`file`, then read scripts per Q3) before any `chmod +x`.

## Siblings

wget, httpie, xh, and aria2 map onto the same Q1–Q5 mechanics (e.g. wget's `--no-check-certificate` ≡ `-k`, `--post-data` ≡ `-d`). Until their instance skills land, apply this skill's rules to them directly; the paws binding already catches the wget shapes.

## Escalation

"Just pipe it to bash": the one-step pipe stays refused — but download → inspect → one-line summary → run does the same job in seconds with no approval ceremony, so offer that and go.
Other conflicts: state the risk in one line and proceed on the user's word — noting the override is recorded.

## Pre-flight checklist (before every gated curl/wget call)

- [ ] Auth via `-H "Authorization: …"` with env-sourced value; no `-u`, no user:pass@ URL
- [ ] No `-k`/`--insecure`; internal CAs via `--cacert {{INTERNAL_CA_PATH}}`
- [ ] No `| bash`/`| sh`; download → inspect → one-line summary → run
- [ ] Non-GET to third-party hosts narrated; internal/dev flows freely; no blind POST retries
- [ ] Downloads: explicit `-o` path; checksum verified when published, one line when not

## Category baseline: HTTP Clients

These rules apply to **every** shell HTTP client. An instance skill (curl, wget, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. Ad-hoc HTTP is the agent's widest egress channel: a token in argv leaks through shell history and the process table, a `-k` flag silently accepts a man-in-the-middle, and a piped installer executes code nobody read.

### Rule Q1: Credentials never ride argv or URLs

Secrets passed on the command line are visible to shell history, the process table (`ps`), and terminal telemetry:

1. Never embed credentials in URLs (`https://user:pass@host/…`) and never paste literal tokens into flags.
2. Authenticate via headers whose values come from the environment, a config file, or `.netrc` — per {{CREDENTIALS_SOURCE}}.
3. If a credential is only available as a literal string, stop and ask where the org keeps it; never inline it "just this once".

### Rule Q2: TLS verification is never disabled

Flags that skip certificate verification (`-k`, `--insecure`, `--no-check-certificate`, and equivalents) are forbidden by default:

- Never add them to make an error go away. A verification failure is a finding, not an obstacle.
- Internal services signed by a private CA are handled by **trusting the CA** ({{INTERNAL_CA_PATH}}) via the instance skill's CA-file mechanics — not by disabling verification.
- If the user explicitly insists, state the risk in one line ("this request will accept any certificate, including an attacker's") and proceed on their word — the insistence is the override; no second round-trip.

### Rule Q3: Piping downloads to a shell is a supply-chain event

Never fetch and execute remote code in one step (`curl … | bash` and every variant) — the one-step pipe stays refused. Per {{PIPE_TO_SHELL_POLICY}}, the flow that replaces it moves freely:

1. Download the script to an explicit local path.
2. Inspect it — read the actual content, not just the first screen.
3. Summarize what it does in one line (installs what, modifies what, fetches what next) — then run it from the local file. The inspection is the gate; no approval ceremony on top.

A truncated or interrupted pipe executes a partial script; a compromised URL executes an attacker's. Neither is recoverable after the fact.

### Rule Q4: Non-GET is narrated

Narration applies to **third-party and external services only** — localhost, dev, staging, and the org's internal APIs flow freely; a rule that fires on every API call teaches everyone to ignore it:

- Before a POST/PUT/PATCH/DELETE to a third-party or external service, state the method, the host, and a one-line summary of the payload ("POST to api.example.com — creating a webhook pointing at …") — then send.
- DELETE and destructive POSTs against production data get explicit same-session confirmation; everything else proceeds on the narration.
- Non-idempotent requests (POST) are never blindly retried on timeout — check whether the first attempt landed before resending.

### Rule Q5: Downloads land deliberately

- Every download gets an explicit output path — never let the tool derive a filename into the current directory unstated.
- When the publisher provides a checksum or signature, verify it before use per {{CHECKSUM_POLICY}}; when absent, one line saying so is enough.
- A downloaded file is never made executable (`chmod +x`) or run without first being inspected per Q3.
- Large or long-running transfers state size and destination up front.

### Pre-flight checklist (run before every HTTP request)

- [ ] No credentials in argv or URL; auth sourced per {{CREDENTIALS_SOURCE}} (Q1)
- [ ] TLS verification on; internal CAs via {{INTERNAL_CA_PATH}}, never -k (Q2)
- [ ] No fetch-and-execute in one step; download → inspect → one-line summary → run (Q3)
- [ ] Non-GET to third-party/external hosts narrated; internal/dev flows freely (Q4)
- [ ] Downloads: explicit output path; checksum verified when published, one line when not (Q5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<internal-ca-path>`: No default; public CAs only. Without a configured CA bundle, a certificate error on an internal host is reported, not bypassed.
