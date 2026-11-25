---
name: postgres
description: 'Mandatory guardrails for every PostgreSQL write operation (UPDATE, DELETE, INSERT, DDL, TRUNCATE, dump/restore) via psql, pg_dump, pg_restore, or a postgres MCP server''s SQL-executing tools. Enforces the databases baseline with psql mechanics: count-then-mutate, explicit transactions with a verify step, migrations for prod DDL, replica routing, and pgpass/service-file credentials. Triggers PROACTIVELY on any write-shaped psql invocation — do NOT wait for the user to invoke it.'
---

# PostgreSQL (psql)

Postgres-specific mechanics for the databases baseline (B1–B5). Apply automatically on every write — never skip. Primary surface is psql; the same rules govern pg_dump/pg_restore and any MCP tool that executes SQL against Postgres. Posture: state what the operation touches in one line, then run it — confirmation only for prod DROP/TRUNCATE.

## Count before you mutate (implements B1)

Before any UPDATE or DELETE, count with the identical WHERE clause, state the number, and proceed — the count is the gate, not a prompt for permission:

    psql -c "SELECT count(*) FROM orders WHERE status = 'stale' AND updated_at < now() - interval '90 days'"
    # state: "This UPDATE will touch 1,204 rows." — then reuse the WHERE verbatim:
    psql -c "UPDATE orders SET status = 'archived' WHERE status = 'stale' AND updated_at < now() - interval '90 days'"

The WHERE clause is copied, not re-typed — a paraphrased clause counts one set and mutates another. A surprising count (0, or 10x expectation) stops the operation. DROP/TRUNCATE on {{PROD_ENVIRONMENTS}} additionally require a stated backup and explicit confirmation restating the object; on dev databases they run with the count and a one-line statement:

    pg_dump -t orders -Fc -f orders_pre_drop.dump "$DB_URL"
    # "Dropping table orders (1.2M rows); dump at ./orders_pre_drop.dump — confirm?"

## Transactions by default (implements B2)

Multi-statement writes run in one explicit transaction with a verify SELECT before COMMIT. State the abort path first: "If the verify looks wrong, I run ROLLBACK; instead of COMMIT;."

    psql "$DB_URL" <<'SQL'
    BEGIN;
    UPDATE accounts SET plan = 'team' WHERE org_id = 42;
    DELETE FROM invites WHERE org_id = 42 AND accepted = false;
    -- verify before commit — output must be shown and sane:
    SELECT plan, count(*) FROM accounts WHERE org_id = 42 GROUP BY plan;
    COMMIT;
    SQL

Never issue the writes as separate autocommitted `psql -c` calls, and never COMMIT without having shown the verify output. If user input is needed mid-transaction, ROLLBACK and restart rather than holding locks.

## Schema changes are migrations (implements B3)

Hand-typed DDL against {{PROD_ENVIRONMENTS}}: state in one line that a migration file is the path, offer to write it in the same reply, and proceed on the user's explicit override (recorded):

    # Rather than: psql -c "ALTER TABLE users ADD COLUMN last_seen_at timestamptz"
    # Via {{MIGRATION_TOOL}} (example shown: alembic):
    alembic revision -m "add users.last_seen_at"
    # write the DDL into the generated versioned file, commit to VCS, apply through the normal pipeline

Response shape: "That's a schema change — I'll write it as a {{MIGRATION_TOOL}} migration so it's versioned and reversible. Here's the file." Dev/scratch databases take direct DDL freely, stated as such in one line.

## Reads route to the replica (implements B4)

Recommendations, stated once per session — not per-query ceremony. Route analytical and exploratory queries to {{READ_REPLICA}}:

    psql "$REPLICA_URL" -c "SELECT ..."

- A SELECT without LIMIT on a large table gets `LIMIT 1000` appended, with a one-line note.
- Long or unknown-cost queries: enable `\timing` and bound them —

      psql "$REPLICA_URL" -c "SET statement_timeout = '{{STATEMENT_TIMEOUT}}'; SELECT ..."

- One-line scope note before an expensive launch: "this scans ~40M rows; replica, {{STATEMENT_TIMEOUT}} timeout."

## Connection hygiene (implements B5)

Credentials come only from {{CREDENTIALS_SOURCE}} — a `PGPASSFILE`/`~/.pgpass` entry or a `pg_service.conf` service:

    PGSERVICE={{PGSERVICE_PROD}} psql -c "..."          # sanctioned
    psql "postgres://app:S3cret@db-prod:5432/app"       # never — password lands in shell history and ps output

- Never pass a password inline in a URI or `PGPASSWORD=... ` prefix on the command line.
- Never set `sslmode=disable` (or drop `sslrootcert`) to silence a TLS error — fix the cert path instead.
- Opening a connection to a {{PROD_ENVIRONMENTS}} host gets one line: "Connecting to db-prod-1 (production) for the archived-orders update."

## Escalation

Rule vs user intent ("skip the count, just delete them"): state the risk in one line ("an uncounted DELETE stakes every row the WHERE happens to match") and proceed on the user's override — noting it's recorded. Confirmation is reserved for prod DROP/TRUNCATE.

## Pre-flight checklist (before every Postgres write)

- [ ] `SELECT count(*)` run with the identical WHERE; number stated; prod DROP/TRUNCATE backed up and confirmed
- [ ] Multi-statement write inside one BEGIN…COMMIT with a shown verify SELECT; ROLLBACK path stated
- [ ] Prod DDL as a {{MIGRATION_TOOL}} migration file (offered, not refused); dev DDL direct and stated
- [ ] Credentials via pgpass/PGSERVICE; no inline passwords; TLS intact; prod connection stated

## Category baseline: Databases

These rules apply to **every** direct-database tool. An instance skill (postgres, mysql, …) adds engine-specific mechanics on top; it may tighten these rules but never loosen them. A database has no undo button: an UPDATE without a count is a bet, and the stake is every row the WHERE clause happens to match. The posture is narrate-and-proceed — state what the operation touches in one line, then run it; confirmation is reserved for irreversible destruction of shared or production data.

### Rule B1: Count before you mutate

Every UPDATE or DELETE is preceded by a count using the **exact same WHERE clause**:

1. Run `SELECT count(*) FROM <table> WHERE <clause>` with the clause copied verbatim — not re-typed, not paraphrased.
2. State the count in one line — "This DELETE will remove 1,204 rows." — and proceed. The count is the gate; no second confirmation after it.
3. A surprising count (zero, or an order of magnitude off the user's expectation) → **stop and show the query**; do not execute.
4. DROP and TRUNCATE on {{PROD_ENVIRONMENTS}} are the exception: state the backup (what was taken, where, when) and get explicit same-session confirmation restating the object. On dev and scratch databases they flow with the count and a one-line statement.

### Rule B2: Transactions by default

Any write of more than one statement runs inside an explicit transaction:

- Open with BEGIN (or the engine's equivalent); never rely on autocommit for multi-statement writes.
- Between the last write and COMMIT, run a verification SELECT and show its output — COMMIT only after the verify looks right.
- State the abort path **before** starting: "If the verify looks wrong, I ROLLBACK instead of COMMIT."
- Never leave a transaction open across a pause for user input on the same connection — locks held are an outage in waiting.

### Rule B3: Schema changes are migrations, not sessions

DDL (CREATE/ALTER/DROP on schema objects) on {{PROD_ENVIRONMENTS}} goes through the org's migration tool ({{MIGRATION_TOOL}}) as a versioned file committed to VCS:

- Hand-typed prod DDL: state in one line that a migration file is the path, offer to write it in the same reply, and proceed only on the user's explicit override (which is recorded).
- Dev and scratch databases take direct DDL freely — one line ("applying directly — dev database, not versioned") and go.
- A migration that cannot be expressed in {{MIGRATION_TOOL}} is an escalation, not an exception.

### Rule B4: Read where reads belong

Recommendations, stated once per session — not per-query ceremony:

- Analytical and exploratory queries belong on {{READ_REPLICA}} rather than the primary; route them there and say so once.
- A query without LIMIT on a large table gets one added, with a one-line note ("added LIMIT 1000 — remove it deliberately if you need the full set").
- Long-running queries get a timeout and a one-line scope note before launch.

### Rule B5: Connection hygiene

- Credentials come only from the sanctioned source ({{CREDENTIALS_SOURCE}}) — never typed, pasted, or read from chat.
- **Never inline a password in the command line** (`postgres://user:pass@`, `-p'secret'`) — it lands in shell history and `ps` output for every user on the host.
- Never disable TLS (`sslmode=disable` and equivalents) to make a connection error go away.
- Opening a connection to a {{PROD_ENVIRONMENTS}} host gets a one-line statement: host, database, operation.

### Pre-flight checklist (run before every database write)

- [ ] Count run with the identical WHERE clause; number stated; prod DROP/TRUNCATE backed up and confirmed (B1)
- [ ] Multi-statement write wrapped in BEGIN with a shown verify SELECT; ROLLBACK path stated (B2)
- [ ] Prod DDL as a migration file (offered, not refused); dev DDL direct and stated (B3)
- [ ] Credentials from the sanctioned source; no inline passwords; TLS on; prod connection stated (B5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<migration-tool>`: The repo's existing migration framework, detected from the codebase (alembic, prisma, rails, drizzle-kit, ...). None found → ask the user.
- `<prod-environments>`: Hosts (and services) whose name matches `*prod*`. Treat those as production and say so.
- `<prod-pgservice>`: Any `pg_service.conf` service whose name contains `prod`.
- `<read-replica>`: The connection in the `REPLICA_URL` env var. If it is unset, run reads on the primary and say so.
