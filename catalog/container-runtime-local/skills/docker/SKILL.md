---
name: docker
description: 'Mandatory guardrails for every mutating Docker operation (run, create, build, exec, compose up/down, rm, rmi, prune, volume ops) on a local machine. Enforces the container-runtime-local baseline with Docker mechanics: one-line privilege narration, loopback port publishing, inspection of unfamiliar images, scoped bind mounts, and volume-safe cleanup. Triggers PROACTIVELY on any docker or docker compose mutating command — do NOT wait for the user to invoke it.'
---

# Docker

Docker-specific mechanics for the container-runtime-local baseline (V1–V5). Apply automatically on every mutating docker command — never skip. Local machines only; kubectl/helm are governed by the containers-orchestration category. All commands target {{DOCKER_CONTEXT}}. Posture: one-line narration and proceed; confirmation only for protected or non-obvious volume data.

## Privilege (implements V1)

Each of these shapes gets its one-line why stated, then proceeds on the user's say-so:

- `--privileged` — disables all isolation; the container is root on the host.
- `-v /var/run/docker.sock:...` (any socket bind mount) — **a socket mount IS root on the host**: anything inside the container can tell the daemon to start a privileged sibling. Say it in those words.
- `--cap-add <CAP>` / `--security-opt seccomp=unconfined` (or apparmor=unconfined) — widens the kernel attack surface beyond the container.

Legitimate needs exist (NET_ADMIN for a VPN container, dind for CI testing); the bar is a stated need, not a ban.

## Ports (implements V2)

Default publish form — loopback named explicitly:

    docker run -p 127.0.0.1:8080:80 <image>

Bare `-p 8080:80` binds 0.0.0.0: the service is reachable by every device on the local network. Policy: {{PORT_BIND_POLICY}}. Rewrite bare `-p` to the loopback form when exposure was not explicitly intended; when it was, note the exposure in one line and run — no confirmation.

## Image provenance (implements V3)

- Pin tags for anything persistent — never `:latest` in compose files, scripts, or docs. In compose files pin by digest: `image: nginx@sha256:<digest>`.
- Docker Hub official images and {{SANCTIONED_REGISTRY}} images run without ceremony. First contact with an image outside those, inspect before first run:

      docker history --no-trunc <image>
      docker inspect <image>

  From the history: layer commands — look for `curl | sh` installs, added users, credentials baked into layers. From the inspect output read `Config.Entrypoint`, `Config.Cmd`, and `Config.Env`: what actually executes at start, and whether the environment carries anything odd. State what you found, then run.

## Mounts (implements V4)

Preferred shape — explicit host path, explicit mode, read-only default:

    docker run -v "$PWD":/work:ro <image>

- Every `-v`/`--mount` states host path and ro/rw; use `:ro` unless the task needs writes.
- A rw mount of `$HOME`, `~/.ssh`, `~/.aws`, `~/.config`, or `/`: state what the container gains ("this hands the container your SSH and cloud credentials") in one line, then proceed on the user's say-so.

## Cleanup (implements V5)

    # 1. Read what exists and what a prune would reclaim
    docker system df

    # 2. State the reclaim, then prune — never with --volumes
    docker system prune

Volumes are data. Before `docker volume rm`, `docker volume prune`, or `docker system prune --volumes`, list first:

    docker volume ls

State the candidate volumes by name. Confirmation only for volumes in {{PROTECTED_VOLUMES}} or holding non-obvious data — obvious scratch volumes flow with the list stated. `docker rm -f` on a running container: state what is being killed, then proceed.

## Compose (implements V1–V5)

`docker compose up` inherits every rule. Before up, read the compose file and note in one line anything found: `privileged: true`, `cap_add`, `security_opt`, or a docker.sock volume (V1); `ports:` entries without a `127.0.0.1:` prefix (V2); unpinned or `:latest` images — prefer digests (V3); bind mounts of HOME/credential dirs (V4). `docker compose down -v` deletes the project's volumes — treat it as V5 volume removal: list, flag protected ones, then run.

## Escalation

Rule vs user intent ("just mount the docker socket, the tool needs it"): state the risk in one line ("a socket mount is root on this machine — the container can start privileged siblings") and proceed on the user's override — noting it's recorded.

## Pre-flight checklist (before every mutating docker command)

- [ ] --privileged / socket mount / cap-add / security-opt loosening stated in one line first (V1)
- [ ] Ports published as 127.0.0.1:host:container; intended 0.0.0.0 binds noted in one line (V2)
- [ ] Tag pinned (digest in compose); images outside official/org registry inspected and findings stated (V3)
- [ ] Bind mounts state path and mode; rw HOME/credential/root mounts stated (V4)
- [ ] docker system df read and reclaim stated before prune; volumes listed by name, protected ones confirmed; compose files checked before up (V5)

## Category baseline: Local Container Runtimes

These rules apply to **every** tool that runs, builds, mounts, publishes, or destroys containers on a local machine. An instance skill (Docker, Podman, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. A local runtime daemon is root on the machine it runs on — but these are local, mostly-reversible operations: the posture is one-line narration and proceed, with confirmation reserved for data that can't be rebuilt. Cluster-side rules (kubectl, helm, multi-node blast radius) live in the sibling **containers-orchestration** category — do not apply these rules to cluster commands, or those rules here.

### Rule V1: Privilege is exceptional

Containers run unprivileged by default; every escalation is stated in one line, then proceeds on the user's say-so:

- `--privileged` disables all isolation between container and host — state the need in one line; proceed on the user's override.
- Bind-mounting the runtime socket: **a socket mount IS root on the host** — anything inside the container can instruct the daemon to start a privileged sibling. Say it in those words, state the need, and proceed on the user's override.
- `--cap-add` and `--security-opt` loosening (seccomp/apparmor unconfined) widen the kernel attack surface: same one-line stated need. Legitimate needs exist (NET_ADMIN for a VPN container); the bar is a stated need, not a ban.

### Rule V2: Ports bind narrow

- Publish to loopback by default: {{PORT_BIND_POLICY}}. The default publish form names 127.0.0.1 explicitly.
- A bare port publish binds 0.0.0.0 and exposes the service to every device on the local network. Rewrite to the loopback form when exposure was not explicitly intended; when it was, note the exposure in one line and proceed — no confirmation.

### Rule V3: Images have provenance

- Pinned tags for anything persistent (compose files, scripts, docs) — never `:latest` outside a throwaway interactive run.
- Images from Docker Hub official images or {{SANCTIONED_REGISTRY}} run without ceremony. First contact with an image outside those: read its layer history and configured entrypoint/cmd/env via the instance skill's inspect mechanics before running it — an unfamiliar image is arbitrary code that will execute on this machine.

### Rule V4: Mounts are scoped

- Every bind mount states its host path and mode (ro/rw); read-only unless the task needs writes.
- A rw mount of `$HOME`, credential directories (`~/.ssh`, `~/.aws`, `~/.config`), or `/` hands the container the user's cloud and SSH identity — state what the container gains in one line, then proceed on the user's say-so.

### Rule V5: Cleanup previews blast radius

- Every prune states what it will reclaim before running — disk usage read first via the instance skill's df mechanics — then proceeds.
- **Volumes are data.** Volume removal never rides along with a general prune; list the candidate volumes by name first. Confirmation only for volumes matching protected patterns or holding non-obvious data — scratch volumes flow with the list stated.
- Force-removing a running container states what is being killed, then proceeds.

### Pre-flight checklist (run before every mutating runtime command)

- [ ] Privilege escalation (privileged / socket mount / caps / security-opt) stated in one line first (V1)
- [ ] Ports published to loopback; any intended 0.0.0.0 bind noted in one line (V2)
- [ ] Tags pinned; images outside official/org registry inspected first (V3)
- [ ] Mounts scoped with explicit mode; rw HOME/credential/root mounts stated (V4)
- [ ] Prune reclaim stated first; volumes listed by name; protected/non-obvious volumes confirmed (V5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<protected-volumes>`: None registered, so every volume removal is listed and confirmed individually.
