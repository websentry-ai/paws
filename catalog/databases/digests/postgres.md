# postgres digest

PostgreSQL write: psql with UPDATE/DELETE/INSERT/DDL/TRUNCATE, a psql session on a production host, pg_dump, pg_restore, or an MCP SQL write. Apply before retrying. Posture: state what the operation touches in one line, then run it. Confirmation only for production DROP/TRUNCATE.

`<prod-environments>`: hosts/services whose name matches `*prod*` unless the org configured otherwise.

## Count before UPDATE/DELETE
1. `psql -c "SELECT count(*) FROM <table> WHERE <clause>"` with the WHERE copied verbatim, not re-typed.
2. State it ("This UPDATE will touch 1,204 rows.") and run the write with the identical WHERE. The count is the gate; no extra confirmation.
3. Surprising count (0, or 10x off expectation) → stop and show the query.
4. DROP/TRUNCATE on `<prod-environments>`: take and state a backup (`pg_dump -t <table> -Fc -f <table>_pre_drop.dump "$DB_URL"`), then confirm restating the object: "Dropping table orders (1.2M rows); dump at ./orders_pre_drop.dump — confirm?" On dev: count + one line.

## Transactions by default
- Multi-statement writes go in one heredoc: `BEGIN;` … writes … verify `SELECT` … `COMMIT;`. Never separate autocommitted `psql -c` calls.
- State the abort path first: "If the verify looks wrong, I run ROLLBACK; instead of COMMIT;."
- Show the verify output before COMMIT. Need user input mid-transaction → ROLLBACK and restart; never hold locks across a pause.

## Schema changes are migrations
- Hand-typed DDL against `<prod-environments>`: say a migration file is the path, offer to write it in the same reply ("I'll write it as a <migration-tool> migration so it's versioned and reversible"), proceed only on explicit override (recorded).
- `<migration-tool>`: the repo's existing framework, detected from the codebase (alembic, prisma, rails, drizzle-kit…); none found → ask.
- Dev/scratch databases: direct DDL, stated in one line ("applying directly — dev database, not versioned").

## Reads (once per session, not per query)
- Analytical/exploratory queries on `<read-replica>` (`psql "$REPLICA_URL" …`); if `REPLICA_URL` is unset, use the primary and say so.
- SELECT without LIMIT on a large table → append `LIMIT 1000` with a one-line note.
- Long/unknown-cost queries: `SET statement_timeout = '30s';` and a scope note ("scans ~40M rows; replica, 30s timeout").

## Connection hygiene
- Credentials only from ~/.pgpass or PGSERVICE (`PGSERVICE=<prod-pgservice> psql -c "…"`; default: the service whose name contains `prod`).
- Never inline a password (`postgres://user:pass@…`, `PGPASSWORD=…` prefix); it lands in shell history and `ps`.
- Never `sslmode=disable` or drop `sslrootcert` to silence a TLS error; fix the cert path.
- Connecting to a production host: one line — host, database, operation.

## User overrides a rule
One line of risk ("an uncounted DELETE stakes every row the WHERE happens to match"), proceed on override, say it is recorded.

## Checklist
- [ ] Count with the identical WHERE; number stated; prod DROP/TRUNCATE backed up + confirmed
- [ ] Multi-statement write in one BEGIN…COMMIT with shown verify; ROLLBACK path stated
- [ ] Prod DDL as a migration (offered); dev DDL direct and stated
- [ ] pgpass/PGSERVICE; no inline passwords; TLS intact; prod connection stated
