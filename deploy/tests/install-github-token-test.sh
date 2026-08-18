#!/bin/bash

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

cat > "$TEMP_DIR/curl" <<'EOF'
#!/bin/bash
printf '%s\n' "$@" > "$CURL_ARGS_LOG"
env > "${CURL_ARGS_LOG}.env"
cat > "${CURL_ARGS_LOG}.stdin"
EOF
chmod +x "$TEMP_DIR/curl"

mkdir "$TEMP_DIR/home"
cat > "$TEMP_DIR/home/.curlrc" <<'EOF'
url = "https://example.com/collect"
header = "X-Leaked-From-Curlrc: yes"
EOF

run_api_curl() {
    CURL_ARGS_LOG="$1" HOME="$TEMP_DIR/home" PATH="$TEMP_DIR:$PATH" UPDATE_GITHUB_TOKEN="${2:-}" \
        GITHUB_TOKEN="github-fallback" GH_TOKEN="gh-fallback" \
        bash -c 'source "$1"; github_api_curl -s "$2"' bash \
        "$ROOT_DIR/deploy/install.sh" "https://api.github.com/repos/Redstonexs/sub2api/releases/latest"
}

run_api_curl "$TEMP_DIR/authenticated" "update-secret"
test "$(head -n 1 "$TEMP_DIR/authenticated")" = '-q'
grep -Fxq -- '--config' "$TEMP_DIR/authenticated"
grep -Fxq -- '-' "$TEMP_DIR/authenticated"
grep -Fxq -- '--globoff' "$TEMP_DIR/authenticated"
grep -Fxq 'header = "Authorization: Bearer update-secret"' "$TEMP_DIR/authenticated.stdin"
if grep -Fq 'update-secret' "$TEMP_DIR/authenticated"; then
    echo "installer exposed the update token in curl argv" >&2
    exit 1
fi
if grep -Eq 'update-secret|github-fallback|gh-fallback' "$TEMP_DIR/authenticated.env"; then
    echo "installer exposed a token in curl environment" >&2
    exit 1
fi
test "$(grep -Fxc 'https://api.github.com/repos/Redstonexs/sub2api/releases/latest' "$TEMP_DIR/authenticated")" -eq 1
if grep -Fq 'example.com/collect' "$TEMP_DIR/authenticated" || grep -Fq 'X-Leaked-From-Curlrc' "$TEMP_DIR/authenticated" ||
    grep -Fq 'example.com/collect' "$TEMP_DIR/authenticated.stdin" || grep -Fq 'X-Leaked-From-Curlrc' "$TEMP_DIR/authenticated.stdin"; then
    echo "installer allowed hostile curl config into authenticated invocation" >&2
    exit 1
fi

run_api_curl "$TEMP_DIR/anonymous"
test "$(head -n 1 "$TEMP_DIR/anonymous")" = '-q'
if grep -Eq 'github-fallback|gh-fallback' "$TEMP_DIR/anonymous.env"; then
    echo "installer exposed a fallback token in anonymous curl environment" >&2
    exit 1
fi
if grep -Fq 'Authorization:' "$TEMP_DIR/anonymous"; then
    echo "installer unexpectedly used a fallback token" >&2
    exit 1
fi
test ! -s "$TEMP_DIR/anonymous.stdin"
test "$(grep -Fxc 'https://api.github.com/repos/Redstonexs/sub2api/releases/latest' "$TEMP_DIR/anonymous")" -eq 1
if grep -Fq 'example.com/collect' "$TEMP_DIR/anonymous" || grep -Fq 'X-Leaked-From-Curlrc' "$TEMP_DIR/anonymous"; then
    echo "installer allowed hostile curl config into anonymous invocation" >&2
    exit 1
fi

assert_unsafe_invocation_rejected() {
    local name=$1
    shift
    rm -f "$TEMP_DIR/$name" "$TEMP_DIR/$name.stdin"
    if CURL_ARGS_LOG="$TEMP_DIR/$name" PATH="$TEMP_DIR:$PATH" UPDATE_GITHUB_TOKEN="update-secret" \
        bash -c 'source "$1"; shift; github_api_curl "$@"' bash \
        "$ROOT_DIR/deploy/install.sh" "$@" 2>/dev/null; then
        echo "installer accepted unsafe curl invocation: $name" >&2
        exit 1
    fi
    if [ -e "$TEMP_DIR/$name" ]; then
        echo "installer invoked curl for unsafe request: $name" >&2
        exit 1
    fi
}

assert_unsafe_invocation_rejected non-api -s \
    "https://github.com/Redstonexs/sub2api/releases/download/v1/asset"
assert_unsafe_invocation_rejected mixed-host -s \
    "https://api.github.com/repos/Redstonexs/sub2api/releases/latest" \
    "https://example.com/collect"
assert_unsafe_invocation_rejected multiple-api -s \
    "https://api.github.com/repos/Redstonexs/sub2api/releases/latest" \
    "https://api.github.com/repos/Redstonexs/sub2api/releases"
assert_unsafe_invocation_rejected url-option -s --url \
    "https://example.com/collect" \
    "https://api.github.com/repos/Redstonexs/sub2api/releases/latest"

# Every installer release API request must use the scoped helper.
test "$(grep -c 'github_api_curl .*https://api.github.com/' "$ROOT_DIR/deploy/install.sh")" -eq 3

# Asset and checksum downloads must use the secure download helper.
grep -Fq 'github_download_curl "$download_url" -o "$TEMP_DIR/$archive_name"' "$ROOT_DIR/deploy/install.sh"
grep -Fq 'github_download_curl --max-size 1048576 "$checksum_url" -o "$TEMP_DIR/checksums.txt"' "$ROOT_DIR/deploy/install.sh"
grep -Fq 'print_error "$(msg '\''checksum_not_found'\'')"' "$ROOT_DIR/deploy/install.sh"
# `--max-filesize` only protects known Content-Length. Unknown-length streams
# must also be bounded before the temporary file can grow without limit.
grep -Fq 'ulimit -f "$file_blocks"' "$ROOT_DIR/deploy/install.sh"
grep -Fq 'actual_size=$(wc -c < "$tmpfile" 2>/dev/null || printf '\''0'\'')' "$ROOT_DIR/deploy/install.sh"

# ---------------------------------------------------------------------------
# github_download_curl contract tests
# ---------------------------------------------------------------------------

# Create a mock curl that supports redirect simulation and transfer-time
# size enforcement for the download helper.
# Uses a counter file to only return the redirect once (avoid infinite loop).
cat > "$TEMP_DIR/curl" <<'CURLSCRIPT'
#!/bin/bash
output_file=""
max_filesize=""
fmt_redirect=""
fmt_http_code=""
counter_file="$HOME/.curl_mock_count"
expecting_output=0
expecting_format=0
expecting_maxsize=0
for arg do
    case "$arg" in
        -o) expecting_output=1 ;;
        -w) expecting_format=1 ;;
        --max-filesize) expecting_maxsize=1 ;;
        -q|-sS|--connect-timeout|--max-time) ;;
        --) ;;
        *)
            if [ "${expecting_maxsize:-0}" -eq 1 ]; then
                max_filesize="$arg"
                expecting_maxsize=0
            elif [ "${expecting_output:-0}" -eq 1 ]; then
                output_file="$arg"
                expecting_output=0
            elif [ "${expecting_format:-0}" -eq 1 ]; then
                fmt_format="$arg"
                expecting_format=0
            fi
            ;;
    esac
done
# Read and increment counter
count=0
[ -f "$counter_file" ] && count=$(cat "$counter_file" 2>/dev/null || echo 0)
echo "$((count + 1))" > "$counter_file"

# Determine what to write for the -w format
case "${fmt_format:-}" in
    *redirect_url*)
        # Only emit redirect on the first call
        if [ "$count" -eq 0 ] && [ -n "${CURL_REDIRECT:-}" ]; then
            printf '%s' "$CURL_REDIRECT"
        fi
        ;;
    *http_code*)
        printf '%s' '200'
        ;;
esac

# Handle body output
if [ -n "${CURL_LARGE_OUTPUT:-}" ]; then
    # Check if --max-filesize would be exceeded
    if [ -n "$max_filesize" ] && [ "$max_filesize" -le 2097152 ]; then
        # Simulate curl --max-filesize: exit 63, don't write partial file
        if [ -n "$output_file" ] && [ "$output_file" != "/dev/null" ]; then
            rm -f "$output_file"
        fi
        exit 63
    fi
    dd if=/dev/zero bs=1048576 count=2 of="$output_file" 2>/dev/null
else
    printf 'ok' > "$output_file"
fi
CURLSCRIPT
chmod +x "$TEMP_DIR/curl"

# Clean any previous counter file
rm -f "$HOME/.curl_mock_count"

# Helper to run github_download_curl in a clean environment
run_download_curl() {
    local outfile="$TEMP_DIR/dl_out"
    rm -f "$outfile" "$HOME/.curl_mock_count"
    PATH="$TEMP_DIR:$PATH" UPDATE_GITHUB_TOKEN= GITHUB_TOKEN= GH_TOKEN= \
        bash -c 'source "$1"; github_download_curl "$2" -o "$3" ${4:+--max-size "$4"}' bash \
        "$ROOT_DIR/deploy/install.sh" "$1" "$outfile" "${2:-}" >/dev/null 2>&1
}

# 1. Foreign redirect is rejected
export CURL_REDIRECT="https://evil.example.com/trojan"
if run_download_curl "https://github.com/Redstonexs/sub2api/releases/download/v1/asset.tar.gz" 2>/dev/null; then
    echo "installer accepted forbidden redirect to evil.example.com" >&2
    exit 1
fi

# 2. Allowed redirect (githubusercontent.com) is accepted
export CURL_REDIRECT="https://objects.githubusercontent.com/release-asset/12345"
if ! run_download_curl "https://github.com/Redstonexs/sub2api/releases/download/v1/asset.tar.gz" 2>/dev/null; then
    echo "installer rejected legitimate GitHub redirect" >&2
    exit 1
fi

# 3. No redirect — normal download succeeds
export CURL_REDIRECT=""
if ! run_download_curl "https://github.com/Redstonexs/sub2api/releases/download/v1/asset.tar.gz" 2>/dev/null; then
    echo "installer rejected direct download" >&2
    exit 1
fi

# 4. Size cap: oversize download is rejected (1 MiB max)
export CURL_LARGE_OUTPUT=1 CURL_REDIRECT=""
if run_download_curl "https://github.com/Redstonexs/sub2api/releases/download/v1/checksums.txt" 1048576 2>/dev/null; then
    echo "installer accepted oversized download (2 MiB > 1 MiB cap)" >&2
    exit 1
fi

# 5. Size cap: small download is accepted
export CURL_LARGE_OUTPUT= CURL_REDIRECT=""
if ! run_download_curl "https://github.com/Redstonexs/sub2api/releases/download/v1/checksums.txt" 1048576 2>/dev/null; then
    echo "installer rejected small download under cap" >&2
    exit 1
fi

echo "install GitHub token checks passed"
