#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

fail() {
  printf 'docs source-contract test: %s\n' "$1" >&2
  exit 1
}

# Extract a markdown section starting at the first line matching $2 up to the
# next top-level (###) heading.
extract_section() {
  file=$1
  start=$2
  awk -v start="$start" '
    $0 ~ start { in_section = 1; next }
    in_section && /^### / { exit }
    in_section { print }
  ' "$file"
}

# ---------------------------------------------------------------------------
# deploy/DOCKER.md — canonical Compose contract
# ---------------------------------------------------------------------------
docker_doc=deploy/DOCKER.md

# Reject stale, unsupported, or unsafe patterns. Connection URLs are rejected
# only in their usage form (VAR=...) so the explanatory "not supported"
# sentence is allowed.
for pattern in 'DATABASE_URL=' 'REDIS_URL=' 'POSTGRES_PASSWORD=postgres'; do
  if grep -qF "$pattern" "$docker_doc"; then
    fail "$docker_doc must not contain unsupported/unsafe pattern: $pattern"
  fi
done

# Prove the expected deployment contract terms.
for pattern in \
  'AUTO_SETUP' \
  'ADMIN_PASSWORD' \
  'DATABASE_HOST' \
  'DATABASE_PASSWORD' \
  'REDIS_HOST' \
  'mode-600' \
  'never printed' \
  '127.0.0.1' \
  'TLS reverse proxy' \
  'until auto-setup completes'
do
  if ! grep -qF "$pattern" "$docker_doc"; then
    fail "$docker_doc must document expected term: $pattern"
  fi
done

# Compose users must not be directed to a setup wizard; the doc must instead
# state explicitly that there is none.
if grep -qE 'reach the Setup Wizard|access the Setup Wizard|open the Setup Wizard|visit the Setup Wizard' "$docker_doc"; then
  fail "$docker_doc must not direct Compose users to a setup wizard"
fi
if ! grep -qF 'no setup wizard' "$docker_doc"; then
  fail "$docker_doc must state that Compose deployments have no setup wizard"
fi
if grep -qE 'docker run.*(-e|--env).*BIND_HOST|(-e|--env)[[:space:]]+BIND_HOST' "$docker_doc"; then
  fail "$docker_doc must not present BIND_HOST as a docker run setting"
fi

# ---------------------------------------------------------------------------
# Root READMEs (EN/ZH/JA) — Compose deployment section
# ---------------------------------------------------------------------------
# Each tuple: file, section-start regex, wizard-directing phrase to reject,
# wizard-negation phrase to prove, generated-secrets line marker,
# no-secrets-printed phrase, login-directing phrase, stale log-claim phrase.
check_readme() {
  file=$1
  section_start=$2
  wizard_direct=$3
  wizard_negation=$4
  secrets_marker=$5
  no_print=$6
  login=$7
  logs_claim=$8

  section=$(extract_section "$file" "$section_start")

  if printf '%s\n' "$section" | grep -qF "$wizard_direct"; then
    fail "$file Compose section must not direct users to the setup wizard"
  fi
  if ! printf '%s\n' "$section" | grep -qF "$wizard_negation"; then
    fail "$file Compose section must distinguish auto-setup from the binary setup wizard"
  fi
  if ! printf '%s\n' "$section" | grep -qF 'AUTO_SETUP'; then
    fail "$file Compose section must document AUTO_SETUP"
  fi
  if printf '%s\n' "$section" | grep -qF "$logs_claim"; then
    fail "$file Compose section must not claim credentials are printed in logs"
  fi

  secrets_line=$(printf '%s\n' "$section" | grep -F "$secrets_marker" | head -n 1)
  if [ -z "$secrets_line" ]; then
    fail "$file Compose section must describe the generated-secrets list"
  fi
  case "$secrets_line" in
    *ADMIN_PASSWORD*) ;;
    *) fail "$file generated-secrets list must include ADMIN_PASSWORD" ;;
  esac

  if ! printf '%s\n' "$section" | grep -qF "$no_print"; then
    fail "$file Compose section must state that no secrets are printed"
  fi
  if ! printf '%s\n' "$section" | grep -qF "$login"; then
    fail "$file Compose section must direct users to the login/application"
  fi
  if ! grep -qE '^ADMIN_PASSWORD=' "$file"; then
    fail "$file must list ADMIN_PASSWORD as an uncommented required .env entry"
  fi
  if grep -qE '^(POSTGRES_PASSWORD|JWT_SECRET|TOTP_ENCRYPTION_KEY|ADMIN_PASSWORD)=[^[:space:]]+' "$file"; then
    fail "$file must not contain copy-paste secret placeholders"
  fi
}

check_readme README.md \
  '^### Method 2: Docker Compose' \
  'reach the Setup Wizard' \
  'no setup wizard' \
  'Generates secure secrets' \
  'nothing is printed to the terminal' \
  'sign in with' \
  'find it in logs'

check_readme README_CN.md \
  '^### 方式二：Docker Compose' \
  '访问设置向导' \
  '无需设置向导' \
  '自动生成安全密钥' \
  '终端不会打印任何密钥' \
  '登录即可' \
  '在日志中查找'

check_readme README_JA.md \
  '^### 方法2: Docker Compose' \
  'セットアップウィザードにアクセス' \
  'セットアップウィザードは不要' \
  'セキュアなシークレット' \
  'ターミナルには一切表示されません' \
  'ログインしてください' \
  'ログで確認'

# Standalone binary setup and Compose auto-setup are distinct paths. Check the
# entire README so a later section cannot contradict the Compose guidance.
check_readme_lifecycle() {
  file=$1
  wizard_only=$2
  bootstrap_token=$3
  ssh_tunnel=$4
  non_reset=$5

  if grep -qF "$wizard_only" "$file"; then
    fail "$file must not claim only the setup wizard creates the initial admin"
  fi
  for pattern in "$bootstrap_token" "$ssh_tunnel" "$non_reset"; do
    if ! grep -qF "$pattern" "$file"; then
      fail "$file must document lifecycle term: $pattern"
    fi
  done
}

check_readme_lifecycle README.md \
  'only created via the setup wizard' \
  'bootstrap token' \
  'SSH tunnel' \
  'does not reset'

check_readme_lifecycle README_CN.md \
  '只能通过 setup 向导创建' \
  'bootstrap token' \
  'SSH 隧道' \
  '不会重置'

check_readme_lifecycle README_JA.md \
  'セットアップウィザード経由でのみ作成' \
  'bootstrap token' \
  'SSH トンネル' \
  'リセットされません'

# ---------------------------------------------------------------------------
# docs/security-audit-2026-08-18.md — accurate password-enforcement statement
# ---------------------------------------------------------------------------
audit=docs/security-audit-2026-08-18.md

if ! grep -qF '8–128 characters' "$audit"; then
  fail "$audit must state the initial password enforcement as an 8–128 character validation"
fi
for pattern in 'weak admin passwords' 'weak password rejection'; do
  if grep -qF "$pattern" "$audit"; then
    fail "$audit must not use generic weak-password wording: $pattern"
  fi
done

printf 'docs source-contract test passed\n'
