#!/usr/bin/env bash

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "${TEST_DIR}/.." && pwd)"
RUNNER="${DEPLOY_DIR}/full-instance-migration.sh"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-migration-test.XXXXXX")"

cleanup() {
    rm -rf "${TEST_ROOT}"
}
trap cleanup EXIT

fail() {
    printf 'FAIL: %s\n' "$*" >&2
    exit 1
}

assert_exists() {
    [[ -e "$1" ]] || fail "Expected path to exist: $1"
}

assert_equals() {
    [[ "$1" == "$2" ]] || fail "Expected '$1' to equal '$2'"
}

write_fake_postgres_clients() {
    local bin_dir="$1"

    mkdir -p "${bin_dir}"
    cat >"${bin_dir}/pg_dump" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' '-- fixture PostgreSQL dump'
EOF
    cat >"${bin_dir}/psql" <<'EOF'
#!/usr/bin/env bash
cat >"${PSQL_CAPTURE:?PSQL_CAPTURE is required}"
EOF
    chmod +x "${bin_dir}/pg_dump" "${bin_dir}/psql"
}

test_export_archive_contains_postgres_sql_and_app_data() {
    local root="${TEST_ROOT}/export"
    local archive="${root}/migration/instance.tar.gz"
    local source_data="${root}/data"

    mkdir -p "${source_data}/templates" "$(dirname "${archive}")"
    printf '%s\n' 'server: migrated' >"${source_data}/config.yaml"
    printf '%s\n' 'custom instructions' >"${source_data}/templates/codex.md.tmpl"
    write_fake_postgres_clients "${root}/bin"

    PATH="${root}/bin:${PATH}" \
        MIGRATION_MODE=export \
        MIGRATION_ARCHIVE="${archive}" \
        MIGRATION_DATA_DIR="${source_data}" \
        /bin/sh "${RUNNER}"

    assert_exists "${archive}"
    assert_equals "600" "$(stat -c '%a' "${archive}")"
    tar -tzf "${archive}" >"${root}/archive.list"
    grep -Fxq 'postgres.sql' "${root}/archive.list" || fail 'Archive did not contain postgres.sql'
    grep -Fxq 'app-data/config.yaml' "${root}/archive.list" || fail 'Archive did not contain config.yaml'
    grep -Fxq 'app-data/templates/codex.md.tmpl' "${root}/archive.list" || fail 'Archive did not contain runtime templates'
    assert_equals '-- fixture PostgreSQL dump' "$(tar -xOzf "${archive}" postgres.sql)"
}

test_import_restores_postgres_sql_and_app_data() {
    local root="${TEST_ROOT}/import"
    local archive="${root}/migration/instance.tar.gz"
    local source_data="${root}/source-data"
    local target_data="${root}/target-data"
    local psql_capture="${root}/psql-input"

    mkdir -p "${source_data}/templates" "${target_data}" "$(dirname "${archive}")"
    printf '%s\n' 'server: migrated' >"${source_data}/config.yaml"
    printf '%s\n' 'custom instructions' >"${source_data}/templates/codex.md.tmpl"
    printf '%s\n' 'stale' >"${target_data}/stale"
    write_fake_postgres_clients "${root}/bin"

    PATH="${root}/bin:${PATH}" \
        MIGRATION_MODE=export \
        MIGRATION_ARCHIVE="${archive}" \
        MIGRATION_DATA_DIR="${source_data}" \
        /bin/sh "${RUNNER}"

    PATH="${root}/bin:${PATH}" \
        PSQL_CAPTURE="${psql_capture}" \
        MIGRATION_MODE=import \
        MIGRATION_ARCHIVE="${archive}" \
        MIGRATION_DATA_DIR="${target_data}" \
        /bin/sh "${RUNNER}"

    assert_equals '-- fixture PostgreSQL dump' "$(<"${psql_capture}")"
    assert_equals 'server: migrated' "$(<"${target_data}/config.yaml")"
    assert_equals 'custom instructions' "$(<"${target_data}/templates/codex.md.tmpl")"
    [[ ! -e "${target_data}/stale" ]] || fail 'Import retained stale target data'
}

test_import_rejects_missing_or_corrupt_archive() {
    local root="${TEST_ROOT}/invalid-import"
    local archive="${root}/migration/instance.tar.gz"
    local target_data="${root}/data"
    local psql_capture="${root}/psql-input"

    mkdir -p "${target_data}" "$(dirname "${archive}")"
    printf '%s\n' 'keep-me' >"${target_data}/sentinel"
    write_fake_postgres_clients "${root}/bin"

    if PATH="${root}/bin:${PATH}" \
        PSQL_CAPTURE="${psql_capture}" \
        MIGRATION_MODE=import \
        MIGRATION_ARCHIVE="${archive}" \
        MIGRATION_DATA_DIR="${target_data}" \
        /bin/sh "${RUNNER}"; then
        fail 'Import accepted a missing archive'
    fi
    assert_equals 'keep-me' "$(<"${target_data}/sentinel")"
    [[ ! -e "${psql_capture}" ]] || fail 'Import invoked psql for a missing archive'

    printf '%s\n' 'not a gzip archive' >"${archive}"
    if PATH="${root}/bin:${PATH}" \
        PSQL_CAPTURE="${psql_capture}" \
        MIGRATION_MODE=import \
        MIGRATION_ARCHIVE="${archive}" \
        MIGRATION_DATA_DIR="${target_data}" \
        /bin/sh "${RUNNER}"; then
        fail 'Import accepted a corrupt archive'
    fi
    assert_equals 'keep-me' "$(<"${target_data}/sentinel")"
    [[ ! -e "${psql_capture}" ]] || fail 'Import invoked psql for a corrupt archive'
}

test_none_mode_leaves_target_untouched() {
    local root="${TEST_ROOT}/none"
    local target_data="${root}/data"

    mkdir -p "${target_data}"
    printf '%s\n' 'keep-me' >"${target_data}/sentinel"

    MIGRATION_MODE=none MIGRATION_DATA_DIR="${target_data}" /bin/sh "${RUNNER}"

    assert_equals 'keep-me' "$(<"${target_data}/sentinel")"
}

test_compose_gates_sub2api_after_migration() {
    local compose_file
    local rendered

    for compose_file in docker-compose.yml docker-compose.local.yml docker-compose.dev.yml; do
        rendered="$(POSTGRES_PASSWORD=test-password docker compose -f "${DEPLOY_DIR}/${compose_file}" config)"
        grep -q '^  migration:$' <<<"${rendered}" || fail "${compose_file} does not define migration"
        grep -q '^        condition: service_completed_successfully$' <<<"${rendered}" || fail "${compose_file} does not gate startup on migration completion"
    done
}

test_deploy_script_downloads_migration_runner() {
    local root="${TEST_ROOT}/docker-deploy"
    local bin_dir="${root}/bin"

    mkdir -p "${bin_dir}"
    cat >"${bin_dir}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

output=""
url=""
while (($#)); do
    case "$1" in
        -o)
            output="$2"
            shift 2
            ;;
        *)
            url="$1"
            shift
            ;;
    esac
done

case "${url}" in
    */docker-compose.local.yml)
        printf '%s\n' 'services: {}' >"${output}"
        ;;
    */.env.example)
        printf '%s\n' 'JWT_SECRET=' 'TOTP_ENCRYPTION_KEY=' 'POSTGRES_PASSWORD=change_this_secure_password' >"${output}"
        ;;
    */full-instance-migration.sh)
        printf '%s\n' '#!/bin/sh' 'exit 0' >"${output}"
        ;;
    *)
        exit 1
        ;;
esac
EOF
    cat >"${bin_dir}/openssl" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' 'test-secret'
EOF
    chmod +x "${bin_dir}/curl" "${bin_dir}/openssl"

    (
        cd "${root}"
        PATH="${bin_dir}:${PATH}" bash "${DEPLOY_DIR}/docker-deploy.sh" >/dev/null
    )

    assert_exists "${root}/full-instance-migration.sh"
}

test_export_archive_contains_postgres_sql_and_app_data
test_import_restores_postgres_sql_and_app_data
test_import_rejects_missing_or_corrupt_archive
test_none_mode_leaves_target_untouched
test_compose_gates_sub2api_after_migration
test_deploy_script_downloads_migration_runner

printf 'Full-instance migration tests passed.\n'
