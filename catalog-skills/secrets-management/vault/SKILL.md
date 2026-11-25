---
name: vault
description: 'Mandatory guardrails for every HashiCorp Vault operation (kv get, put, patch, delete, token create, policy write, lease revoke) via the vault CLI. Enforces the secrets-management baseline with Vault mechanics: -field piped retrieval, stdin writes, patch-and-burn rotation, narrow-policy short-TTL tokens, dynamic creds, and stated-purpose prod reads. Triggers PROACTIVELY on any `vault kv`, `vault token`, or `vault lease` invocation — do NOT wait for the user to invoke it.'
---

# HashiCorp Vault

Vault-specific mechanics for the secrets-management baseline (X1–X5). Apply automatically on every vault invocation — never skip. All commands target {{VAULT_ADDR}}; application secrets live under {{KV_MOUNT}}.

## No-echo retrieval (implements X1)

**Never run bare `vault kv get <path>`** — it prints the value into the transcript, which is itself an exposure (X3). Retrieval always selects one `-field` and pipes it directly into the consumer:

    # Pipe straight into the consuming command
    vault kv get -field=api_key {{KV_MOUNT}}/<service>/<name> | ./deploy --key-stdin

    # Env var scoped to a single child process — never exported, never echoed
    DB_PASSWORD="$(vault kv get -field=password {{KV_MOUNT}}/<service>/db)" ./migrate

    # File only when the consumer demands one — permissions BEFORE content
    install -m 0600 /dev/null ./tls.key && vault kv get -field=key {{KV_MOUNT}}/<service>/tls > ./tls.key

When the user asks to see a secret, show metadata only and say so:

    vault kv metadata get {{KV_MOUNT}}/<service>/<name>

Report version, created_time, updated_time — never the value. If the value itself is needed: `vault kv get -field=<f> <path> | pbcopy` (or `xclip -selection clipboard`) and state "it's on your clipboard".

## Writes from stdin, never argv (implements X2)

A value typed on argv lands in shell history and `ps` output — an exposure before the write even completes. The `-` sentinel reads the field from stdin:

    # Generate and store without the value ever appearing anywhere
    openssl rand -base64 32 | vault kv put {{KV_MOUNT}}/<service>/db password=-

    # Value already in a file: 0600 file, removed after the write
    vault kv put {{KV_MOUNT}}/<service>/api @creds.json && rm creds.json

Secrets live in {{SECRET_STORE}} and nowhere else: never write one into code, a VCS-tracked `.env`, a ticket, or CI config. Discovering one outside the store (hardcoded, committed, logged) → rotate per X3 below, then make the Vault copy canonical.

## Rotation on exposure (implements X3)

An exposed secret is burned the moment it touches history, a log, a commit, or chat. Rotate first, explain second:

    # New value in, old version burned — generated and piped, never printed
    openssl rand -base64 32 | vault kv patch {{KV_MOUNT}}/<service>/db password=-

    # Record why the version rolled, for auditors
    vault kv metadata put -custom-metadata=rotation_reason="exposed in <where>, <date>" {{KV_MOUNT}}/<service>/db

    # Leased/dynamic secret exposed → revoke the lease instead
    vault lease revoke <lease_id>

Then state it: which secret, where it leaked, that the prior version is dead. Clearing history, force-pushing, or editing the log is never remediation — copies survive deletion.

## Tokens: narrow policy, explicit TTL (implements X4)

    vault token create -policy=<narrowest-that-works> -ttl={{DEFAULT_TOKEN_TTL}}

Never `-policy=root`, never omit `-ttl` for routine work. If no policy is narrow enough, write one scoped to the exact paths rather than borrowing a broader one:

    vault policy write <service>-ro - <<'EOF'
    path "{{KV_MOUNT}}/data/<service>/*" { capabilities = ["read"] }
    EOF

Prefer dynamic secrets over static ones wherever a secrets engine exists — issued per use, auto-revoked at lease end:

    vault read -field=password database/creds/<role> | <consumer>   # -field still applies (X1)

## Stated access (implements X5)

Before any read under a production path: one line first, then the command — "Reading {{KV_MOUNT}}/payments/stripe-api-key to configure the webhook relay." That statement is the whole gate; no confirmation. Reads under dev/sandbox mounts flow freely — no statement needed (X1 piping still applies). Bulk enumeration — `vault kv list` walked recursively into reads, export or migration scripts, dump-to-file — is exfiltration-shaped and needs explicit confirmed justification; on confirmation it still follows X1 mechanics (piped, 0600, nothing printed).

## Escalation

Rule vs user intent ("just print the password"): state in one line that a printed value is an exposure forcing rotation, offer `| pbcopy` or metadata — then follow the user's explicit call, rotating per X3 if the value was burned.
Exception: X1 never bends silently — a value never lands in the transcript by the agent's own hand.

## Pre-flight checklist (before every gated vault call)

- [ ] Retrieval uses `-field` piped into the consumer; no bare `vault kv get`; "show" = metadata only (X1)
- [ ] Write reads the value from stdin (`field=-`) or a 0600 file, never argv; no copy outside Vault (X2)
- [ ] Any exposure this session already rotated (`kv patch` / `lease revoke`), stated, and referenced (X3)
- [ ] `token create` carries `-policy=<narrow>` and `-ttl={{DEFAULT_TOKEN_TTL}}`; never root; dynamic creds preferred (X4)
- [ ] Prod read: secret and purpose stated first (dev mounts flow freely); bulk enumeration/export only on confirmed justification (X5)

## Category baseline: Secrets Management

These rules apply to **every** secrets-store tool. An instance skill (Vault, AWS Secrets Manager, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. A secret is the one kind of data that is destroyed by being seen: every surface a value touches — stdout, a log line, shell history, a chat transcript — is a copy no one can revoke, and the only remediation is rotation.

### Rule X1: Never echo a secret

Secret values never appear in stdout, logs, narration, files outside the store, or this conversation:

1. Retrieval pipes directly into the consumer: an env var scoped to a single child process, a file created with 0600 permissions before content lands, or process substitution — never a bare read that prints.
2. Never interpolate a secret value into a command you narrate, an error message, a commit, or any file tracked by VCS.
3. When a value must be shown, show only its metadata (name, version, created/updated time, lease) — never the value itself.

### Rule X2: One sanctioned store

Secrets live in {{SECRET_STORE}} and nowhere else:

- Never committed to code, a `.env` file in VCS, a ticket, chat, a wiki, or a CI log — not even "temporarily".
- New secrets are created in the store first and consumed from it; there is no "I'll move it in later".
- Discovering a secret outside the store (hardcoded, in a tracked env file, in a log) is an exposure — trigger X3, then make the store's copy canonical.

### Rule X3: Exposure means rotation

A secret that reached shell history, a log, a commit (pushed or not), or a chat is burned:

1. Rotate it immediately via the instance skill's rotation mechanics — do not wait to be asked.
2. State that you rotated it and reference the exposure: which secret, where it leaked, when.
3. Deleting the trace (clearing history, force-pushing, editing the log) is **not** remediation — copies survive deletion; rotation is the only fix.

### Rule X4: Least scope, always expiring

- Every token or credential minted carries the narrowest policy that works and an explicit TTL ({{DEFAULT_TOKEN_TTL}}).
- No root/admin tokens for routine work — if no narrow policy exists, create one; never reach for a broader one.
- Prefer dynamic/periodic credentials the store issues and revokes over static values that live forever.

### Rule X5: Access is stated

- Before reading a production secret, say which secret and why — one line, then read. No confirmation needed; the statement is the control.
- Dev/sandbox-mount secrets flow freely — no statement required (X1 no-echo mechanics still apply).
- Bulk enumeration or export of secrets (recursive listing fed into reads, dump-to-file, migration scripts) requires explicit, confirmed justification — exfiltration-shaped — and even then follows X1 mechanics.

### Pre-flight checklist (run before every secrets-store operation)

- [ ] Value pipes into its consumer; nothing printed, logged, or narrated (X1)
- [ ] Secret exists only in {{SECRET_STORE}}; nothing written to code, VCS, chat, or CI (X2)
- [ ] Any exposure rotated immediately, stated, and referenced (X3)
- [ ] Minted credentials: narrowest policy, explicit TTL, dynamic over static (X4)
- [ ] Prod read: secret and purpose stated (dev mounts flow freely); bulk export only on confirmed justification (X5)
