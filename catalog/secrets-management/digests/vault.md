# vault digest

Vault operation: `kv get/put/patch/delete`, `kv metadata delete`, `token create`, `policy write`, `lease revoke`. Commands target the environment's existing `VAULT_ADDR` (never set it or print a token to configure it); app secrets live under `secret/`. Apply before retrying.

## Never echo a value
- Never bare `vault kv get <path>`: the printed value is itself an exposure that forces rotation.
- Select one `-field` and pipe it into the consumer:
  - `vault kv get -field=api_key secret/<service>/<name> | ./deploy --key-stdin`
  - `DB_PASSWORD="$(vault kv get -field=password secret/<service>/db)" ./migrate` (single child process, never exported or echoed)
  - File only if required, permissions first: `install -m 0600 /dev/null ./tls.key && vault kv get -field=key secret/<service>/tls > ./tls.key`
- "Show me the secret": `vault kv metadata get secret/<service>/<name>`; report version, created/updated time only. Value needed: `| pbcopy` (or `xclip -selection clipboard`) and say it's on the clipboard.
- Never put a value in narration, error text, commits or tracked files.

## Writes from stdin, never argv
- `openssl rand -base64 32 | vault kv put secret/<service>/db password=-`
- From a file: 0600 file, `vault kv put secret/<service>/api @creds.json && rm creds.json`.
- Secrets live in this Vault instance only: never code, tracked `.env`, tickets, chat, CI config.
- Secret found outside the store: rotate (below), then make the Vault copy canonical.

## Exposure means rotation (do it unasked)
- `openssl rand -base64 32 | vault kv patch secret/<service>/db password=-`
- `vault kv metadata put -custom-metadata=rotation_reason="exposed in <where>, <date>" secret/<service>/db`
- Leased/dynamic secret: `vault lease revoke <lease_id>`.
- Then state which secret, where it leaked, that the old version is dead. Clearing history or force-pushing is not remediation.

## Tokens: narrow policy, explicit TTL
- `vault token create -policy=<narrowest-that-works> -ttl=1h`. Never `-policy=root`, never omit `-ttl`.
- No narrow policy: write one scoped to exact paths (`path "secret/data/<service>/*" { capabilities = ["read"] }`) rather than borrowing a broader one.
- Prefer dynamic secrets: `vault read -field=password database/creds/<role> | <consumer>`.

## Stated access
- Production path: one line first ("Reading secret/payments/stripe-api-key to configure the webhook relay."), then read. No confirmation.
- Dev/sandbox mounts flow freely (piping still applies).
- Bulk enumeration/export (recursive list into reads, dumps, migration scripts) needs explicit confirmed justification, and still follows no-echo mechanics.

## Escalation
"Just print the password": say a printed value is an exposure that forces rotation, offer `| pbcopy` or metadata, then follow the user's explicit call and rotate if burned. The agent never puts a value into the transcript on its own.

## Baseline (all secrets stores)
- Values pipe into consumers; only metadata is ever shown.
- One sanctioned store; nothing "temporarily" elsewhere.
- Exposure = immediate rotation, stated and referenced.
- Least scope, explicit TTL (1h), dynamic over static, no root tokens.
- Prod reads stated; bulk export only on confirmed justification.

## Checklist
- [ ] `-field` piped; no bare `kv get`; "show" = metadata
- [ ] Writes via `field=-` or 0600 file, never argv
- [ ] Exposures rotated and stated
- [ ] `token create` with narrow `-policy` and `-ttl=1h`; never root
- [ ] Prod read stated; bulk export justified and confirmed
