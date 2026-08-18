#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

check_application_security_opt() {
  file=$1
  count=$(
    awk '
      $0 == "  sub2api:" {
        in_application = 1
        next
      }
      in_application && $0 ~ /^  [A-Za-z0-9_-]+:$/ {
        in_application = 0
      }
      in_application && $0 == "    security_opt:" {
        in_security_opt = 1
        next
      }
      in_application && in_security_opt && $0 == "      - no-new-privileges:true" {
        count++
      }
      END { print count + 0 }
    ' "$file"
  )

  if [ "$count" -ne 1 ]; then
    printf '%s must enable no-new-privileges exactly once for the sub2api service\n' "$file" >&2
    exit 1
  fi
}

check_required_admin_password() {
  file=$1
  expected='      - ADMIN_PASSWORD=${ADMIN_PASSWORD:?ADMIN_PASSWORD is required}'
  count=$(grep -Fxc "$expected" "$file" || true)

  if [ "$count" -ne 1 ]; then
    printf '%s must require ADMIN_PASSWORD exactly once for the sub2api service\n' "$file" >&2
    exit 1
  fi
}

check_loopback_bind_default() {
  file=$1
  expected='      - "${BIND_HOST:-127.0.0.1}:${SERVER_PORT:-8080}:8080"'
  count=$(grep -Fxc -- "$expected" "$file" || true)

  if [ "$count" -ne 1 ]; then
    printf '%s must default BIND_HOST to 127.0.0.1 (loopback)\n' "$file" >&2
    exit 1
  fi
}

for compose_file in \
  deploy/docker-compose.yml \
  deploy/docker-compose.local.yml \
  deploy/docker-compose.standalone.yml \
  deploy/docker-compose.dev.yml
do
  check_application_security_opt "$compose_file"
  check_required_admin_password "$compose_file"
  if [ "$compose_file" != "deploy/docker-compose.dev.yml" ]; then
    check_loopback_bind_default "$compose_file"
  fi
done

printf 'docker compose security test passed\n'
