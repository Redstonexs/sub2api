#!/usr/bin/env bash
#
# tools/tests/upstream_sync_test.sh
#
# Network-free, pure-bash fixture test harness for tools/upstream_sync.sh.
# TDD RED phase: the engine does not exist yet, so every scenario is expected
# to FAIL (engine invocation exits 127 with "No such file or directory").
#
# For each scenario the harness builds a FRESH fixture pair under mktemp -d:
#   upstream/  a local git repo ("the upstream project"), tagged releases
#   fork/      a local-path clone of upstream with fork divergences applied
# The engine is then invoked inside the fork with the frozen contract:
#
#   cd <fork> && UPSTREAM_URL=<upstream path> UPSTREAM_TAG=<tag> \
#     BASE_BRANCH=main STATE_FILE=.github/upstream-sync-tag \
#     bash <repo>/tools/upstream_sync.sh
#
# All git remotes are plain filesystem paths: ZERO network access.
# Temp dirs are removed via a trap on EXIT, even when scenarios fail.
#
# Usage: bash tools/tests/upstream_sync_test.sh   (runnable from anywhere)

set -u
set -o pipefail
# NOTE: deliberately no `set -e` — failures must be captured and reported
# per scenario, not abort the harness.

# Hermetic git fixtures: never let a caller's git env leak in.
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY \
  GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_PREFIX GIT_WORK_TREE 2>/dev/null || true

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENGINE="$REPO_ROOT/tools/upstream_sync.sh"
BASE_BRANCH="main"
STATE_FILE_REL=".github/upstream-sync-tag"
DEFAULT_UPSTREAM_URL="https://github.com/Wei-Shaw/sub2api.git" # documented default; never used (offline)

TMPROOT="$(mktemp -d "${TMPDIR:-/tmp}/upstream_sync_test.XXXXXXXX")"
cleanup() {
  rm -rf "$TMPROOT" 2>/dev/null || true
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
# TAP reporting
# ---------------------------------------------------------------------------

FAILURES=()
FIX="" # absolute path of the current scenario's fixture dir
TESTS_RUN=0
TESTS_FAILED=0

record_failure() { FAILURES+=("$1"); }

begin_scenario() { FAILURES=(); }

end_scenario() {
  local name="$1"
  TESTS_RUN=$((TESTS_RUN + 1))
  if ((${#FAILURES[@]} == 0)); then
    echo "ok $TESTS_RUN - $name"
    return
  fi
  TESTS_FAILED=$((TESTS_FAILED + 1))
  echo "not ok $TESTS_RUN - $name"
  local f
  for f in "${FAILURES[@]}"; do
    echo "#   $f"
  done
  if [[ -s "$FIX/engine.stderr" ]]; then
    local line
    while IFS= read -r line; do
      echo "#   stderr: $line"
    done <"$FIX/engine.stderr"
  fi
}

# ---------------------------------------------------------------------------
# Assertion helpers
# ---------------------------------------------------------------------------

assert_exit() {
  local expected="$1" actual="$2" label="$3"
  if [[ "$actual" != "$expected" ]]; then
    record_failure "$label: expected exit code $expected, got $actual"
  fi
}

assert_file_content() {
  local path="$1" expected="$2" label="$3"
  if [[ ! -f "$path" ]]; then
    record_failure "$label: file '$path' does not exist"
    return
  fi
  local actual
  actual="$(cat "$path")"
  if [[ "$actual" != "$expected" ]]; then
    record_failure "$label: '$path' content mismatch: expected <<$expected>> got <<$actual>>"
  fi
}

assert_file_absent() {
  local path="$1" label="$2"
  if [[ -e "$path" ]]; then
    record_failure "$label: '$path' should not exist"
  fi
}

assert_commit_count_delta() {
  local repo="$1" before="$2" expected_delta="$3" label="$4"
  local after actual_delta
  after="$(git -C "$repo" rev-list --count HEAD)"
  actual_delta=$((after - before))
  if [[ "$actual_delta" != "$expected_delta" ]]; then
    record_failure "$label: expected commit-count delta $expected_delta, got $actual_delta (before=$before after=$after)"
  fi
}

assert_stdout_contains() {
  local stdout_path="$1" needle="$2" label="$3"
  if [[ ! -f "$stdout_path" ]] || ! grep -qF -- "$needle" "$stdout_path"; then
    record_failure "$label: stdout does not contain '$needle'"
  fi
}

# Byte-identical comparison against a saved pre-merge copy.
assert_file_bytes_equal() {
  local path="$1" ref="$2" label="$3"
  if [[ ! -f "$path" ]]; then
    record_failure "$label: '$path' does not exist"
    return
  fi
  if [[ ! -f "$ref" ]]; then
    record_failure "$label: reference copy '$ref' missing (harness bug)"
    return
  fi
  if ! cmp -s "$path" "$ref"; then
    record_failure "$label: '$path' differs byte-wise from its pre-merge content"
  fi
}

# Contract: sync = merge commit + separate state-file commit on top (never amended).
assert_merge_and_state_commit() {
  local repo="$1" tag="$2" label="$3"
  local subject fields
  subject="$(git -C "$repo" log -1 --format=%s HEAD)"
  if [[ "$subject" != "chore: record last-synced upstream tag $tag [skip ci]" ]]; then
    record_failure "$label: HEAD is not the state-file commit (subject: <<$subject>>)"
  fi
  fields="$(git -C "$repo" rev-list --parents -n 1 'HEAD^' | wc -w | tr -d '[:space:]')"
  if [[ "$fields" != "3" ]]; then
    record_failure "$label: HEAD^ is not a merge commit (rev-list --parents field count=$fields, expected 3)"
  fi
}

assert_head_unchanged() {
  local repo="$1" before_sha="$2" label="$3"
  local now
  now="$(git -C "$repo" rev-parse HEAD)"
  if [[ "$now" != "$before_sha" ]]; then
    record_failure "$label: HEAD moved (was $before_sha, now $now)"
  fi
}

assert_git_clean() {
  local repo="$1" label="$2"
  local out
  out="$(git -C "$repo" status --porcelain 2>/dev/null)"
  if [[ -n "$out" ]]; then
    record_failure "$label: git status not clean: $out"
  fi
}

# Tag-hygiene contract: upstream release tags must never enter the fork's
# refs/tags (they would corrupt git describe-based fork version resolution).
assert_ref_absent() {
  local repo="$1" ref="$2" label="$3"
  if git -C "$repo" show-ref --verify --quiet "$ref"; then
    record_failure "$label: ref '$ref' should not exist"
  fi
}

# ---------------------------------------------------------------------------
# Fixture helpers
# ---------------------------------------------------------------------------

write_file() {
  local repo="$1" rel="$2" content="$3"
  mkdir -p "$repo/$(dirname "$rel")"
  printf '%s\n' "$content" >"$repo/$rel"
}

git_setup_repo() {
  local repo="$1" name="$2" email="$3"
  git -C "$repo" config user.name "$name"
  git -C "$repo" config user.email "$email"
  git -C "$repo" config commit.gpgsign false
  git -C "$repo" config core.autocrlf false
}

git_commit_all() {
  local repo="$1" msg="$2"
  git -C "$repo" add -A
  git -C "$repo" commit -qm "$msg"
}

GATEWAY_V1='package service

// pickAccount returns the upstream account id.
func pickAccount() string {
  return "upstream-v1"
}'

# Seed the initial upstream commit: every whitelisted file plus a shared,
# non-whitelisted gateway_service.go, all with "upstream" content.
seed_shared_files() {
  local repo="$1"
  write_file "$repo" "backend/cmd/server/VERSION" "1.0.0"
  write_file "$repo" "README.md" "# sub2api (upstream)"
  write_file "$repo" "README_CN.md" "# sub2api CN (upstream)"
  write_file "$repo" "README_JA.md" "# sub2api JA (upstream)"
  write_file "$repo" "deploy/install.sh" '#!/bin/sh
# upstream install script'
  write_file "$repo" "deploy/docker-deploy.sh" '#!/bin/sh
# upstream docker deploy script'
  write_file "$repo" "deploy/APPLE_CONTAINER.md" "# Apple container guide (upstream)"
  write_file "$repo" "frontend/src/i18n/locales/en/fork.ts" 'export default { brand: "upstream-en" }'
  write_file "$repo" "frontend/src/i18n/locales/zh/fork.ts" 'export default { brand: "upstream-zh" }'
  write_file "$repo" "frontend/src/composables/useConfirm.ts" 'export function useConfirm() { return "upstream" }'
  write_file "$repo" "frontend/src/utils/chartTheme.ts" 'export const chartTheme = "upstream"'
  write_file "$repo" "frontend/tailwind.config.js" 'module.exports = { theme: "upstream" }'
  write_file "$repo" "frontend/src/style.css" '/* upstream styles */'
  write_file "$repo" "backend/internal/service/announcement_broadcast_service.go" 'package service

// upstream announcement broadcast service'
  write_file "$repo" "backend/internal/service/gateway_service.go" "$GATEWAY_V1"
}

# make_fixture <name>: base repo -> tag v1.0.0 -> local-path clone to fork/.
# Sets the global FIX to the fixture dir. Caller derives:
#   upstream=$FIX/upstream  fork=$FIX/fork
make_fixture() {
  local name="$1"
  FIX="$TMPROOT/$name"
  local upstream="$FIX/upstream"
  local fork="$FIX/fork"

  mkdir -p "$FIX"
  git init -q -b "$BASE_BRANCH" "$upstream"
  git_setup_repo "$upstream" "Upstream Bot" "upstream@example.invalid"

  seed_shared_files "$upstream"
  git -C "$upstream" add -A
  git -C "$upstream" commit -qm "upstream: initial shared files (v1.0.0)"
  git -C "$upstream" tag v1.0.0

  git clone -q "$upstream" "$fork"
  git_setup_repo "$fork" "Fork Bot" "fork@example.invalid"
}

# Baseline fork divergence shared by all scenarios: fork VERSION, fork-only
# i18n overlay content, warm theme files. Whitelist-only, so S1 stays clean.
apply_fork_baseline() {
  local fork="$1"
  write_file "$fork" "backend/cmd/server/VERSION" "1.4.11"
  write_file "$fork" "frontend/src/i18n/locales/en/fork.ts" 'export default { brand: "fork-en" }'
  write_file "$fork" "frontend/src/i18n/locales/zh/fork.ts" 'export default { brand: "fork-zh" }'
  write_file "$fork" "frontend/src/utils/chartTheme.ts" 'export const chartTheme = "fork-warm"'
  write_file "$fork" "frontend/tailwind.config.js" 'module.exports = { theme: "fork-warm-palette" }'
  write_file "$fork" "frontend/src/style.css" '/* fork warm styles */'
  git_commit_all "$fork" "fork: brand overlays, warm palette, VERSION 1.4.11"
}

# run_engine <fork_dir> <upstream_url> <tag>
# Invokes the engine under the frozen contract; returns its exit code.
run_engine() {
  local fork_dir="$1" upstream_url="$2" tag="$3"
  (
    cd "$fork_dir" || exit 99
    UPSTREAM_URL="$upstream_url" \
    UPSTREAM_TAG="$tag" \
    BASE_BRANCH="$BASE_BRANCH" \
    STATE_FILE="$STATE_FILE_REL" \
    bash "$ENGINE" >"$FIX/engine.stdout" 2>"$FIX/engine.stderr"
  )
}

# ---------------------------------------------------------------------------
# Scenarios — each builds its own FRESH fixture pair; no state is shared.
# ---------------------------------------------------------------------------

scenario_s1() {
  begin_scenario
  make_fixture "s1"
  local upstream="$FIX/upstream" fork="$FIX/fork"

  apply_fork_baseline "$fork"

  write_file "$upstream" "backend/internal/service/gateway_service.go" 'package service

// pickAccount returns the upstream account id.
func pickAccount() string {
  return "upstream-v1.1.0"
}'
  git_commit_all "$upstream" "upstream: gateway_service v1.1.0"
  git -C "$upstream" tag v1.1.0

  run_engine "$fork" "$upstream" "v1.1.0"
  local rc=$?

  assert_exit 0 "$rc" "S1 exit code"
  assert_merge_and_state_commit "$fork" "v1.1.0" "S1 merge+state commits on main"
  assert_file_content "$fork/$STATE_FILE_REL" "v1.1.0" "S1 STATE_FILE written"
  assert_file_content "$fork/backend/internal/service/gateway_service.go" 'package service

// pickAccount returns the upstream account id.
func pickAccount() string {
  return "upstream-v1.1.0"
}' "S1 upstream change present"
  assert_ref_absent "$fork" "refs/tags/v1.1.0" "S1 upstream tag not leaked into refs/tags"
  assert_ref_absent "$fork" "refs/upstream-sync/v1.1.0" "S1 temp sync ref cleaned"
  assert_stdout_contains "$FIX/engine.stdout" "sync-result=merged" "S1 reports merged"

  end_scenario "S1 clean merge of non-whitelist upstream change"
}

scenario_s2() {
  begin_scenario
  make_fixture "s2"
  local upstream="$FIX/upstream" fork="$FIX/fork"

  apply_fork_baseline "$fork"
  write_file "$fork" "README.md" "# sub2api (fork edition)"
  git_commit_all "$fork" "fork: README branding"

  write_file "$upstream" "backend/cmd/server/VERSION" "1.1.0"
  write_file "$upstream" "README.md" "# sub2api (upstream v1.1.0)"
  git_commit_all "$upstream" "upstream: release v1.1.0"
  git -C "$upstream" tag v1.1.0

  cp "$fork/backend/cmd/server/VERSION" "$FIX/ref_VERSION"

  run_engine "$fork" "$upstream" "v1.1.0"
  local rc=$?

  assert_exit 0 "$rc" "S2 exit code"
  assert_merge_and_state_commit "$fork" "v1.1.0" "S2 merge+state commits on main"
  assert_file_bytes_equal "$fork/backend/cmd/server/VERSION" "$FIX/ref_VERSION" "S2 VERSION byte-identical to pre-merge"
  assert_file_content "$fork/backend/cmd/server/VERSION" "1.4.11" "S2 VERSION kept ours"
  assert_file_content "$fork/README.md" "# sub2api (fork edition)" "S2 README kept ours"
  assert_file_content "$fork/$STATE_FILE_REL" "v1.1.0" "S2 STATE_FILE written"

  end_scenario "S2 whitelist conflict auto-resolved keeping ours"
}

scenario_s3() {
  begin_scenario
  make_fixture "s3"
  local upstream="$FIX/upstream" fork="$FIX/fork"

  apply_fork_baseline "$fork"
  write_file "$fork" "backend/internal/service/gateway_service.go" 'package service

// pickAccount returns the upstream account id.
func pickAccount() string {
  return "fork-v1.1.0"
}'
  git_commit_all "$fork" "fork: gateway_service tweak"

  write_file "$upstream" "backend/internal/service/gateway_service.go" 'package service

// pickAccount returns the upstream account id.
func pickAccount() string {
  return "upstream-v1.1.0"
}'
  git_commit_all "$upstream" "upstream: gateway_service v1.1.0"
  git -C "$upstream" tag v1.1.0

  local before_sha
  before_sha="$(git -C "$fork" rev-parse HEAD)"

  run_engine "$fork" "$upstream" "v1.1.0"
  local rc=$?

  assert_exit 10 "$rc" "S3 exit code"
  assert_file_content "$FIX/engine.stdout" "backend/internal/service/gateway_service.go" "S3 stdout is exactly the conflict path"
  assert_stdout_contains "$FIX/engine.stdout" "backend/internal/service/gateway_service.go" "S3 stdout lists conflict"
  assert_git_clean "$fork" "S3 working tree clean after abort"
  assert_head_unchanged "$fork" "$before_sha" "S3 no merge commit created"
  assert_file_absent "$fork/$STATE_FILE_REL" "S3 STATE_FILE untouched"
  assert_file_content "$fork/backend/internal/service/gateway_service.go" 'package service

// pickAccount returns the upstream account id.
func pickAccount() string {
  return "fork-v1.1.0"
}' "S3 fork content preserved by abort"
  assert_ref_absent "$fork" "refs/tags/v1.1.0" "S3 upstream tag not leaked into refs/tags"
  assert_ref_absent "$fork" "refs/upstream-sync/v1.1.0" "S3 temp sync ref cleaned after abort"

  end_scenario "S3 necessary conflict on non-whitelist file aborts merge"
}

scenario_s4() {
  begin_scenario
  make_fixture "s4"
  local upstream="$FIX/upstream" fork="$FIX/fork"

  apply_fork_baseline "$fork"
  write_file "$fork" "$STATE_FILE_REL" "v1.1.0"
  git_commit_all "$fork" "fork: record last sync tag v1.1.0"

  write_file "$upstream" "README.md" "# sub2api (upstream v1.1.0)"
  git_commit_all "$upstream" "upstream: release v1.1.0"
  git -C "$upstream" tag v1.1.0

  local before_count before_sha
  before_count="$(git -C "$fork" rev-list --count HEAD)"
  before_sha="$(git -C "$fork" rev-parse HEAD)"

  run_engine "$fork" "$upstream" "v1.1.0"
  local rc=$?

  assert_exit 0 "$rc" "S4 exit code"
  assert_commit_count_delta "$fork" "$before_count" 0 "S4 zero new commits"
  assert_head_unchanged "$fork" "$before_sha" "S4 HEAD unchanged"
  assert_file_content "$fork/$STATE_FILE_REL" "v1.1.0" "S4 STATE_FILE still v1.1.0"
  assert_stdout_contains "$FIX/engine.stdout" "sync-result=noop" "S4 reports noop"

  end_scenario "S4 no-op when STATE_FILE already contains tag"
}

scenario_s5() {
  begin_scenario
  make_fixture "s5"
  local upstream="$FIX/upstream" fork="$FIX/fork"

  apply_fork_baseline "$fork"

  write_file "$upstream" "README_CN.md" "# sub2api CN (upstream v1.1.0)"
  git_commit_all "$upstream" "upstream: README_CN v1.1.0"
  git -C "$upstream" tag v1.1.0

  cp "$fork/frontend/src/i18n/locales/en/fork.ts" "$FIX/ref_en_fork.ts"
  cp "$fork/frontend/src/utils/chartTheme.ts" "$FIX/ref_chartTheme.ts"

  run_engine "$fork" "$upstream" "v1.1.0"
  local rc=$?

  assert_exit 0 "$rc" "S5 exit code"
  assert_merge_and_state_commit "$fork" "v1.1.0" "S5 merge+state commits on main"
  assert_file_bytes_equal "$fork/frontend/src/i18n/locales/en/fork.ts" "$FIX/ref_en_fork.ts" "S5 en/fork.ts byte-identical"
  assert_file_bytes_equal "$fork/frontend/src/utils/chartTheme.ts" "$FIX/ref_chartTheme.ts" "S5 chartTheme.ts byte-identical"
  assert_file_content "$fork/$STATE_FILE_REL" "v1.1.0" "S5 STATE_FILE written"

  end_scenario "S5 fork-only overlay files byte-identical after merge"
}

scenario_s6a() {
  begin_scenario
  make_fixture "s6a"
  local upstream="$FIX/upstream" fork="$FIX/fork"

  apply_fork_baseline "$fork"
  write_file "$fork" "deploy/APPLE_CONTAINER.md" "# Apple container guide (fork edition)"
  git_commit_all "$fork" "fork: apple container guide"

  git -C "$upstream" rm -q deploy/APPLE_CONTAINER.md
  git_commit_all "$upstream" "upstream: drop APPLE_CONTAINER.md"
  git -C "$upstream" tag v1.3.0

  run_engine "$fork" "$upstream" "v1.3.0"
  local rc=$?

  assert_exit 0 "$rc" "S6a exit code"
  assert_merge_and_state_commit "$fork" "v1.3.0" "S6a merge+state commits on main"
  assert_file_content "$fork/deploy/APPLE_CONTAINER.md" "# Apple container guide (fork edition)" "S6a modified file kept (UD -> ours)"
  assert_file_content "$fork/$STATE_FILE_REL" "v1.3.0" "S6a STATE_FILE written"

  end_scenario "S6a UD conflict: upstream delete vs fork modify keeps ours"
}

scenario_s6b() {
  begin_scenario
  make_fixture "s6b"
  local upstream="$FIX/upstream" fork="$FIX/fork"

  apply_fork_baseline "$fork"
  git -C "$fork" rm -q README_JA.md
  git_commit_all "$fork" "fork: drop README_JA.md"

  write_file "$upstream" "README_JA.md" "# sub2api JA (upstream v1.3.0)"
  git_commit_all "$upstream" "upstream: README_JA v1.3.0"
  git -C "$upstream" tag v1.3.0

  run_engine "$fork" "$upstream" "v1.3.0"
  local rc=$?

  assert_exit 0 "$rc" "S6b exit code"
  assert_merge_and_state_commit "$fork" "v1.3.0" "S6b merge+state commits on main"
  assert_file_absent "$fork/README_JA.md" "S6b deletion kept (DU -> git rm)"
  assert_file_content "$fork/$STATE_FILE_REL" "v1.3.0" "S6b STATE_FILE written"

  end_scenario "S6b DU conflict: fork delete vs upstream modify keeps deletion"
}

scenario_s8() {
  begin_scenario
  make_fixture "s8"
  local upstream="$FIX/upstream" fork="$FIX/fork"

  apply_fork_baseline "$fork"

  write_file "$upstream" "backend/internal/service/gateway_service.go" 'package service

// pickAccount returns the upstream account id.
func pickAccount() string {
  return "upstream-v1.2.0"
}'
  git_commit_all "$upstream" "upstream: gateway_service v1.2.0"
  git -C "$upstream" tag -a v1.2.0 -m "release v1.2.0"

  run_engine "$fork" "$upstream" "v1.2.0"
  local rc=$?

  assert_exit 0 "$rc" "S8 exit code"
  assert_merge_and_state_commit "$fork" "v1.2.0" "S8 merge+state commits on main"
  assert_file_content "$fork/$STATE_FILE_REL" "v1.2.0" "S8 STATE_FILE written"
  assert_file_content "$fork/backend/internal/service/gateway_service.go" 'package service

// pickAccount returns the upstream account id.
func pickAccount() string {
  return "upstream-v1.2.0"
}' "S8 upstream change present"

  end_scenario "S8 annotated upstream tag merges successfully"
}

# Regression guard for the real-world drift that wedged the workflow: the
# maintainer merges upstream/main by hand, so the tag is already an ancestor of
# HEAD while STATE_FILE still names a much older tag. The engine must recognise
# that from ancestry, skip the merge entirely, and only move the bookkeeping —
# no empty merge commit, and a marker telling the caller there is nothing to
# verify.
scenario_s9() {
  begin_scenario
  make_fixture "s9"
  local upstream="$FIX/upstream" fork="$FIX/fork"

  apply_fork_baseline "$fork"
  write_file "$fork" "$STATE_FILE_REL" "v1.0.0"
  git_commit_all "$fork" "fork: record last sync tag v1.0.0"

  write_file "$upstream" "backend/internal/service/gateway_service.go" 'package service

// pickAccount returns the upstream account id.
func pickAccount() string {
  return "upstream-v1.1.0"
}'
  git_commit_all "$upstream" "upstream: gateway_service v1.1.0"
  git -C "$upstream" tag v1.1.0

  # Simulate `git merge upstream/main` done by hand, bypassing this engine:
  # the code lands but STATE_FILE is left behind at v1.0.0.
  git -C "$fork" fetch -q origin "$BASE_BRANCH"
  git -C "$fork" merge -q --no-ff -m "Merge upstream/main by hand" FETCH_HEAD

  local before_count
  before_count="$(git -C "$fork" rev-list --count HEAD)"

  run_engine "$fork" "$upstream" "v1.1.0"
  local rc=$?

  assert_exit 0 "$rc" "S9 exit code"
  assert_stdout_contains "$FIX/engine.stdout" "sync-result=state-only" "S9 reports state-only"
  assert_commit_count_delta "$fork" "$before_count" 1 "S9 exactly one new commit (state file only)"
  assert_file_content "$fork/$STATE_FILE_REL" "v1.1.0" "S9 STATE_FILE fast-forwarded"
  assert_git_clean "$fork" "S9 tree clean"
  assert_ref_absent "$fork" "refs/tags/v1.1.0" "S9 upstream tag not leaked into refs/tags"
  assert_ref_absent "$fork" "refs/upstream-sync/v1.1.0" "S9 temp sync ref cleaned"

  local head_subject
  head_subject="$(git -C "$fork" log -1 --format=%s HEAD)"
  if [[ "$head_subject" != "chore: record last-synced upstream tag v1.1.0 [skip ci]" ]]; then
    record_failure "S9 HEAD is not the state-file commit (subject: <<$head_subject>>)"
  fi
  # The engine must not have manufactured a merge commit on top.
  local head_parents
  head_parents="$(git -C "$fork" rev-list --parents -n 1 HEAD | wc -w | tr -d '[:space:]')"
  if [[ "$head_parents" != "2" ]]; then
    record_failure "S9 HEAD should be a normal one-parent commit (field count=$head_parents, expected 2)"
  fi

  end_scenario "S9 tag already an ancestor records state without re-merging"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
  echo "TAP version 13"
  echo "1..9"
  if [[ -f "$ENGINE" ]]; then
    echo "# engine: $ENGINE (present)"
  else
    echo "# engine: $ENGINE (MISSING - TDD RED phase)"
  fi

  scenario_s1
  scenario_s2
  scenario_s3
  scenario_s4
  scenario_s5
  scenario_s6a
  scenario_s6b
  scenario_s8
  scenario_s9

  local total=9 passed
  passed=$((total - TESTS_FAILED))
  echo "PASSED: $passed/$total"

  if ((TESTS_FAILED > 0)); then
    exit 1
  fi
  exit 0
}

main "$@"
