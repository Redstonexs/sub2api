#!/bin/bash

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
DEPLOY_DIR="${ROOT_DIR}/deploy"

fail() {
    printf 'deployment consistency test: %s\n' "$*" >&2
    exit 1
}

# ---------------------------------------------------------------------------
# install.sh print_completion contract (no live install needed)
# ---------------------------------------------------------------------------
install_script="${DEPLOY_DIR}/install.sh"

# install.sh guards its entrypoint, so source the tested functions directly.
source "${install_script}"
LANG_CHOICE=en

if ! bash "${install_script}" --help >/dev/null; then
    fail "install.sh --help must run without a terminal"
fi

# A public SERVER_HOST and a non-default SERVER_PORT must still produce a
# loopback-only wizard URL, a tunnel on the selected port, an INSTALL_DIR-based
# token path, and safe proxy timing — and must never print the token.
out=$(SERVER_HOST=0.0.0.0 SERVER_PORT=8443 INSTALL_DIR=/opt/sub2api print_completion)

grep -Fq 'http://127.0.0.1:8443' <<<"${out}" || \
    fail "completion must show the loopback wizard URL with the selected port"
if grep -Fq 'http://0.0.0.0' <<<"${out}"; then
    fail "completion must not show a 0.0.0.0 wizard URL"
fi
grep -Fq 'ssh -L 8443:localhost:8443' <<<"${out}" || \
    fail "completion must give the SSH tunnel on the selected port"
if grep -Fq 'ssh -L 8080:localhost:8080' <<<"${out}"; then
    fail "completion must not hardcode port 8080 in the SSH tunnel"
fi
grep -Fq 'sudo cat /opt/sub2api/data/.bootstrap_token' <<<"${out}" || \
    fail "completion must show the token path under INSTALL_DIR"
grep -Fq 'regardless of the SERVER_HOST setting' <<<"${out}" || \
    fail "completion must state the wizard is loopback regardless of SERVER_HOST"
grep -Fq 'only AFTER setup completes' <<<"${out}" || \
    fail "completion must state TLS proxy timing after setup"
if grep -Eq '[0-9a-f]{64}' <<<"${out}"; then
    fail "completion must not print the bootstrap token"
fi

# A custom INSTALL_DIR and the default port must be reflected in the output.
out2=$(SERVER_HOST=127.0.0.1 SERVER_PORT=8080 INSTALL_DIR=/custom/sub2api print_completion)
grep -Fq 'http://127.0.0.1:8080' <<<"${out2}" || \
    fail "completion must show the loopback URL for the default port"
grep -Fq 'ssh -L 8080:localhost:8080' <<<"${out2}" || \
    fail "completion must show the tunnel on the default port"
grep -Fq 'sudo cat /custom/sub2api/data/.bootstrap_token' <<<"${out2}" || \
    fail "completion must use the configured INSTALL_DIR for the token path"

# Source contract: the installer must interpolate the selected values and must
# not fetch or display a public IP for the loopback-only wizard.
if grep -Fq 'get_public_ip' "${install_script}"; then
    fail "install.sh must not fetch a public IP for the loopback-only wizard"
fi
if grep -Fq 'PUBLIC_IP' "${install_script}"; then
    fail "install.sh must not reference PUBLIC_IP"
fi
grep -Fq 'http://127.0.0.1:${SERVER_PORT}' "${install_script}" || \
    fail "install.sh wizard URL must be loopback with the selected port"
grep -Fq 'ssh -L ${SERVER_PORT}:localhost:${SERVER_PORT}' "${install_script}" || \
    fail "install.sh tunnel must use the selected SERVER_PORT"
grep -Fq 'sudo cat ${INSTALL_DIR}/data/.bootstrap_token' "${install_script}" || \
    fail "install.sh token path must use INSTALL_DIR"

# ---------------------------------------------------------------------------
# config.example.yaml — server.host security default
# ---------------------------------------------------------------------------
config_example="${DEPLOY_DIR}/config.example.yaml"
grep -Fq 'host: "127.0.0.1"' "${config_example}" || \
    fail "config.example.yaml must default server.host to 127.0.0.1"
if grep -Fq 'host: "0.0.0.0"' "${config_example}"; then
    fail "config.example.yaml must not default server.host to 0.0.0.0"
fi
grep -Fq 'only after local setup/login succeeds' "${config_example}" || \
    fail "config.example.yaml must document the proxy-guarded/public override timing"

# ---------------------------------------------------------------------------
# .env.example — BIND_HOST guidance
# ---------------------------------------------------------------------------
env_example="${DEPLOY_DIR}/.env.example"
grep -Fq 'BIND_HOST=127.0.0.1' "${env_example}" || \
    fail ".env.example must default BIND_HOST to 127.0.0.1"
grep -Fq 'Keep this for a same-host TLS reverse' "${env_example}" || \
    fail ".env.example must keep loopback for a same-host TLS proxy or SSH tunnel"
grep -Fq 'direct-LAN exception' "${env_example}" || \
    fail ".env.example must describe 0.0.0.0 as a direct-LAN exception"
grep -Fq 'only after' "${env_example}" || \
    fail ".env.example must gate 0.0.0.0 behind local setup/login"
if grep -Fq 'then set BIND_HOST=0.0.0.0' "${env_example}"; then
    fail ".env.example must not instruct setting BIND_HOST=0.0.0.0 after choosing a proxy/tunnel"
fi

# ---------------------------------------------------------------------------
# apple-container.sh — BIND_HOST fallback
# ---------------------------------------------------------------------------
apple_script="${DEPLOY_DIR}/apple-container.sh"
grep -Fq 'read_env_value BIND_HOST 127.0.0.1' "${apple_script}" || \
    fail "apple-container.sh must default BIND_HOST to 127.0.0.1"
if grep -Fq 'read_env_value BIND_HOST 0.0.0.0' "${apple_script}"; then
    fail "apple-container.sh must not default BIND_HOST to 0.0.0.0"
fi

# ---------------------------------------------------------------------------
# deploy/README.md — systemd override example and wizard loopback behavior
# ---------------------------------------------------------------------------
deploy_readme="${DEPLOY_DIR}/README.md"
grep -Fq 'Environment=SERVER_PORT=8080' "${deploy_readme}" || \
    fail "deploy/README.md systemd override example must use the default port 8080"
if grep -Fq 'Environment=SERVER_PORT=3000' "${deploy_readme}"; then
    fail "deploy/README.md must not show a stale SERVER_PORT=3000 override"
fi
grep -Fq 'always listens on loopback' "${deploy_readme}" || \
    fail "deploy/README.md must explain the wizard loopback behavior"
grep -Fq 'regardless of the `SERVER_HOST` setting' "${deploy_readme}" || \
    fail "deploy/README.md must state the wizard loopback is independent of SERVER_HOST"

# ---------------------------------------------------------------------------
# APPLE_CONTAINER.md — BIND_HOST default/override/access documentation
# ---------------------------------------------------------------------------
apple_doc="${DEPLOY_DIR}/APPLE_CONTAINER.md"
grep -Fq 'defaults to `127.0.0.1`' "${apple_doc}" || \
    fail "APPLE_CONTAINER.md must document the BIND_HOST loopback default"
grep -Fq 'direct-LAN exception' "${apple_doc}" || \
    fail "APPLE_CONTAINER.md must document the direct-LAN exception"
grep -Fq 'only after local auto-setup/login succeeds' "${apple_doc}" || \
    fail "APPLE_CONTAINER.md must gate the public bind behind local setup/login"

# ---------------------------------------------------------------------------
# DOCKER.md — same truth as .env.example / APPLE_CONTAINER.md
# ---------------------------------------------------------------------------
docker_doc="${DEPLOY_DIR}/DOCKER.md"
grep -Fq 'direct LAN access behind a firewall' "${docker_doc}" || \
    fail "DOCKER.md must frame BIND_HOST=0.0.0.0 as a direct-LAN exception"
grep -Fq 'until auto-setup completes and local login succeeds' "${docker_doc}" || \
    fail "DOCKER.md must gate public publishing behind auto-setup and local login"

# ---------------------------------------------------------------------------
# sub2api.service — loopback security default
# ---------------------------------------------------------------------------
service_file="${DEPLOY_DIR}/sub2api.service"
grep -Fq 'Environment=SERVER_HOST=127.0.0.1' "${service_file}" || \
    fail "sub2api.service must default SERVER_HOST to 127.0.0.1"

printf 'deployment consistency test passed\n'
