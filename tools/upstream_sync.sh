#!/usr/bin/env bash
#
# tools/upstream_sync.sh — merge an upstream release tag into the fork's
# BASE_BRANCH.
#
# Conflicts touching only the fork-preservation whitelist are "invalid
# conflicts" and auto-resolve to OURS; any other conflict is a necessary
# conflict: the merge is aborted, the offending paths are printed one per
# line on stdout, and the script exits 10.
#
# UPSTREAM_URL may be an https URL or a plain filesystem path (the offline
# test harness uses local fixture repos). This script never pushes.
#
# Tag hygiene: the upstream tag is fetched into refs/upstream-sync/<tag> (never
# refs/tags) so upstream releases cannot pollute the fork's own version line.
#
# Sync state is derived from git ancestry, not from STATE_FILE. A maintainer
# who merges upstream/main by hand advances the merge graph without touching
# STATE_FILE, so treating that file as the source of truth makes it drift and
# the engine then re-merges tags the fork already contains. STATE_FILE is kept
# as a human-readable record and a fast path, but ancestry decides.
#
# On exit 0 a single machine-readable line is printed on stdout:
#   sync-result=merged      upstream commits were merged; verify before pushing
#   sync-result=state-only  tag already an ancestor; only STATE_FILE moved
#   sync-result=noop        nothing to do; no commits created
# On exit 10 stdout carries the conflicted paths instead (one per line).
#
# Requires: bash, git. Run from anywhere inside the fork working tree.

set -u
set -o pipefail
# NOTE: no `set -e` — the merge is expected to fail on conflicts and its
# failure must be inspected, not abort the script.

UPSTREAM_URL="${UPSTREAM_URL:-https://github.com/Wei-Shaw/sub2api.git}"
UPSTREAM_TAG="${UPSTREAM_TAG:-}"
BASE_BRANCH="${BASE_BRANCH:-main}"
STATE_FILE="${STATE_FILE:-.github/upstream-sync-tag}"
SYNC_REF=""

# Fork-preservation whitelist — source of truth: AGENTS.md "FORK-SPECIFIC PRESERVATION". Conflicts limited to these files are "invalid conflicts" and auto-resolve to OURS; anything else is a necessary conflict -> abort + exit 10.
WHITELIST=(
  backend/cmd/server/VERSION
  README.md
  README_CN.md
  README_JA.md
  deploy/install.sh
  deploy/docker-deploy.sh
  deploy/APPLE_CONTAINER.md
  frontend/src/i18n/locales/en/fork.ts
  frontend/src/i18n/locales/zh/fork.ts
  frontend/src/composables/useConfirm.ts
  frontend/src/utils/chartTheme.ts
  frontend/tailwind.config.js
  frontend/src/style.css
  backend/internal/service/announcement_broadcast_service.go
)

usage() {
  cat <<'EOF'
tools/upstream_sync.sh — merge an upstream release tag into this fork.

Usage:
  UPSTREAM_TAG=<tag> bash tools/upstream_sync.sh   (run from the fork repo)

Environment:
  UPSTREAM_URL   Upstream remote URL or local filesystem path.
                 (default: https://github.com/Wei-Shaw/sub2api.git)
  UPSTREAM_TAG   Upstream tag to fetch and merge. REQUIRED; empty -> exit 1.
  BASE_BRANCH    Branch to merge into. (default: main)
  STATE_FILE     File recording the last synced tag, relative to repo root.
                 (default: .github/upstream-sync-tag)

Exit codes:
  0   success (merged + state file committed), state-only (tag was already an
      ancestor of HEAD, so only STATE_FILE moved), or no-op (zero new commits).
      The exact outcome is on stdout as sync-result=<merged|state-only|noop>.
  10  necessary conflict outside the fork-preservation whitelist; the merge
      was aborted and the offending paths are printed one per line on stdout
  1   error (usage, environment, fetch, or unexpected git failure; stderr
      carries diagnostics)
EOF
}

log() { printf '%s\n' "$*" >&2; }
err() { printf 'error: %s\n' "$*" >&2; }
# The only thing this script writes to stdout on a 0 exit; the workflow reads
# it to decide whether the verification suite has anything to verify.
result() { printf 'sync-result=%s\n' "$1"; }

is_whitelisted() {
  local p="$1" w
  for w in "${WHITELIST[@]}"; do
    [[ "$p" == "$w" ]] && return 0
  done
  return 1
}

preflight() {
  if [[ "$(git rev-parse --is-inside-work-tree 2>/dev/null || true)" != "true" ]]; then
    err "not inside a git working tree"
    exit 1
  fi

  # Anchor all relative paths (STATE_FILE) at the repo root.
  local top
  top="$(git rev-parse --show-toplevel)"
  cd "$top" || {
    err "cannot cd to repo root $top"
    exit 1
  }

  if [[ "$(git rev-parse --is-shallow-repository 2>/dev/null || echo true)" == "true" ]]; then
    err "shallow repositories are not supported; use a full clone"
    exit 1
  fi

  local current
  current="$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)"
  if [[ "$current" != "$BASE_BRANCH" ]]; then
    log "switching to base branch $BASE_BRANCH (was: ${current:-detached})"
    if ! git checkout "$BASE_BRANCH" >&2; then
      err "could not check out $BASE_BRANCH"
      exit 1
    fi
  fi

  # Identity must already exist (repo-local or global); never set it here.
  if [[ -z "$(git config --get user.name 2>/dev/null || true)" ]] ||
    [[ -z "$(git config --get user.email 2>/dev/null || true)" ]]; then
    err "git identity (user.name / user.email) is not configured; configure it before running"
    exit 1
  fi
}

noop_check() {
  if [[ -f "$STATE_FILE" ]]; then
    local recorded
    recorded="$(tr -d '[:space:]' <"$STATE_FILE")"
    if [[ "$recorded" == "$UPSTREAM_TAG" ]]; then
      log "STATE_FILE ($STATE_FILE) already records $UPSTREAM_TAG; nothing to do."
      result noop
      exit 0
    fi
  fi
}

fetch_tag() {
  # Private namespace, NOT refs/tags: keeps upstream release tags out of the
  # fork's tag space even when this engine runs in a developer's own clone.
  SYNC_REF="refs/upstream-sync/$UPSTREAM_TAG"
  log "fetching tag $UPSTREAM_TAG from $UPSTREAM_URL"
  if ! git fetch --no-tags "$UPSTREAM_URL" "refs/tags/$UPSTREAM_TAG:$SYNC_REF" >&2; then
    err "failed to fetch tag $UPSTREAM_TAG from $UPSTREAM_URL"
    exit 1
  fi

  local tag_commit
  if ! tag_commit="$(git rev-parse --verify "$SYNC_REF^{commit}" 2>/dev/null)"; then
    err "cannot resolve $SYNC_REF^{commit} after fetch"
    exit 1
  fi
  log "resolved $UPSTREAM_TAG -> $tag_commit"
}

cleanup_sync_ref() {
  git update-ref -d "$SYNC_REF" 2>/dev/null || true
}

# True when the tag's commit is already reachable from HEAD — i.e. the fork
# has this upstream code, however it arrived (this engine, or a maintainer's
# manual `git merge upstream/main`). This is the drift-proof no-op test:
# git ancestry cannot go stale the way a checked-in state file does.
already_merged() {
  git merge-base --is-ancestor "$SYNC_REF^{commit}" HEAD 2>/dev/null
}

merge_tag() {
  if git merge --no-ff -m "Merge upstream tag $UPSTREAM_TAG" "$SYNC_REF" >&2; then
    log "clean merge of $UPSTREAM_TAG"
    cleanup_sync_ref
    return
  fi

  local conflicts=()
  # while-read instead of mapfile: bash 3.2 (stock macOS) lacks mapfile, and
  # developers may run this engine locally outside the CI runner.
  local line
  while IFS= read -r line; do
    [[ -n "$line" ]] && conflicts+=("$line")
  done < <(git diff --name-only --diff-filter=U)
  if ((${#conflicts[@]} == 0)); then
    err "merge failed without unmerged paths; attempting abort"
    git merge --abort >&2 2>&1 || true
    cleanup_sync_ref
    exit 1
  fi

  local f
  local necessary=()
  for f in "${conflicts[@]}"; do
    if is_whitelisted "$f"; then
      log "auto-resolving whitelisted conflict (keep ours): $f"
      if git checkout --ours -- "$f" >&2 2>&1; then
        git add -- "$f" >&2
      else
        # DU: we deleted the file (no stage-2 entry) — keep the deletion.
        git rm -f -- "$f" >&2
      fi
    else
      necessary+=("$f")
    fi
  done

  if ((${#necessary[@]} > 0)); then
    git merge --abort >&2 2>&1
    cleanup_sync_ref
    log "necessary conflict(s) outside the whitelist; merge aborted"
    printf '%s\n' "${necessary[@]}"
    exit 10
  fi

  if ! git commit --no-edit >&2; then
    err "failed to conclude the merge commit"
    exit 1
  fi
  cleanup_sync_ref
  log "merge conflicts auto-resolved (whitelist); merge committed"
}

write_state_file() {
  local state_dir
  state_dir="$(dirname -- "$STATE_FILE")"
  mkdir -p -- "$state_dir"
  printf '%s\n' "$UPSTREAM_TAG" >"$STATE_FILE"
  git add -- "$STATE_FILE"
  if ! git commit -m "chore: record last-synced upstream tag $UPSTREAM_TAG [skip ci]" >&2; then
    err "failed to commit $STATE_FILE"
    exit 1
  fi
  log "recorded $UPSTREAM_TAG in $STATE_FILE; sync complete"
}

main() {
  case "${1:-}" in
    -h | --help)
      usage
      exit 0
      ;;
    "") ;;
    *)
      err "unknown argument: $1"
      usage >&2
      exit 1
      ;;
  esac

  if [[ -z "$UPSTREAM_TAG" ]]; then
    err "UPSTREAM_TAG is required"
    usage >&2
    exit 1
  fi

  preflight
  noop_check
  fetch_tag

  # Ancestry fast path: the code is already here, so there is nothing to merge
  # and nothing for the caller to verify. Only the bookkeeping needs to catch
  # up. Without this the engine builds an empty merge commit and burns a full
  # verification cycle every time the fork was synced by hand.
  if already_merged; then
    log "$UPSTREAM_TAG is already an ancestor of HEAD; recording it without a merge"
    cleanup_sync_ref
    write_state_file
    result state-only
    exit 0
  fi

  merge_tag
  write_state_file
  result merged
  exit 0
}

main "$@"
