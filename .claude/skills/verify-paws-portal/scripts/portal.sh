#!/usr/bin/env bash
# Launch, check, drive and tear down a private paws portal stack for verification:
# paws-backend (test mode, its own database), paws-website (next dev, dev-token sign-in), and a scratch
# laptop for the paws CLI built from this checkout. Kills only the processes it started.
#
#   portal.sh up                 start the stack (fresh DB, fresh scratch laptop, new evidence dir)
#   portal.sh doctor             read-only: is this instance ours and healthy?
#   portal.sh run [feature ...]  drive features (default: the browser/laptop suite; "all" adds claude-gate)
#   portal.sh down               stop what `up` started, drop the DB, delete scratch; evidence stays
#   portal.sh up --real          backend with real PropelAuth (ENVIRONMENT=dev), website without the dev token
set -euo pipefail

SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO="$(git -C "$SKILL_DIR" rev-parse --show-toplevel)"
PAWS_BACKEND_DIR="${PAWS_BACKEND_DIR:-$REPO/../paws-backend}"
PAWS_WEBSITE_DIR="${PAWS_WEBSITE_DIR:-$REPO/../paws-website}"
VERIFY_PG_ADMIN_URL="${VERIFY_PG_ADMIN_URL:-postgres://postgres:postgres@localhost:5432/postgres}"
VERIFY_DB="${VERIFY_DB:-paws_verify}"
VERIFY_API_PORT="${VERIFY_API_PORT:-8013}"
VERIFY_SITE_PORT="${VERIFY_SITE_PORT:-3135}"
VERIFY_CHROME="${VERIFY_CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
STATE="${VERIFY_STATE:-${TMPDIR:-/tmp}/paws-verify-portal}"
PW_DIR="${VERIFY_PW_DIR:-${TMPDIR:-/tmp}/paws-verify-playwright}" # cache, survives `down`
PW_VERSION=1.63.0
if [[ -z "${VERIFY_PYTHON:-}" ]]; then
  if [[ -x "$PAWS_BACKEND_DIR/.venv/bin/python" ]]; then VERIFY_PYTHON="$PAWS_BACKEND_DIR/.venv/bin/python"; else VERIFY_PYTHON=python3; fi
fi
DB_URL="${VERIFY_PG_ADMIN_URL%/*}/$VERIFY_DB"

die() { echo "portal.sh: $*" >&2; exit 1; }
killtree() { local p=$1 c; for c in $(pgrep -P "$p" 2>/dev/null); do killtree "$c"; done; kill "$p" 2>/dev/null || true; }
alive() { [[ -f "$STATE/$1.pid" ]] && kill -0 "$(cat "$STATE/$1.pid")" 2>/dev/null; }
# The listener on a port must be the recorded process or one of its descendants.
owns_port() {
  local pid; pid=$(cat "$STATE/$1.pid"); local l
  for l in $(lsof -t -iTCP:"$2" -sTCP:LISTEN 2>/dev/null); do
    local p=$l
    while [[ -n "$p" && "$p" != 1 ]]; do [[ "$p" == "$pid" ]] && return 0; p=$(ps -o ppid= -p "$p" | tr -d ' '); done
  done
  return 1
}
wait_for() { # url, seconds, what
  local i; for ((i = 0; i < $2; i++)); do curl -sf -o /dev/null "$1" && return 0; sleep 1; done
  die "$3 did not answer at $1 within $2 s (see logs in $ARTIFACTS)"
}

up() {
  local real=0; [[ "${1:-}" == "--real" ]] && real=1
  { alive backend || alive site; } && die "already up ($STATE); run \`portal.sh down\` first"
  for port in "$VERIFY_API_PORT" "$VERIFY_SITE_PORT"; do
    lsof -t -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1 && die "port $port is taken by another process; set VERIFY_API_PORT / VERIFY_SITE_PORT"
  done
  [[ -f "$PAWS_BACKEND_DIR/manage.py" ]] || die "no paws-backend at $PAWS_BACKEND_DIR (set PAWS_BACKEND_DIR)"
  [[ -f "$PAWS_WEBSITE_DIR/package.json" ]] || die "no paws-website at $PAWS_WEBSITE_DIR (set PAWS_WEBSITE_DIR)"
  [[ -d "$PAWS_WEBSITE_DIR/node_modules" ]] || die "run npm ci in $PAWS_WEBSITE_DIR first"
  [[ -x "$VERIFY_CHROME" ]] || die "no Chrome at $VERIFY_CHROME (set VERIFY_CHROME)"

  rm -rf "$STATE"; mkdir -p "$STATE/run/fakebin"
  local run_id; run_id="$(date +%Y%m%d-%H%M%S)"
  ARTIFACTS="$REPO/.verify-artifacts/portal-$run_id"; mkdir -p "$ARTIFACTS"

  echo "building paws"; (cd "$REPO" && go build -o "$STATE/paws" ./cmd/paws)
  if [[ ! -f "$PW_DIR/node_modules/playwright-core/index.mjs" ]]; then
    echo "installing playwright-core@$PW_VERSION into $PW_DIR"; mkdir -p "$PW_DIR"
    npm install --silent --no-save --prefix "$PW_DIR" "playwright-core@$PW_VERSION" >/dev/null
  fi
  # `paws login` opens the browser with `open`; this recorder stands in so no real browser opens.
  printf '#!/bin/sh\necho "$@" >> "%s/run/opened.txt"\n' "$STATE" > "$STATE/run/fakebin/open"; chmod +x "$STATE/run/fakebin/open"

  echo "database $VERIFY_DB (fresh)"
  PGOPTIONS='-c client_min_messages=warning' psql "$VERIFY_PG_ADMIN_URL" -qc "drop database if exists $VERIFY_DB" -c "create database $VERIFY_DB" >/dev/null

  local site="http://localhost:$VERIFY_SITE_PORT" api="http://127.0.0.1:$VERIFY_API_PORT"
  local benv=(ENVIRONMENT=test DEBUG=true)
  if ((real)); then
    [[ -f "${PAWS_PROPELAUTH_ENV:-}" ]] || die "--real needs PAWS_PROPELAUTH_ENV=<file with PAWS_PROPELAUTH_URL and PAWS_PROPELAUTH_API_KEY>"
    benv=(ENVIRONMENT=dev)
  fi
  echo "backend on :$VERIFY_API_PORT ($([[ $real == 1 ]] && echo 'real PropelAuth' || echo 'test mode'))"
  (
    cd "$PAWS_BACKEND_DIR"
    if ((real)); then set -a; . "$PAWS_PROPELAUTH_ENV"; set +a; fi
    export "${benv[@]}" DATABASE_URL="$DB_URL" PAWS_PORTAL_URL="$site" PAWS_PORTAL_ORIGINS="$site"
    "$VERIFY_PYTHON" manage.py migrate --noinput >"$ARTIFACTS/migrate.log" 2>&1
    nohup "$VERIFY_PYTHON" manage.py runserver "127.0.0.1:$VERIFY_API_PORT" --noreload >"$ARTIFACTS/backend.log" 2>&1 &
    echo $! >"$STATE/backend.pid"
  )
  echo "website on :$VERIFY_SITE_PORT"
  (
    cd "$PAWS_WEBSITE_DIR"
    export NEXT_PUBLIC_PAWS_API="http://localhost:$VERIFY_API_PORT" NEXT_PUBLIC_POSTHOG_KEY=
    ((real)) || export NEXT_PUBLIC_PAWS_DEV_TOKEN=verify
    nohup npx next dev -p "$VERIFY_SITE_PORT" >"$ARTIFACTS/site.log" 2>&1 &
    echo $! >"$STATE/site.pid"
  )
  wait_for "$api/healthz" 60 backend
  wait_for "$site/portal" 180 website

  cat >"$STATE/env.sh" <<EOF
export VERIFY_DIR="$STATE/run" VERIFY_API="$api" VERIFY_SITE="$site" VERIFY_DB_URL="$DB_URL"
export PAWS_BIN="$STATE/paws" VERIFY_ARTIFACTS="$ARTIFACTS" VERIFY_BACKEND_DIR="$PAWS_BACKEND_DIR"
export VERIFY_PYTHON="$VERIFY_PYTHON" VERIFY_CHROME="$VERIFY_CHROME" VERIFY_USER=verify VERIFY_REAL=$real
export PLAYWRIGHT_CORE="$PW_DIR/node_modules/playwright-core/index.mjs"
EOF
  echo "up. evidence: $ARTIFACTS"
  doctor
}

doctor() {
  [[ -f "$STATE/env.sh" ]] || die "not up (no $STATE/env.sh); run \`portal.sh up\`"
  # shellcheck disable=SC1091
  . "$STATE/env.sh"
  local ok=1
  report() { if eval "$2"; then echo "  ok    $1"; else echo "  FAIL  $1"; ok=0; fi; }
  report "backend process alive (pid $(cat "$STATE/backend.pid"))" "alive backend"
  report "backend owns :$VERIFY_API_PORT" "owns_port backend $VERIFY_API_PORT"
  report "website process alive (pid $(cat "$STATE/site.pid"))" "alive site"
  report "website owns :$VERIFY_SITE_PORT" "owns_port site $VERIFY_SITE_PORT"
  report "backend /healthz says database ok" "curl -sf $VERIFY_API/healthz | grep -q '\"database\": \"ok\"'"
  report "website serves /portal" "curl -sf -o /dev/null $VERIFY_SITE/portal"
  report "paws binary built from $REPO" "\"$PAWS_BIN\" version >/dev/null"
  report "database $VERIFY_DB reachable" "psql \"$VERIFY_DB_URL\" -tAc 'select 1' >/dev/null"
  report "playwright-core present" "[[ -f \"$PLAYWRIGHT_CORE\" ]]"
  local tok="${VERIFY_CLAUDE_TOKEN_FILE:-$HOME/.config/paws-verify/claude-token}"
  if [[ -s "$tok" ]]; then echo "  ok    claude token file present (claude-gate runs isolated)"
  else echo "  note  no claude token at $tok (claude-gate uses the fallback; see SKILL.md)"; fi
  echo "  mode  $([[ "$VERIFY_REAL" == 1 ]] && echo real || echo test)   evidence: $VERIFY_ARTIFACTS"
  ((ok)) || exit 1
}

run() {
  [[ -f "$STATE/env.sh" ]] || die "not up; run \`portal.sh up\` first"
  # shellcheck disable=SC1091
  . "$STATE/env.sh"
  [[ "$VERIFY_REAL" == 1 ]] && die "the scripted features need test mode (they sign in with the dev token); follow the real-mode recipe in SKILL.md"
  node "$SKILL_DIR/scripts/run.mjs" "$@"
}

down() {
  local evidence=""
  if [[ -f "$STATE/env.sh" ]]; then
    # shellcheck disable=SC1091
    . "$STATE/env.sh"; evidence="$VERIFY_ARTIFACTS"
  fi
  for svc in site backend; do
    if alive "$svc"; then killtree "$(cat "$STATE/$svc.pid")"; echo "stopped $svc"; fi
  done
  psql "$VERIFY_PG_ADMIN_URL" -qc "drop database if exists $VERIFY_DB" >/dev/null 2>&1 && echo "dropped $VERIFY_DB" || true
  rm -rf "$STATE"
  [[ -n "$evidence" ]] && { [[ -d "$evidence" ]] && echo "evidence kept: $evidence" || echo "WARNING: evidence missing at $evidence"; }
  return 0
}

case "${1:-}" in
  up) shift; up "$@" ;;
  doctor) doctor ;;
  run) shift; run "$@" ;;
  down) down ;;
  *) sed -n '2,10p' "$0"; exit 2 ;;
esac
