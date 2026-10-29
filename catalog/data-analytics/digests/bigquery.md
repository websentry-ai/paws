# bigquery digest

BigQuery scan or write: `bq query|mk|rm|cp|load|extract|update`, `dbt run|build`, or a BigQuery MCP query/create/delete. Apply before retrying. Target `<bq-project>` (default: `gcloud config get-value project`; say which); pass `--project_id <bq-project>` where the CLI default differs.

## Cost preview first
- Any non-trivial query (a join, an unsized table, more than a LIMITed peek): `bq query --dry_run --use_legacy_sql=false '<sql>'`.
- State bytes processed and approximate dollars at $6.25/TiB ("~412 GB scanned ≈ $2.50"), then run.
- Above 100 GB: restate and get explicit confirmation — "This query scans ~1.4 TiB (≈ $8.75) — run it?"
- `SELECT *` on a large table: check the schema with `bq show`, prune columns, re-dry-run. The dry run is free; never skip it.

## Scratch expires
- Exploratory output only in a scratch dataset named `scratch_<user>`, created with an expiration:
  `bq mk --dataset --default_table_expiration 604800 <bq-project>:scratch_<user>` (604800 s = 7 days).
- Never remove or extend an expiration to keep a scratch table. Never point a scheduled query, curated view or dbt source at scratch.
- Promotion out of scratch goes through dbt, never a bare `bq cp` into curated.

## Sensitive data stays in its zone
- Before a join, CTAS or export touching a table not inspected this session: `bq show --schema --format=prettyjson <bq-project>:<dataset>.<table>`.
- Check names and policy tags against policy-tagged columns plus obvious PII names (email, phone, ssn, dob, address).
- Would PII land in scratch, an export (`bq extract`) or a less-restricted dataset → refuse and offer: aggregate, hash or exclude the columns, or query in place and return only the answer. Never `bq cp`/`bq load` sensitive columns across zones. Not overridable by a user statement.

## Reproducible transformations
- Anything others consume (shared table, metric, backfill) is a versioned model in the repo's dbt project: `dbt run --select <model>`.
- Ad-hoc write into a curated dataset (any dataset not matching the scratch naming convention) via `--destination_table`, `bq cp` or `bq load`: allowed with one line naming the target ("writing ad-hoc into analytics_core.events_backfill — uncodified") plus a note to codify it as a model.

## Labels and deletes
- Every query/job: `--label owner:<human> --label created-by:ai-agent --label purpose:<slug>`; same on `bq load`, `bq cp`, `bq mk` where supported.
- Never `bq rm` a shared dataset/table unprompted. Before a removal: `bq ls <bq-project>:<dataset>`, narrate contents, state the downstream check (views, scheduled queries, dbt sources), name the object in the confirmation ("delete <bq-project>:analytics.events_2024 — confirm?"), proceed only on explicit yes.
- Expired scratch tables clean themselves up; don't bulk-delete them.

## User overrides a rule
State the risk and the compliant path in one line, then proceed. Still stop for a yes on: above-threshold scans, shared deletes. Zone violations need the sanctioned path.

## Checklist
- [ ] Dry run; bytes + ~$ stated; over 100 GB confirmed; SELECT * pruned
- [ ] Exploratory writes in expiring `scratch_<user>`
- [ ] Schemas checked for PII before unfamiliar joins/exports
- [ ] Curated ad-hoc write stated + codify note
- [ ] Labels set; `bq rm` only with dependents listed + confirmation
