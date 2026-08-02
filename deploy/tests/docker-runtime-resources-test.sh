#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

fail() {
  printf 'docker runtime resources test failed: %s\n' "$1" >&2
  exit 1
}

assert_line() {
  file=$1
  line=$2
  grep -Fqx "$line" "$file" || fail "$file is missing: $line"
}

# Every docker image build must ship the entry, whatever the goreleaser layout
# happens to be. Checking each extra_files block beats a hardcoded occurrence
# count, which silently goes stale when entries are merged or split (legacy
# `dockers:` had one per platform/registry; `dockers_v2:` covers them all).
assert_extra_files_include() {
  file=$1
  entry=$2
  summary=$(awk -v want="      - $entry" '
    /^    extra_files:$/ { inblock = 1; blocks++; found = 0; next }
    inblock && $0 == want { found = 1; next }
    inblock && /^      - / { next }
    inblock { inblock = 0; if (!found) missing++ }
    END { if (inblock && !found) missing++; print blocks + 0, missing + 0 }
  ' "$file")
  blocks=${summary% *}
  missing=${summary#* }
  [ "$blocks" -gt 0 ] || fail "$file has no extra_files block"
  [ "$missing" -eq 0 ] || fail "$file: $missing of $blocks extra_files blocks omit '$entry'"
}

test -s backend/resources/model-pricing/model_prices_and_context_window.json || \
  fail 'fallback pricing data is missing or empty'

assert_line Dockerfile.goreleaser 'COPY --chown=sub2api:sub2api backend/resources /app/resources'
assert_line deploy/Dockerfile 'COPY --from=backend-builder --chown=sub2api:sub2api /app/backend/resources /app/resources'
assert_extra_files_include .goreleaser.yaml backend/resources
assert_extra_files_include .goreleaser.simple.yaml backend/resources

printf 'docker runtime resources test passed\n'
