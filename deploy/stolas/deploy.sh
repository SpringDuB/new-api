#!/usr/bin/env bash
set -euo pipefail
umask 077

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
deploy_dir="$repo_dir/deploy/stolas"
env_file="$deploy_dir/.env"
action="${1:-deploy}"
cd -- "$repo_dir"

case "$action" in
  deploy|update|backup|status|logs) ;;
  *) printf 'Usage: bash deploy/stolas/deploy.sh [deploy|update|backup|status|logs]\n' >&2; exit 1 ;;
esac

for dependency in docker git openssl python3 curl getent ss; do
  if ! command -v "$dependency" >/dev/null 2>&1; then
    printf 'Required command is missing: %s\n' "$dependency" >&2
    exit 1
  fi
done
docker compose version >/dev/null
docker info >/dev/null

if [[ "$action" == update ]]; then
  if [[ "$(git branch --show-current)" != codex/deploy ]]; then
    printf 'Updates require the codex/deploy branch.\n' >&2
    exit 1
  fi
  if [[ -n "$(git status --porcelain)" ]]; then
    printf 'Commit or move server-side repository changes before updating.\n' >&2
    exit 1
  fi
  git pull --ff-only origin codex/deploy
  exec bash "$deploy_dir/deploy.sh" deploy
fi

if [[ ! -e "$env_file" ]]; then
  if [[ "$action" != deploy ]]; then
    printf 'Run bash deploy/stolas/deploy.sh to initialize the deployment first.\n' >&2
    exit 1
  fi
  postgres_password="$(openssl rand -hex 32)"
  redis_password="$(openssl rand -hex 32)"
  session_secret="$(openssl rand -hex 32)"
  crypto_secret="$(openssl rand -hex 32)"
  (
    set -o noclobber
    sed \
      -e "s/^POSTGRES_PASSWORD=$/POSTGRES_PASSWORD=$postgres_password/" \
      -e "s/^REDIS_PASSWORD=$/REDIS_PASSWORD=$redis_password/" \
      -e "s/^SESSION_SECRET=$/SESSION_SECRET=$session_secret/" \
      -e "s/^CRYPTO_SECRET=$/CRYPTO_SECRET=$crypto_secret/" \
      "$deploy_dir/.env.example" > "$env_file"
  )
  printf 'Created server-only secrets in %s\n' "$env_file"
fi
chmod 600 "$env_file"

export DEPLOY_COMMIT="$(git rev-parse --short=12 HEAD)"
export DOMAIN="$(sed -n 's/^DOMAIN=//p' "$env_file")"
compose=(docker compose --env-file "$env_file" -f "$deploy_dir/compose.yml")
"${compose[@]}" config --quiet

backup_database() {
  local backup_dir backup_file
  backup_dir=/var/backups/new-api-stolas
  mkdir -p -m 700 -- "$backup_dir"
  backup_file="$(mktemp "$backup_dir/database-$(date -u +%Y%m%dT%H%M%SZ)-XXXXXX.partial")"
  "${compose[@]}" exec -T postgres pg_dump -U new_api -d new-api -Fc > "$backup_file"
  mv -- "$backup_file" "${backup_file%.partial}.dump"
  printf 'Database backup: %s\n' "${backup_file%.partial}.dump"
}

case "$action" in
  status) exec "${compose[@]}" ps ;;
  logs) exec "${compose[@]}" logs --tail=100 -f new-api caddy ;;
  backup) backup_database; exit 0 ;;
esac

"${compose[@]}" build new-api
"${compose[@]}" up -d --wait --wait-timeout 180 postgres redis
backup_database
"${compose[@]}" up -d --wait --wait-timeout 300 new-api

# Keep the first administrator setup reachable only through the SSH tunnel.
setup_status="$(curl --fail --silent --show-error --max-time 15 http://127.0.0.1:13000/api/setup)"
initialized="$(python3 -c '
import json, sys
response = json.load(sys.stdin)
if response.get("success") is not True or not isinstance(response.get("data"), dict):
    raise SystemExit("Cannot determine whether setup is complete")
print("yes" if response["data"].get("status") is True else "no")
' <<< "$setup_status")"
if [[ "$initialized" != yes ]]; then
  "${compose[@]}" stop caddy
  printf '\nPrivate startup complete. Keep your SSH tunnel open and visit:\n'
  printf '  http://localhost:13000/setup\n'
  printf 'Create the administrator account, then run this command again:\n'
  printf '  bash deploy/stolas/deploy.sh\n'
  exit 0
fi

if ! getent ahostsv4 "$DOMAIN" >/dev/null; then
  printf 'The server cannot resolve an IPv4 address for %s. Check its A record.\n' "$DOMAIN" >&2
  exit 1
fi
caddy_id="$("${compose[@]}" ps -aq caddy)"
if [[ -z "$caddy_id" && -n "$(ss -H -ltn '( sport = :80 or sport = :443 )')" ]]; then
  printf 'Ports 80/443 are occupied. Configure the existing proxy before continuing.\n' >&2
  exit 1
fi
"${compose[@]}" up -d --wait --wait-timeout 120 caddy

for attempt in {1..24}; do
  if curl --fail --silent --max-time 10 "https://$DOMAIN/api/status" >/dev/null; then
    printf '\nHTTPS is responding: https://%s\n' "$DOMAIN"
    "${compose[@]}" ps
    exit 0
  fi
  sleep 5
done
printf 'Containers are running, but HTTPS verification failed. Check DNS, ports 80/443 and Caddy logs.\n' >&2
printf '  bash deploy/stolas/deploy.sh logs\n' >&2
exit 1
