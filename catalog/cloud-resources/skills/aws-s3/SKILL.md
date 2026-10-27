---
name: aws-s3
description: 'Mandatory guardrails for creating or modifying S3 buckets via AWS CLI, SDK calls in Bash, or AWS MCP servers. Enforces the cloud-resources baseline with S3 mechanics: Block Public Access always on, default encryption, mandatory org tags, TTL tagging, and lifecycle hygiene. Triggers PROACTIVELY on any `aws s3api create-bucket`, `aws s3 mb`, or bucket policy/ACL change — do NOT wait for the user to invoke it.'
---

# AWS S3

S3-specific mechanics for the cloud-resources baseline (C1–C5). Apply automatically on every bucket create or policy change — never skip.

## Bucket creation (implements C1, C2, C3)

Every new bucket is created in this exact sequence — one step per call, verify each:

1. **Create** with the org naming convention: `{{BUCKET_NAME_PREFIX}}-<purpose>-<env>` in region {{DEFAULT_REGION}}.

```bash
aws s3api create-bucket --bucket {{BUCKET_NAME_PREFIX}}-<purpose>-<env> \
  --region {{DEFAULT_REGION}} \
  --create-bucket-configuration LocationConstraint={{DEFAULT_REGION}}
```

2. **Block ALL public access — immediately, before any objects land:**

```bash
aws s3api put-public-access-block --bucket <name> \
  --public-access-block-configuration \
  BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
```

3. **Default encryption** ({{ENCRYPTION_MODE}} — SSE-KMS with {{KMS_KEY_ALIAS}} unless the org config says SSE-S3):

```bash
aws s3api put-bucket-encryption --bucket <name> \
  --server-side-encryption-configuration '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"aws:kms","KMSMasterKeyID":"{{KMS_KEY_ALIAS}}"}}]}'
```

4. **Tag with the full org set** (C2) — owner, team, env, cost center, TTL, `created-by=ai-agent`:

```bash
aws s3api put-bucket-tagging --bucket <name> --tagging 'TagSet=[
  {Key={{TAG_OWNER_KEY}},Value=<requesting-human>},
  {Key={{TAG_TEAM_KEY}},Value={{DEFAULT_TEAM}}},
  {Key={{TAG_ENV_KEY}},Value=<env>},
  {Key={{TAG_COSTCENTER_KEY}},Value={{DEFAULT_COST_CENTER}}},
  {Key={{TAG_TTL_KEY}},Value=<today + {{DEFAULT_TTL_DAYS}}d, or permanent>},
  {Key=created-by,Value=ai-agent}]'
```

5. **Versioning** on for anything holding real data ({{VERSIONING_DEFAULT}}):

```bash
aws s3api put-bucket-versioning --bucket <name> --versioning-configuration Status=Enabled
```

6. **Verify** — read back public-access-block, encryption, and tags; report the three results in one line and fix anything that didn't take.

## Public exposure (implements C1, C4)

An S3 bucket is **never** made public by this workflow. If the user genuinely needs public objects (static site, public downloads):

1. Say: "S3-level public access is disallowed by org policy; the paved road is {{PUBLIC_CONTENT_PATH}} (e.g. CloudFront + OAC in front of a private bucket)."
2. Only if the user explicitly confirms after that, restate the exposure and proceed — the audit log records the override.

None of `put-bucket-acl --acl public-read`, a bucket policy with `"Principal": "*"` Allow, or loosening the public-access-block runs without that explicit confirmation — exposure is confirm-class (C1).

## Policy and ACL changes (implements C5)

Before `put-bucket-policy`, `put-bucket-acl`, or `delete-public-access-block` on an existing bucket: fetch the current policy, show a before/after diff of principals and actions, then apply. Confirmation is needed only when the diff makes anything public (C1) — a reviewed diff otherwise proceeds.

## Pre-flight checklist (before every S3 mutation)

- [ ] Bucket name follows `{{BUCKET_NAME_PREFIX}}-<purpose>-<env>`
- [ ] Public access block + default encryption ({{ENCRYPTION_MODE}}) applied in the creation sequence
- [ ] Full org tag set in place, TTL included
- [ ] No `Principal:*` Allow, public ACL, or PAB loosening without explicit confirmation (C1)
- [ ] Post-create verification read back, reported in one line, gaps fixed

## Category baseline: Cloud Resources

These rules apply to **every** action that creates, modifies, or destroys cloud resources, regardless of provider or tool. Instance skills add provider-specific mechanics.

### Rule C1: Least exposure by default

Every resource is created with the most restrictive access posture the provider supports — private, non-public, least-privilege. Making anything publicly reachable requires the user to say so explicitly in this session, and you restate the exposure before executing: "This makes X publicly readable — confirm?"

### Rule C2: Mandatory tags on everything

Every resource that supports tags/labels gets the org's tag set at creation time, not later:

| Tag | Value |
|-----|-------|
| `{{TAG_OWNER_KEY}}` | the requesting human's identifier |
| `{{TAG_TEAM_KEY}}` | {{DEFAULT_TEAM}} |
| `{{TAG_ENV_KEY}}` | one of {{ENV_VALUES}} |
| `{{TAG_COSTCENTER_KEY}}` | {{DEFAULT_COST_CENTER}} |
| `{{TAG_TTL_KEY}}` | ISO date the resource can be reclaimed (see C3) |
| `created-by` | `ai-agent` — always, so FinOps can distinguish agent-created resources |

If a required tag value is unknown, use the org default and state it in one line; cost center defaults to `{{DEFAULT_COST_CENTER}}` outside prod; ask for it only when env is prod.

### Rule C3: Everything has a termination date

Unless the user states the resource is permanent, set `{{TAG_TTL_KEY}}` to **{{DEFAULT_TTL_DAYS}} days** from creation. The org's reaper honors this tag. For "permanent" resources, set the value to `permanent` and tell the user that requires an owner and cost-center tag to survive review.

### Rule C4: Prefer the org's paved road

If the org has a sanctioned provisioning path for this resource type — a Terraform module, an internal CLI wrapper, a service catalog entry ({{PAVED_ROAD_REGISTRY}}) — use it instead of raw provider calls. Raw CLI is the fallback, not the default.

### Rule C5: Destructive and expensive actions are narrated

Before creating anything above {{COST_THRESHOLD}} estimated monthly cost, state a one-line cost note ("~$X/month for Y") and proceed. Before any resize or delete of an ephemeral/dev resource, state what happens and proceed. Confirmation is reserved for two cases: deleting a data-bearing resource, and making anything public (C1).

### Pre-flight checklist (run before every provisioning call)

- [ ] Access posture is private/least-privilege; public exposure explicitly confirmed (C1)
- [ ] Full org tag set + TTL present in the creation call itself (C2, C3)
- [ ] Paved-road path checked before raw CLI (C4)
- [ ] Cost above threshold noted in one line; data-bearing deletes and exposure confirmed (C5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<encryption-mode>`: SSE-S3 (AES256) — SSE-KMS when the org names a key. Use it and say which value you used.
- `<kms-key-alias>`: No default — SSE-S3 applies until a key is configured. Skip the step that needs it and say so.
- `<org-prefix>`: The org's short name, lowercased. Use it and say which value you used.
- `<paved-road-registry>`: No default — raw CLI allowed, stated. Skip the step that needs it and say so.
- `<region>`: The AWS CLI's configured region (aws configure get region). Use it and say which value you used.
- `<team>`: The requesting human's team. Use it and say which value you used (state the team used).
- `unassigned`: The `cost-center` tag value when the user gives none. Outside prod, use it and say so in one line; when env is prod, ask the user instead (billing attribution on prod spend is not worth guessing).
