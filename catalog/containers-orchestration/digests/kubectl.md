# kubectl digest

Cluster mutations: kubectl apply, create, delete, edit, patch, scale, drain, cordon, uncordon, rollout, exec, label, annotate; helm install/upgrade/uninstall/rollback; Kubernetes MCP writes. Prod = contexts whose name contains `prod`. Dev/kind/minikube/docker-desktop flow freely with explicit flags. Apply before retrying.

## K1: explicit context and namespace
- Every command carries `--context <ctx> -n <ns>`; never bare `kubectl <verb>`. Resolve first: `kubectl config get-contexts`. Shorthand ("delete that pod") → resolve to context + namespace, then show.
- Prod mutation: restate in plain language, "running against context `<ctx>` (production), namespace `<ns>`", then run. Cluster-scoped verbs (drain, cordon) still name `--context`.

## K2: images pinned, from the sanctioned registry
Before `apply -f m.yaml` to a shared or prod cluster:
- `grep -n 'image:' m.yaml`; `grep -nE 'image:.*:latest' m.yaml` (any hit → repin first); `grep -nc '@sha256:' m.yaml` (0 on prod → flag, prefer digests).
- No `:latest`, untagged or floating tags (`stable`, `main`) outside local throwaway runs.
- Images from the org's private registry (`<sanctioned-registry>`); none configured → ask before pulling public images into prod or shared clusters.

## K3: requests and limits on every container
Before applying to any non-local context (anything but kind-*, minikube, docker-desktop):
`yq -e '.. | select(has("containers")) | .containers[] | select(.resources.limits.cpu == null or .resources.limits.memory == null) | .name' m.yaml` — any name printed → add requests + limits (or conservative estimates, labeled). No yq → `grep -c 'limits:'` must be ≥ container count. Skip only on explicit user override, stated in one line.

## K4: declarative, validated applies
- Long-lived workloads change through manifests in version control and the org's delivery path.
- Prod apply = two steps on the identical file: `kubectl --context <ctx> -n <ns> apply --dry-run=server -f m.yaml`, then `apply -f m.yaml` only after a clean dry-run.
- Imperative `edit` / `patch` / `scale` on prod: one line on why the manifest path is bypassed, run, then a sync note ("live change: replicas 3→5 on deploy/<name>; update <manifest path> to match"). Non-prod: free.
- Rollback with recorded revisions: `rollout history`, `rollout undo deployment/<name> --to-revision=<rev>`, `rollout status`. Never a hand-rebuilt manifest.
- `kubectl exec` into a prod pod: state what the command changes first.

## K5: destructive ops narrated first
- Before `delete`, `rollout restart`, `scale --replicas=0`: preview (`get pods -l app=<x>`), state blast radius in one line ("Deleting 4 pods behind service `<x>`; traffic re-shards"), then run.
- Prod deletes and drains also need explicit same-session confirmation.
- Drain: `cordon <node>` → `get pods -A --field-selector spec.nodeName=<node>` → narrate evictions → `drain <node> --ignore-daemonsets --delete-emptydir-data`.
- `--force` / `--grace-period=0`: never by default; state that in-flight work can be lost; confirm on prod.

## Escalation
State the risk in one line and proceed on the user's override, noting it is recorded. Confirmation only for prod deletes and drains.

## Checklist
- [ ] `--context` and `-n` explicit; prod target restated
- [ ] Manifests grepped: no `:latest`/untagged; sanctioned registry; digests for prod
- [ ] Requests + limits on every container in shared clusters
- [ ] Prod: server dry-run first on the identical file; imperative edits narrated + sync note
- [ ] Blast radius stated; prod deletes/drains confirmed
