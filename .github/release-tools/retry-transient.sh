#!/usr/bin/env bash
# Preserve the fork's registry/API retry policy around the split publish steps.
# Compilation, configuration, and authentication failures still fail immediately.
set -uo pipefail

if (( $# == 0 )); then
  echo 'usage: retry-transient.sh command [args...]' >&2
  exit 2
fi

transient='secondary rate limit|toomanyrequests|too many requests|HTTP status code 5[0-9][0-9]|internal server error|bad gateway|service unavailable|gateway time-?out|unexpected EOF|connection reset by peer|TLS handshake timeout|i/o timeout'
log=$(mktemp "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/release-retry.XXXXXX") || exit 1
trap 'rm -f "$log"' EXIT
max_attempts=3
delay=180

for ((attempt = 1; attempt <= max_attempts; attempt++)); do
  echo "::group::Release command attempt $attempt/$max_attempts"
  "$@" 2>&1 | tee "$log"
  status=${PIPESTATUS[0]}
  echo '::endgroup::'
  if (( status == 0 )); then exit 0; fi
  if (( attempt == max_attempts )); then
    echo "::error::Release command failed after $max_attempts attempts (exit $status)"
    exit "$status"
  fi
  if ! grep -Eiq "$transient" "$log"; then
    echo "::error::Non-transient release failure (exit $status); not retrying"
    exit "$status"
  fi
  echo "::warning::Transient registry/API failure; retrying in ${delay}s"
  sleep "$delay"
  delay=$((delay * 2))
done
