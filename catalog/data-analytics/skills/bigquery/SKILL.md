---
name: bigquery
description: 'Mandatory guardrails for every BigQuery operation that scans or writes (bq query, mk, rm, cp, load, extract, update; dbt run/build) via the bq CLI or BigQuery MCP servers. Enforces the data-analytics baseline with BigQuery mechanics: dry-run cost preview, expiring scratch datasets, PII zone checks, narrated curated writes, and job labels. Triggers PROACTIVELY on any bq query or write — do NOT wait for the user to invoke it.'
---

# BigQuery

BigQuery-specific mechanics for the data-analytics baseline (A1–A5). Apply automatically on every query and write — never skip. All commands target {{BQ_PROJECT}}; pass `--project_id {{BQ_PROJECT}}` wherever the CLI default differs.

## Cost preview (implements A1)

Before any non-trivial query (a join, a table you haven't sized, anything beyond a LIMITed peek), dry-run it, state the cost in one line, and run:

    bq query --dry_run --use_legacy_sql=false '<sql>'

Read the "bytes processed" figure, state it, and convert to approximate dollars at $6.25/TiB on-demand ("~412 GB scanned ≈ $2.50"). Above {{SCAN_CONFIRM_THRESHOLD}}: restate and get explicit confirmation before running — "This query scans ~1.4 TiB (≈ $8.75) — run it?" `SELECT *` on a large table → check the schema with `bq show`, prune to the needed columns, then re-dry-run. The dry run itself is free — never skip it to save a step.

## Scratch datasets (implements A2)

Exploratory results land only in a scratch dataset named per {{SCRATCH_NAMING}}, created with a default expiration so it cleans itself up:

    bq mk --dataset --default_table_expiration {{SCRATCH_TTL_SECONDS}} {{BQ_PROJECT}}:{{SCRATCH_NAMING}}

({{SCRATCH_TTL_SECONDS}} seconds = {{SCRATCH_TTL}}.) Never remove or extend an expiration to make a scratch table permanent; never point a scheduled query, curated view, or dbt source at a scratch table. Promotion out of scratch goes through the dbt path (A4), never a bare `bq cp` into curated.

## Sensitive data stays in its zone (implements A3)

Before any join, CTAS, or export involving a table you haven't inspected this session:

    bq show --schema --format=prettyjson {{BQ_PROJECT}}:<dataset>.<table>

Check column names and policy tags against {{SENSITIVITY_TIERS}}. If the result would materialize PII/regulated columns into scratch, an export (`bq extract`), or a less-restricted dataset → refuse, and offer the sanctioned path: aggregate first, hash or exclude the sensitive columns, or query in place and return only the answer. Never `bq cp` or `bq load` sensitive columns across a zone boundary.

## Reproducible transformations (implements A4)

The paved road for anything others will consume — a table others read, a metric, a backfill — is a versioned model in {{TRANSFORM_TOOL}}:

    dbt run --select <model>

An ad-hoc write into a curated dataset ({{CURATED_DATASETS}}) via `bq query --destination_table`, `bq cp`, or `bq load` is allowed with a one-line statement naming the target ("writing ad-hoc into analytics_core.events_backfill — uncodified"), plus a follow-up note to codify it as a model in the repo. Refuse only when the write violates A3 (sensitive columns crossing zones).

## Attribution and deletes (implements A5)

- Every query carries labels — owner is a human, agent work marked per {{AGENT_ATTRIBUTION_CONVENTION}}:

      bq query --label owner:<human> --label created-by:ai-agent --use_legacy_sql=false '<sql>'

  Add `--label purpose:<short-slug>` describing the task. Same labels on `bq load`, `bq cp`, and `bq mk` via their `--label` flags where supported.
- **Never run `bq rm` on a shared dataset or table unprompted.** Before any confirmed removal, list and narrate what is inside:

      bq ls {{BQ_PROJECT}}:<dataset>

  State the downstream check performed (views, scheduled queries, dbt sources referencing it), name the object in the confirmation ("delete {{BQ_PROJECT}}:analytics.events_2024 — confirm?"), and proceed only on explicit confirmation. Scratch tables past their TTL expire on their own — don't bulk-delete them manually.

## Escalation

Rule vs user intent: state the risk and the compliant path in one line, then proceed as asked — the override is audit-logged. Only above-threshold scans, shared deletes, and A3 zone violations still stop for a yes (A3 is not overridable by statement — it needs the sanctioned path).

## Pre-flight checklist (before every gated call)

- [ ] Dry run done; bytes + approx $ stated; above {{SCAN_CONFIRM_THRESHOLD}} confirmed; SELECT * pruned (A1)
- [ ] Exploratory writes land in an expiring scratch dataset per {{SCRATCH_NAMING}} (A2)
- [ ] Schemas checked for PII before unfamiliar joins/exports; no sensitive column leaves its zone (A3)
- [ ] Curated ad-hoc write stated in one line + codify note; dbt remains the paved road (A4)
- [ ] Labels set (owner human, created-by:ai-agent, purpose); bq rm only with dependents listed + confirmation (A5)

## Category baseline: Data Analytics

These rules apply to **every** warehouse-touching tool. An instance skill (BigQuery, Snowflake, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. A warehouse bills by the byte and remembers every copy — but exploration is the job, not a risk. Queries flow with a stated cost; only above-threshold scans, shared deletes, and PII crossing zones stop the work.

### Rule A1: Cost is previewed, not discovered

Every non-trivial query gets a cost/scan estimate before it runs, using the instance skill's exact dry-run mechanics:

1. Run the estimate, state the bytes/credits in one line, and execute.
2. Above {{SCAN_CONFIRM_THRESHOLD}} → restate the estimate and get explicit user confirmation first.
3. `SELECT *` on a large table → prune to the needed columns first, then re-estimate.

### Rule A2: Scratch work expires

Exploratory output lands only in scratch, and scratch cleans itself up:

- Scratch datasets/tables follow the org naming convention ({{SCRATCH_NAMING}}) and carry a default expiration ({{SCRATCH_TTL}}).
- Nothing permanent lands in scratch: no scheduled job, curated view, or pipeline source may reference a scratch table.
- Promotion out of scratch goes through the versioned transform path (A4), never a copy.

### Rule A3: Sensitive data stays in its zone

PII and regulated columns ({{SENSITIVITY_TIERS}}) never move somewhere less restricted:

- Never copy sensitive columns into scratch, exports, or less-restricted datasets.
- Check the schema/tags of any table you haven't inspected this session before joining or materializing from it.
- A join or export that would materialize PII outside its zone is **refused**, with the sanctioned path offered: aggregate, hash, exclude the columns, or query in place.

### Rule A4: Transformations are reproducible

Exploration is ad-hoc; anything others will consume should be versioned:

- The paved road for any transformation beyond exploration is a versioned artifact — a dbt model or saved query in VCS ({{TRANSFORM_TOOL}}).
- An ad-hoc write to a curated dataset is allowed with a one-line statement naming the target, plus a follow-up note to codify it as a versioned model. Refusal applies only when the write violates A3 (sensitive data crossing zones).

### Rule A5: Jobs are attributed

- Every query and job carries the org's labels: owner (a human), purpose, and the agent-attribution convention ({{AGENT_ATTRIBUTION_CONVENTION}}) so platform teams can distinguish agent workloads in billing and audit.
- Shared datasets/tables are never deleted without explicit confirmation **and** a stated downstream check (views, scheduled jobs, pipeline sources that reference them).

### Pre-flight checklist (run before every warehouse operation)

- [ ] Cost estimated and stated; above-threshold scans confirmed; SELECT * pruned (A1)
- [ ] Exploratory writes go to expiring scratch; no sensitive column crosses zones (A2/A3)
- [ ] Curated ad-hoc writes stated in one line with a codify note left (A4)
- [ ] Labels set; no shared deletes without confirmation + downstream check (A5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<bq-project>`: The CLI's default project (`gcloud config get-value project`). Use it and say which project you targeted.
