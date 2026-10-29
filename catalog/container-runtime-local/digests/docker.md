# docker digest

Mutating docker/podman commands on a local machine: run, create, build, exec, push, compose up/down, rm, rmi, volume and system prune, network create. Commands target the CLI's current context (`docker context show`). Posture: one-line narration, then proceed; confirm only protected or non-obvious volume data. Cluster commands (kubectl, helm) are out of scope.

## V1: privilege is stated
Each of these gets a one-line why, then proceeds on the user's say-so:
- `--privileged`: disables all isolation; the container is root on the host.
- Any runtime socket mount (`-v /var/run/docker.sock:...`): say "a socket mount IS root on the host" — the container can start privileged siblings.
- `--cap-add <CAP>`, `--security-opt seccomp=unconfined` / `apparmor=unconfined`: wider kernel attack surface.
Legitimate needs exist (NET_ADMIN for a VPN, dind for CI); the bar is a stated need.

## V2: ports bind to loopback
- Default: `docker run -p 127.0.0.1:8080:80 <image>`. Policy: 127.0.0.1 unless exposure is explicitly intended.
- Bare `-p 8080:80` binds 0.0.0.0 (whole local network). Rewrite to loopback unless exposure was intended; if intended, note it in one line and run.

## V3: image provenance
- Pin tags for anything persistent; never `:latest` in compose files, scripts or docs. Compose: pin by digest (`image: nginx@sha256:<digest>`).
- Docker Hub official images and the org's own registry: no ceremony. Anything else, first contact: `docker history --no-trunc <image>` (look for `curl | sh`, added users, baked credentials) and `docker inspect <image>` (`Config.Entrypoint`, `Config.Cmd`, `Config.Env`). State findings, then run.

## V4: mounts are scoped
- Every `-v` / `--mount` states host path and mode; `:ro` unless writes are needed: `docker run -v "$PWD":/work:ro <image>`.
- rw mount of `$HOME`, `~/.ssh`, `~/.aws`, `~/.config` or `/`: state what the container gains ("your SSH and cloud credentials"), then proceed on the user's say-so.

## V5: cleanup previews blast radius
1. `docker system df`; state the reclaim; then `docker system prune` (never `--volumes` as a ride-along).
2. Volumes are data. Before `docker volume rm`, `docker volume prune`, `docker system prune --volumes` or `docker compose down -v`: `docker volume ls`, state candidates by name. Confirm volumes on the protected list (`<protected-volumes>`; none registered by default, so confirm each removal individually) or holding non-obvious data; obvious scratch flows with the list stated.
3. `docker rm -f` on a running container: state what is being killed, then proceed.

## Compose
Before `docker compose up`, read the compose file and note in one line: `privileged: true`, `cap_add`, `security_opt`, docker.sock volumes (V1); `ports:` without `127.0.0.1:` (V2); unpinned/`:latest` images (V3); HOME/credential bind mounts (V4).

## Escalation
State the risk in one line ("a socket mount is root on this machine") and proceed on the user's override, noting it is recorded.

## Checklist
- [ ] Privilege / socket / caps / security-opt stated first (V1)
- [ ] Ports as `127.0.0.1:host:container`; intended 0.0.0.0 noted (V2)
- [ ] Tags pinned; unfamiliar images inspected, findings stated (V3)
- [ ] Mounts state path + mode; rw HOME/credential/root stated (V4)
- [ ] `docker system df` before prune; volumes listed by name; protected confirmed; compose file checked (V5)
