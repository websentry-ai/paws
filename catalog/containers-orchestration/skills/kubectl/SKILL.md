---
name: kubectl
description: 'Mandatory guardrails for every kubectl mutating operation (apply, create, delete, edit, patch, scale, drain, cordon, rollout, exec, label/annotate). Enforces the containers-orchestration baseline with kubectl mechanics: explicit --context/-n on every command, server-side dry-run before prod applies, image-pin and limits checks on manifests, one-line blast-radius narration before destructive ops. Triggers PROACTIVELY on any kubectl write — do NOT wait for the user to invoke it.'
---

# kubectl

kubectl-specific mechanics for the containers-orchestration baseline (K1–K5). Apply automatically on every mutating kubectl call — never skip. Posture: dev/kind/minikube contexts flow freely with explicit flags; narration and confirmation ceremony is reserved for {{PROD_CONTEXTS}}.

## Explicit context and namespace (implements K1)

Every command — reads included, when shown to the user — carries `--context <ctx> -n <ns>`. Never run bare `kubectl <verb>`:

    kubectl config get-contexts                       # resolve the target first
    kubectl --context <ctx> -n <ns> get deploy <name>

Before a mutation against {{PROD_CONTEXTS}}, restate the target in plain language — "running against context `<ctx>` (production), namespace `<ns>`" — then run. Dev and local contexts (kind-*, minikube, docker-desktop) need only the flags. Cluster-scoped verbs (drain, cordon) still name `--context` explicitly.

## Image pin check before apply (implements K2)

Before `kubectl apply -f m.yaml` to a shared or prod cluster, check the manifest mechanically:

    grep -n 'image:' m.yaml                           # list every image reference
    grep -nE 'image:.*:latest' m.yaml                 # any hit → repin before applying
    grep -nc '@sha256:' m.yaml                        # 0 on a prod-context manifest → flag, prefer digests

Every image must resolve from {{SANCTIONED_REGISTRY}}; an untagged or `:latest` image is repinned before it lands in a shared cluster — local throwaway runs are exempt. Digest pins (`image@sha256:…`) are the preferred form for prod.

## Limits check before apply (implements K3)

Check every container for `resources.limits` before applying to {{SHARED_CLUSTER_CONTEXTS}}:

    yq -e '.. | select(has("containers")) | .containers[]
           | select(.resources.limits.cpu == null or .resources.limits.memory == null) | .name' m.yaml
    # any name printed → add requests+limits before applying
    # yq unavailable → grep -c 'limits:' m.yaml must be >= container count

A missing limit: add one, or propose conservative estimates labeled as such; proceed without limits only on the user's explicit override, stated in one line.

## Declarative changes, validated applies (implements K4)

Long-lived workloads change through manifests per {{MANIFEST_REPO_POLICY}}. Prod applies are a two-step on the same file:

    kubectl --context <ctx> -n <ns> apply --dry-run=server -f m.yaml
    kubectl --context <ctx> -n <ns> apply -f m.yaml    # identical file, only after clean dry-run

On non-prod contexts the dry-run is recommended, not required, and imperative mutations (`kubectl edit`, `kubectl patch`, `kubectl scale`) flow freely. Against {{PROD_CONTEXTS}} they get one line on why the manifest path is bypassed, then a sync note — "live change: replicas 3→5 on deploy/<name>; update <manifest path> to match." Rollbacks use the recorded revision, never a hand-rebuilt manifest:

    kubectl --context <ctx> -n <ns> rollout history deployment/<name>
    kubectl --context <ctx> -n <ns> rollout undo deployment/<name> --to-revision=<rev>
    kubectl --context <ctx> -n <ns> rollout status deployment/<name>

`kubectl exec` into a prod pod: state what the command changes inside the container before running it.

## Destructive operations narrated first (implements K5)

Before any `kubectl delete`, preview, state the blast radius in one line, and proceed:

    kubectl --context <ctx> -n <ns> get pods -l app=<x>

"Deleting 4 pods behind service `<x>`; traffic re-shards to remaining replicas" — then delete. Same preview-then-state flow for `rollout restart` and `scale --replicas=0`. Deletes and drains against {{PROD_CONTEXTS}} additionally require same-session confirmation. Drain is a three-step:

    kubectl --context <ctx> cordon <node>              # stop new scheduling first
    kubectl --context <ctx> get pods -A --field-selector spec.nodeName=<node>
    # narrate: "<n> pods evicted from <node>, incl. <notable workloads>" — then:
    kubectl --context <ctx> drain <node> --ignore-daemonsets --delete-emptydir-data

`--force` and `--grace-period=0` skip graceful shutdown and can lose in-flight requests: never add them by default; state that in one line before using them, and confirm only on {{PROD_CONTEXTS}}.

## Escalation

Rule vs user intent ("skip the dry-run", "force-delete the stuck pod"): state the risk in one line ("force-deleting skips graceful shutdown; in-flight requests to this pod are dropped") and proceed on the user's override — noting it's recorded. Confirmation is reserved for prod deletes and drains.

## Pre-flight checklist (before every kubectl mutation)

- [ ] `--context` and `-n` explicit; prod target restated in plain language
- [ ] Shared/prod manifests grepped: no `:latest`/untagged images; {{SANCTIONED_REGISTRY}} only; digests for prod
- [ ] Every container in {{SHARED_CLUSTER_CONTEXTS}} has requests + limits (yq check run)
- [ ] Prod applies dry-run first on the identical file; imperative prod edits narrated + sync note
- [ ] Destructive ops: one-line blast radius stated; prod deletes/drains confirmed

## Category baseline: Containers & Orchestration

These rules apply to **every** container- or cluster-mutating tool. An instance skill (kubectl, helm, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. A cluster is shared production infrastructure: a command fired at the wrong context takes down someone else's service, a floating tag makes an outage unreproducible, and a limitless pod starves every neighbor on the node. The posture is narrate-and-proceed — dev, kind, and minikube contexts flow freely; ceremony is reserved for production.

### Rule K1: Context and namespace are explicit

Every command names its target — never rely on the current-context or default-namespace setting:

1. Every invocation states the context and namespace explicitly using the instance skill's exact flag mechanics.
2. Before a mutation against a production context ({{PROD_CONTEXTS}}), restate the target in plain language — "running against `prod-eu`, namespace `payments`" — then run. Dev and local contexts (kind-*, minikube, docker-desktop) need only the explicit flags, no restating.
3. A command inherited from user shorthand ("delete that pod") is resolved to an explicit context + namespace first, then shown.

### Rule K2: Images are pinned and provenanced

- No `:latest` and no floating tags (`stable`, `main`, untagged) in anything beyond a local throwaway experiment.
- Images come from {{SANCTIONED_REGISTRY}} — never pull unvetted public images into shared or production clusters.
- Production workloads prefer digest pins (`@sha256:…`) over tags; a tag can move, a digest cannot.
- Before any apply/deploy to a shared or production cluster, the manifest is mechanically checked for violations per the instance skill.

### Rule K3: Every workload declares resource limits

- Every container spec carries `resources.requests` **and** `resources.limits` for both CPU and memory.
- A missing limit on a shared cluster: add limits (or propose conservative estimates, labeled as such) before applying; proceed without them only on the user's explicit override, stated in one line.

### Rule K4: Declarative over imperative

Long-lived workloads change through manifests in version control, not live edits:

1. Changes go through manifests tracked per {{MANIFEST_REPO_POLICY}} and the org's delivery path (CI/CD, GitOps).
2. Imperative mutations (`edit`, `patch`, `scale` by hand) against production are exceptional: one line on why the paved road is being bypassed, run, then a manifest sync note so the repo and cluster reconverge. On non-prod contexts, imperative edits flow freely.
3. Prod applies validate first: server-side dry-run, then apply the identical file. Elsewhere the dry-run is a recommendation, not a gate.

### Rule K5: Destructive operations narrated first

Delete, drain, cordon, rollout restart, and scale-to-zero state their blast radius in one line **before** running — affected pod count and traffic impact, using the instance skill's preview commands — then proceed:

- Deletes and drains against {{PROD_CONTEXTS}} additionally get explicit same-session confirmation; non-prod flows on the one-line statement.
- `--force`, `--grace-period=0`, and equivalents skip graceful shutdown and can lose in-flight work — state that in one line; confirm only on production.
- Rollbacks use the tool's recorded-revision mechanism, never a hand-rebuilt state.

### Pre-flight checklist (run before every cluster mutation)

- [ ] Context + namespace explicit on the command; prod target restated in plain language (K1)
- [ ] Images pinned, from {{SANCTIONED_REGISTRY}}; no `:latest`; digests for prod (K2)
- [ ] Requests + limits on every container in shared clusters (K3)
- [ ] Prod changes declarative (or imperative edit narrated + sync note); prod applies dry-run first (K4)
- [ ] Destructive ops: one-line blast radius stated; prod deletes/drains confirmed (K5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<sanctioned-registry>`: The org's private registry. None configured: ask before pulling images from public registries into prod or shared clusters.
