#!/bin/sh

set -eu

mode="${MIGRATION_MODE:-none}"
archive="${MIGRATION_ARCHIVE:-/migration/sub2api-full-instance.tar.gz}"
data_dir="${MIGRATION_DATA_DIR:-/app/data}"
force="${MIGRATION_FORCE:-0}"

archive_dir="$(dirname "${archive}")"
marker="${archive_dir}/.$(basename "${archive}").imported"

fail() {
    printf '%s\n' "$*" >&2
    exit 1
}

archive_hash() {
    sha256sum "${archive}" | awk '{print $1}'
}

write_import_marker() {
    marker_tmp="$(mktemp "${archive_dir}/.$(basename "${archive}").imported.XXXXXX")"
    printf '%s\n' "$1" >"${marker_tmp}"
    mv "${marker_tmp}" "${marker}"
}

create_archive() {
    work_dir="$(mktemp -d)"
    trap 'rm -rf "${work_dir}"' EXIT HUP INT TERM

    [ -d "${data_dir}" ] || fail "migration data directory does not exist: ${data_dir}"
    mkdir -p "$(dirname "${archive}")" "${work_dir}/app-data"
    umask 077
    pg_dump --no-owner --no-acl --clean --if-exists >"${work_dir}/postgres.sql"
    cp -a "${data_dir}/." "${work_dir}/app-data/"
    tar -C "${work_dir}" -czf "${archive}" postgres.sql app-data
    chmod 600 "${archive}"
}

restore_archive() {
    work_dir="$(mktemp -d)"
    trap 'rm -rf "${work_dir}"' EXIT HUP INT TERM

    [ -f "${archive}" ] || fail "migration archive does not exist: ${archive}"
    tar -tzf "${archive}" >/dev/null 2>&1 || fail "migration archive is invalid: ${archive}"
    tar -xzf "${archive}" -C "${work_dir}"
    [ -f "${work_dir}/postgres.sql" ] || fail "migration archive does not contain postgres.sql"
    [ -d "${work_dir}/app-data" ] || fail "migration archive does not contain app-data"

    psql -v ON_ERROR_STOP=1 --single-transaction <"${work_dir}/postgres.sql"
    mkdir -p "${data_dir}"
    find "${data_dir}" -mindepth 1 -maxdepth 1 -exec rm -rf {} +
    cp -a "${work_dir}/app-data/." "${data_dir}/"
    write_import_marker "$(archive_hash)"
}

case "${mode}" in
    none)
        exit 0
        ;;
    export)
        if [ -f "${archive}" ] && [ "${force}" != 1 ]; then
            printf '%s\n' "migration archive already exists: ${archive}; skipping export (set MIGRATION_FORCE=1 to overwrite)"
            exit 0
        fi
        create_archive
        ;;
    import)
        if [ "${force}" != 1 ] && [ -f "${marker}" ]; then
            if [ ! -f "${archive}" ]; then
                printf '%s\n' "migration archive already imported: ${archive}"
                exit 0
            fi
            if [ "$(cat "${marker}")" = "$(archive_hash)" ]; then
                printf '%s\n' "migration archive already imported: ${archive}"
                exit 0
            fi
        fi
        restore_archive
        ;;
    *)
        fail "unsupported migration mode: ${mode}"
        ;;
esac
