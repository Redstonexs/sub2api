#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
caddyfile="$repo_root/deploy/Caddyfile"

# Normalize a comment-stripped Caddyfile: collapse runs of whitespace and
# drop empty lines. Deliberately no quote/brace re-interpretation -- the guard
# below is a strict source-contract whitelist, not a Caddyfile parser.
normalize_config_lines() {
	awk '
		NF > 0 {
			for (field = 1; field <= NF; field++) {
				printf "%s%s", (field == 1 ? "" : " "), $field
			}
			print ""
		}
	'
}

# Keep one canonical encode block instead of reimplementing Caddy matcher
# semantics. These `header Content-Type ...` matcher lines are the only
# `header` directives the Caddyfile may contain.
expected_encode_block=$(cat <<'EOF'
encode {
zstd
gzip 6
minimum_length 256
match {
header Content-Type text/css*
header Content-Type text/csv*
header Content-Type text/html*
header Content-Type text/javascript*
header Content-Type text/markdown*
header Content-Type text/plain*
header Content-Type text/xml*
header Content-Type application/json*
header Content-Type application/javascript*
header Content-Type application/xml*
header Content-Type application/rss+xml*
header Content-Type image/svg+xml*
}
}
EOF
)

# Strict source-contract whitelist, not Caddy grammar equivalence. The guard
# reads the raw Caddyfile on stdin and rejects anything outside the exact
# contract the deployment depends on:
#   * Caddy environment substitutions `{$...}` anywhere in the raw source,
#     including comments: Caddy expands `{$...}` before lexing comments, so a
#     newline-bearing env value could inject a directive after these checks.
#     Runtime placeholders such as {remote_host}, {scheme}, {host}, and
#     {err.*} do not use `$` and remain allowed.
#   * any non-ASCII byte in the active (comment-stripped) config: Caddy
#     recognizes Unicode whitespace as token separators, so quoted or
#     non-ASCII directive-name forms must be excluded wholesale. Comments may
#     still be non-ASCII because they are removed first.
#   * any quote or backtick in the active config except the one exact,
#     normalized `respond "{err.status_code} {err.status_text}"` line in
#     handle_errors (blocks quoted/backtick directive-name tricks).
#   * any backslash in the active config, trailing or otherwise -- line
#     continuations and escapes would hide text from this line-oriented
#     inspection (an intentional configuration restriction),
#   * any `header_down` directive anywhere -- it rewrites every response and
#     has no matcher syntax to scope it,
#   * any `header` directive outside the exact canonical encode block above,
#     which is the only place `header Content-Type ...` matchers are allowed,
#   * any `import` directive,
#   * any `flush_interval` directive.
# `header_up` and `request_header` only affect the upstream request and are
# deliberately exempt. Prints each violation; exits 1 if any were found.
config_policy_violations() {
	raw_config=$(cat)
	config=$(printf '%s\n' "$raw_config" | LC_ALL=C sed 's/[[:space:]]*#.*$//')
	normalized=$(printf '%s\n' "$config" | normalize_config_lines)

	bad=0

	# Scanned on the raw source, before comment stripping: an env value could
	# contain a newline and inject a directive, and Caddy expands {$...} even
	# inside comments.
	if printf '%s\n' "$raw_config" | grep -Eq '[{][$]'; then
		echo "Caddyfile must not use Caddy environment substitution ({$...}) anywhere, including comments; runtime placeholders like {remote_host} and {err.*} are fine"
		bad=1
	fi

	# Any backslash, not just a trailing one: continuations and escapes both
	# hide text from this line-oriented whitelist.
	if printf '%s\n' "$config" | grep -Eq '[\\]'; then
		echo "Caddyfile must not use backslashes (line continuations or escapes; intentional strict-policy restriction)"
		bad=1
	fi

	# Pure-ASCII active config: Unicode whitespace would otherwise serve as a
	# separator for directive names this first-token whitelist cannot see.
	if printf '%s\n' "$config" | LC_ALL=C grep -Eq '[^[:print:][:space:]]'; then
		echo "Caddyfile active config must be pure ASCII: non-ASCII bytes such as Unicode whitespace are rejected by this strict source-contract whitelist"
		bad=1
	fi

	# Quotes and backticks are stripped by Caddy during lexing, so they must
	# not smuggle a directive name past the first-token checks. The single
	# permitted exception is the exact canonical handle_errors respond line.
	if ! printf '%s\n' "$normalized" | awk '
		/"|`/ && $0 != "respond \"{err.status_code} {err.status_text}\"" { bad = 1 }
		END { exit bad }
	'; then
		echo "Caddyfile must not use quotes or backticks, except the canonical handle_errors respond line"
		bad=1
	fi

	if printf '%s\n' "$normalized" | grep -Eq '^import([[:space:]]|$)'; then
		echo "Caddyfile must not import configuration outside this canonical policy check"
		bad=1
	fi

	if printf '%s\n' "$normalized" | grep -Eiq '^flush_interval([[:space:]]|$)'; then
		echo "Caddyfile must leave flush_interval unset so SSE auto-flushing and client cancellation remain intact"
		bad=1
	fi

	# The canonical encode block is the only place `header` may appear. If an
	# encode block exists, require it to match the canonical block exactly,
	# then drop it so only out-of-block directives are inspected below.
	encode_count=$(printf '%s\n' "$normalized" | awk '$1 == "encode" { count++ } END { print count + 0 }')
	if [ "$encode_count" -gt 0 ]; then
		actual_encode_block=$(printf '%s\n' "$normalized" | awk '
			$1 == "encode" { in_block = 1 }
			in_block {
				print
				for (field = 1; field <= NF; field++) {
					if ($field == "{") depth++
					if ($field == "}") depth--
				}
				if (depth == 0) exit
			}
		')
		if [ "$actual_encode_block" != "$expected_encode_block" ]; then
			echo "Caddyfile encode block must keep the canonical non-SSE compression policy"
			bad=1
		fi
		remainder=$(printf '%s\n' "$normalized" | awk '
			$1 == "encode" { in_block = 1 }
			{
				if (in_block) {
					for (field = 1; field <= NF; field++) {
						if ($field == "{") depth++
						if ($field == "}") depth--
					}
					if (depth == 0) {
						in_block = 0
					}
					next
				}
				print
			}
		')
	else
		remainder=$normalized
	fi

	if printf '%s\n' "$remainder" | grep -Eq '^(header|header_down)([[:space:]]|$)'; then
		echo "Caddyfile must not alter origin response headers: header_down is always rejected, and header is allowed only for the canonical encode Content-Type matchers (header_up and request_header are request-only and exempt)"
		bad=1
	fi

	if [ "$bad" -ne 0 ]; then
		return 1
	fi
}

if violations=$(config_policy_violations < "$caddyfile"); then
	:
else
	printf '%s\n' "$violations" >&2
	exit 1
fi

active_config=$(sed 's/[[:space:]]*#.*$//' "$caddyfile")
normalized_config=$(printf '%s\n' "$active_config" | normalize_config_lines)

if ! printf '%s\n' "$normalized_config" | grep -Eq '^reverse_proxy localhost:8080([[:space:]]|$)'; then
	echo "Caddyfile must continue proxying all application routes to localhost:8080" >&2
	exit 1
fi

encode_directive_count=$(printf '%s\n' "$normalized_config" | awk '$1 == "encode" { count++ } END { print count + 0 }')
if [ "$encode_directive_count" -ne 1 ]; then
	echo "Caddyfile must contain exactly one explicit encode block" >&2
	exit 1
fi

# Deterministic self-check: prove the strict guard above by running fixture
# snippets through the exact same normalization and guard code used for the
# real Caddyfile. The Caddyfile itself is never modified here. Every fixture
# is fed to check_case via a heredoc redirect (never a pipeline) so the case
# counters update in the same shell that runs the function.
self_check_passed=0
self_check_total=0
check_case() {
	# $1 = case name, $2 = expected outcome (rejected|accepted); stdin = snippet
	self_check_total=$((self_check_total + 1))
	name=$1
	expected=$2
	status=0
	violations=$(config_policy_violations) || status=$?
	case "$expected" in
		rejected)
			if [ "$status" -ne 0 ]; then
				self_check_passed=$((self_check_passed + 1))
				echo "self-check ok: $name rejected"
			else
				echo "self-check FAIL: $name accepted but must be rejected" >&2
				exit 1
			fi
			;;
		accepted)
			if [ "$status" -eq 0 ]; then
				self_check_passed=$((self_check_passed + 1))
				echo "self-check ok: $name accepted"
			else
				echo "self-check FAIL: $name rejected but must be accepted" >&2
				exit 1
			fi
			;;
		*)
			echo "self-check internal error: unknown expectation '$expected'" >&2
			exit 1
			;;
	esac
}

echo "self-check: exercising the strict response-header source-contract whitelist"

# Any response-side `header` directive is rejected, whatever matcher form or
# field it carries.
check_case "response header field" rejected <<'EOF'
header Cache-Control no-store
EOF
check_case "response header wildcard matcher" rejected <<'EOF'
header * Cache-Control no-store
EOF
check_case "response header named matcher" rejected <<'EOF'
header @static Cache-Control no-store
EOF
check_case "response header path matcher" rejected <<'EOF'
header /assets/* Cache-Control no-store
EOF
check_case "response header block" rejected <<'EOF'
header {
	Cache-Control no-store
}
EOF

# `header_down` is rejected unconditionally -- it rewrites every response, even
# when the field or value does not look cache-related.
check_case "header_down cache field" rejected <<'EOF'
reverse_proxy localhost:8080 {
	header_down Cache-Control no-store
}
EOF
check_case "header_down non-cache field" rejected <<'EOF'
reverse_proxy localhost:8080 {
	header_down X-Debug 1
}
EOF
check_case "header_down value spelling of Cache-Control" rejected <<'EOF'
reverse_proxy localhost:8080 {
	header_down X-Debug Cache-Control
}
EOF

# Caddy strips quotes and backticks during lexing, so quoted or backticked
# directive names must be rejected by the quote/backtick whitelist even
# though they would evade the first-token checks.
check_case "quoted header directive" rejected <<'EOF'
"header" Cache-Control no-store
EOF
check_case "backtick header directive" rejected <<'EOF'
`header` Cache-Control no-store
EOF
check_case "quoted header_down directive" rejected <<'EOF'
"header_down" Cache-Control no-store
EOF
check_case "backtick header_down directive" rejected <<'EOF'
`header_down` Cache-Control no-store
EOF

# Strict policy: `header` is allowed only inside the canonical encode block,
# so even valid matcher-block usage elsewhere is rejected.
check_case "header matcher inside named matcher block" rejected <<'EOF'
@static {
	header Content-Type text/css*
}
EOF
check_case "header matcher inside match block" rejected <<'EOF'
route {
	match {
		header Content-Type text/css*
	}
}
EOF

# Request-side directives only affect the upstream request and are exempt.
check_case "header_up cache field" accepted <<'EOF'
reverse_proxy localhost:8080 {
	header_up Cache-Control no-store
}
EOF
check_case "request_header cache field" accepted <<'EOF'
reverse_proxy localhost:8080 {
	request_header Cache-Control no-store
}
EOF
check_case "header_up runtime placeholder" accepted <<'EOF'
reverse_proxy localhost:8080 {
	header_up X-Real-IP {remote_host}
}
EOF

# Caddy environment substitutions are rejected on the raw source (they could
# inject directives or fields after these checks); runtime placeholders stay
# allowed.
check_case "environment substitution in argument" rejected <<'EOF'
reverse_proxy {$UPSTREAM} {
	header_up X-Real-IP {remote_host}
}
EOF
check_case "environment substitution with default" rejected <<'EOF'
respond {$GATEWAY_URL:https://api.example.com}
EOF
check_case "environment substitution in comment" rejected <<'EOF'
# upstream must never be overridden via {$UPSTREAM}
reverse_proxy localhost:8080
EOF
check_case "runtime placeholders" accepted <<'EOF'
header_up X-Forwarded-Proto {scheme}
header_up X-Forwarded-Host {host}
handle_errors {
	respond "{err.status_code} {err.status_text}"
}
EOF

# The exact canonical handle_errors respond line is the one permitted quote
# use; any other quote-carrying active line fails.
check_case "canonical quoted respond line" accepted <<'EOF'
handle_errors {
	respond "{err.status_code} {err.status_text}"
}
EOF
check_case "additional quoted respond line" rejected <<'EOF'
handle_errors {
	respond "{err.status_code} {err.status_text}"
	respond "extra response"
}
EOF

# Any backslash in active config is rejected, not just a trailing one --
# continuations and escapes would both hide text from this line-oriented
# whitelist.
check_case "line continuation" rejected <<'EOF'
header_up X-Real-IP \
	{remote_host}
EOF
check_case "backslash in value" rejected <<'EOF'
header_up X-Custom foo\bar
EOF

# Caddy recognizes Unicode whitespace (here: U+00A0 no-break space, produced
# from ASCII octal escapes) as a token separator, so any non-ASCII byte in
# active config is rejected outright.
check_case "unicode whitespace separator" rejected <<EOF
header$(printf '\302\240')Cache-Control no-store
EOF

# The exact canonical encode block, and only it, may carry `header` matchers.
# An expandable heredoc keeps this fixture off a pipeline so the case counter
# updates in the calling shell.
check_case "canonical encode block with header matchers" accepted <<EOF
$expected_encode_block
EOF

if [ "$self_check_passed" -ne "$self_check_total" ]; then
	echo "self-check FAIL: $self_check_passed/$self_check_total cases passed" >&2
	exit 1
fi
echo "self-check: $self_check_passed/$self_check_total cases passed"

echo "Caddyfile preserves backend cache policy, SSE streaming, and non-SSE compression"
