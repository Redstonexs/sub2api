# Sub2API Docker Image

Sub2API is an AI API Gateway Platform for distributing and managing AI product subscription API quotas.

## Quick Start (Docker Compose)

The canonical deployment is Docker Compose with the bundled PostgreSQL and Redis containers. The image auto-initializes on first run (`AUTO_SETUP=true`): it applies database migrations, creates the admin account from `ADMIN_PASSWORD`, and writes its configuration under `/app/data`. There is no setup wizard for Compose deployments.

### 1. Prepare the environment

Download the deployment files and generate the mandatory secrets:

```bash
mkdir -p sub2api-deploy && cd sub2api-deploy
curl -sSL https://raw.githubusercontent.com/Redstonexs/sub2api/main/deploy/docker-deploy.sh | bash
```

`docker-deploy.sh` downloads `docker-compose.local.yml` (saved as `docker-compose.yml`) and `.env.example`, then generates `ADMIN_PASSWORD`, `POSTGRES_PASSWORD`, `JWT_SECRET`, and `TOTP_ENCRYPTION_KEY` into a mode-600 `.env` file. The secrets are **never printed**; keep the `.env` file private (owner-only read/write) and never commit or share it.

### 2. Start the stack

```bash
docker compose up -d
docker compose logs -f sub2api
```

On first run the application:

- connects to PostgreSQL and Redis,
- applies database migrations,
- creates the admin account using `ADMIN_PASSWORD` from `.env`,
- writes its configuration under `/app/data`.

### 3. Log in

Open `http://127.0.0.1:8080` and sign in with `ADMIN_EMAIL` (default `admin@sub2api.local`) and the `ADMIN_PASSWORD` from your `.env` file.

## Manual Compose Setup

If you prefer to configure `.env` by hand:

```bash
git clone https://github.com/Redstonexs/sub2api.git
cd sub2api/deploy
cp .env.example .env
chmod 600 .env
nano .env   # set ADMIN_PASSWORD and POSTGRES_PASSWORD (required)
```

Generate strong values with `openssl rand -hex 32` (or `openssl rand -hex 12` for the admin password). The Compose files require `POSTGRES_PASSWORD` and `ADMIN_PASSWORD` and fail fast at startup if either is missing. `JWT_SECRET` and `TOTP_ENCRYPTION_KEY` are strongly recommended so login sessions and 2FA survive container restarts.

Then start with either:

- `docker compose -f docker-compose.local.yml up -d` (local directories, easy migration), or
- `docker compose up -d` (named volumes).

## Startup and Database Recovery

Sub2API runs database migrations while starting. PostgreSQL may still be
recovering briefly after a host or Docker daemon restart. The application
retries transient PostgreSQL startup and connection errors with bounded
exponential backoff, then continues startup when the database is ready.
Permanent errors such as invalid credentials, migration checksum mismatches,
SQL errors, and incompatible data fail immediately.

The Compose deployment also checks PostgreSQL readiness with both `pg_isready`
and a simple SQL query. `depends_on: condition: service_healthy` helps order a
fresh Compose start, but application-level retries are still required when
Docker restores existing containers after a host restart.

## Environment Variables

The image is configured with discrete `DATABASE_*` and `REDIS_*` variables — connection URLs such as `DATABASE_URL`/`REDIS_URL` are **not** supported. The bundled Compose files map the variables for you; the standalone Compose file (`docker-compose.standalone.yml`) requires you to provide `DATABASE_HOST`, `DATABASE_PASSWORD`, `REDIS_HOST`, and `ADMIN_PASSWORD` for externally managed PostgreSQL and Redis instances.

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `AUTO_SETUP` | Enable first-run auto-initialization | Yes (Compose) | - |
| `ADMIN_PASSWORD` | Initial admin password (8–128 characters) | Yes | - |
| `ADMIN_EMAIL` | Initial admin email | No | `admin@sub2api.local` |
| `DATABASE_HOST` | PostgreSQL host | Yes | - |
| `DATABASE_PORT` | PostgreSQL port | No | `5432` |
| `DATABASE_USER` | PostgreSQL user | No | `sub2api` |
| `DATABASE_PASSWORD` | PostgreSQL password | Yes | - |
| `DATABASE_DBNAME` | PostgreSQL database name | No | `sub2api` |
| `DATABASE_SSLMODE` | PostgreSQL SSL mode | No | `disable` |
| `REDIS_HOST` | Redis host | Yes | - |
| `REDIS_PORT` | Redis port | No | `6379` |
| `REDIS_PASSWORD` | Redis password | No | *(empty)* |
| `JWT_SECRET` | JWT signing secret (fixed for persistent sessions) | Recommended | *(auto-generated)* |
| `TOTP_ENCRYPTION_KEY` | TOTP encryption key (fixed for persistent 2FA) | Recommended | *(auto-generated)* |
| `SERVER_HOST` | Container-internal listen address | No | `0.0.0.0` |
| `SERVER_PORT` | Container-internal listen port | No | `8080` |
| `SERVER_MODE` | Server mode (`release`/`debug`) | No | `release` |
| `BIND_HOST` | Host-side bind address for the published port | No | `127.0.0.1` |

## TLS / Remote Access

By default, the container port is published on `127.0.0.1` (loopback only). For production access from remote clients, **always use a TLS reverse proxy**:

- **Caddy** (recommended) — automatic HTTPS via Let's Encrypt
- **Nginx** — see `README.md` for the `underscores_in_headers on;` directive
- **SSH tunnel** — for ad-hoc access

**To override the bind address in Compose** (for example, for direct LAN access behind a firewall), set `BIND_HOST=0.0.0.0` in `.env`. For a custom `docker run` command, choose the host interface in the publish flag instead, for example `-p 127.0.0.1:8080:8080`; `BIND_HOST` is a Compose interpolation setting, not a container runtime setting. Do not publish the service externally or configure a reverse proxy until auto-setup completes and local login succeeds. The container-internal `SERVER_HOST` may remain `0.0.0.0` in all cases.

## Supported Architectures

- `linux/amd64`
- `linux/arm64`

## Tags

- `latest` - Latest stable release
- `x.y.z` - Specific version
- `x.y` - Latest patch of minor version
- `x` - Latest minor of major version

## Links

- [GitHub Repository](https://github.com/Redstonexs/sub2api)
- [Documentation](https://github.com/Redstonexs/sub2api#readme)
