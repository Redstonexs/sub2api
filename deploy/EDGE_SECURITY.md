# Edge and HTTP Ingress Security

Sub2API supports long-lived SSE and WebSocket requests. Protect the request
ingress without imposing a response `WriteTimeout`: a write deadline would
terminate healthy long generations and streams.

## Application defaults

- `server.max_header_bytes: 65536` limits HTTP/1 request headers to 64 KiB;
  Go maps it to the corresponding HTTP/2 header-list limit.
- `server.read_header_timeout: 10` bounds slow-header attacks. It does not
  limit request processing or response streaming.
- `server.max_request_body_size: 268435456` is the absolute 256 MiB safety net.
- `gateway.max_body_size: 268435456` remains available to multimodal, Gemini,
  image, video, and batch-image endpoints.
- `gateway.text_max_body_size: 33554432` limits the known pure-text
  `/embeddings` and `/alpha/search` endpoints to 32 MiB.
- H2C defaults to 50 concurrent streams per connection, a 2 MiB connection
  upload window, and a 512 KiB stream upload window.
- Invalid credential abuse is limited in process by trusted client IP (IPv6
  `/64`): 120 failures per 60 seconds followed by a 60-second block. This is a
  per-instance safety net; multi-instance enforcement still belongs at the
  load balancer, CDN, or WAF.

Do not add a single application-wide request semaphore: an SSE request may
legitimately occupy it for many minutes. Apply connection and unauthenticated
request controls at the edge; authenticated user/API-key concurrency remains
the application's responsibility.

## Trusted client IPs

`security.trust_forwarded_ip_for_api_key_acl` is disabled by default. While
enabled, raw forwarding headers take over client-IP resolution for logs and
security-sensitive paths, and any caller holding a valid API key can forge its
client IP to defeat that key's IP allowlist/denylist. Custom headers from
`security.forwarded_client_ip_headers` are checked in configured order before
the built-in `CF-Connecting-IP`, `X-Real-IP`, and `X-Forwarded-For` fallback.
Header names are case-insensitive, normalized when loaded, de-duplicated, and
limited to 16 unique valid HTTP field names. Header values must contain IP
literals; comma-separated values are supported, invalid entries are skipped,
and public addresses are preferred over private fallback addresses.

The list can be supplied in YAML or with the comma-separated environment
variable `SECURITY_FORWARDED_CLIENT_IP_HEADERS`; an explicitly empty environment
value clears YAML values. It is also editable from the admin security settings
and updates at runtime without a restart. A request snapshots the switch and
header list together, so one request cannot mix old and new settings. Custom
headers are ignored completely when the switch is disabled. In that mode Gin's
`server.trusted_proxies` chain is authoritative: configure only the exact
CIDR/IP addresses that connect directly to Sub2API. An explicit empty list
trusts no forwarded client IPs.

### Precedence

The switch is both a configuration key and an admin-editable setting stored in
the database, so the order matters:

1. **`config.yaml` or `SECURITY_TRUST_FORWARDED_IP_FOR_API_KEY_ACL`** — setting
   either one *pins* the value. It overrides the stored database row on every
   startup, rewrites that row so the admin UI shows the effective state, and is
   never touched by the legacy compatibility migration below. Use this to enforce
   the setting from a deployment manifest.
2. **The admin UI toggle** — authoritative whenever the key is *not* pinned.
   Applies immediately without a restart.
3. **The startup default (`false`)** — used only to seed a fresh installation.

Pinning logs a warning at startup whenever it overrides a differing stored value.
Because the pin re-asserts on every restart, a UI change made against a pinned
deployment survives only until the process restarts.

On the first upgrade to this mode, a legacy `false` value is changed to `true`
only when `server.trusted_proxies` was not explicitly configured and the key is
not pinned; explicit proxy policies remain in secure mode. New installations seed
the stored value from configuration and persist the configured custom header list
during database initialization. Existing installations
backfill a missing database value from the YAML configuration. A hidden
migration marker prevents later administrator changes from being overwritten.
If settings cannot be read or the persisted custom-header list is malformed,
the process fails closed to trusted-proxy mode with no custom headers. If a
migration write fails, the computed mode remains active for the current process
and startup records a warning.

Compatibility takeover accepts forwarded headers without validating the direct
peer, including any configured custom header. Protect the origin from direct
access while it is enabled. A CDN deployment must firewall the origin so only
the CDN or load balancer can reach it, and that proxy must overwrite every
trusted client-IP header rather than append an untrusted client value.

Example for a proxy on the same host:

```yaml
server:
  trusted_proxies:
    - 127.0.0.1/32
    - ::1/128
```

## Nginx baseline

Define shared zones in the `http` block. Tune rates to measured legitimate
traffic; the values below are conservative starting points, not universal
capacity targets.

```nginx
limit_conn_zone $binary_remote_addr zone=sub2api_conn:20m;
limit_req_zone  $binary_remote_addr zone=sub2api_auth:20m rate=5r/s;
limit_req_zone  $binary_remote_addr zone=sub2api_api:40m rate=30r/s;
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    client_header_timeout 10s;
    client_max_body_size 256m;
    large_client_header_buffers 4 16k;
    limit_conn sub2api_conn 40;

    location ~ ^/(auth|api/auth)/ {
        limit_req zone=sub2api_auth burst=10 nodelay;
        proxy_pass http://127.0.0.1:8080;
    }

    location ~ ^/(v1/)?(embeddings|alpha/search)$ {
        client_max_body_size 32m;
        limit_req zone=sub2api_api burst=60 nodelay;
        proxy_pass http://127.0.0.1:8080;
    }

    location / {
        limit_req zone=sub2api_api burst=60 nodelay;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_buffering off;
        proxy_request_buffering off;
        proxy_read_timeout 1800s;
        proxy_send_timeout 1800s;
        proxy_pass http://127.0.0.1:8080;
    }
}
```

If Nginx gzip is enabled in the `http` block, keep `text/event-stream` out of
`gzip_types` and do not use `gzip_types *` for Sub2API. The
`proxy_buffering off` setting above prevents proxy buffering, but it does not
disable the gzip response filter. Use an explicit list for ordinary responses:

```nginx
gzip on;
gzip_types text/plain text/css application/json application/javascript application/xml image/svg+xml;
```

If a shared global configuration cannot exclude SSE by content type, set
`gzip off;` in the locations serving streaming API routes. This leaves gzip
available for the web UI and static assets.

Do not use an incoming `$http_x_forwarded_for` value unless Nginx real-IP
processing is restricted to explicit trusted proxy CIDRs.

## Caddy and CDN

The bundled `deploy/Caddyfile` sets a 64 KiB header limit, a 10-second header
timeout, a 256 MiB absolute body limit, and overwrites forwarded addresses from
the TCP peer. It uses no response-header mutation directives — no `header`
outside the canonical `encode` block and no `header_down` anywhere. Its `encode`
middleware does make the expected encoding representation transformations: it
manages `Content-Encoding` and a correctly scoped `Vary: Accept-Encoding`, and
it may remove `Content-Length`/`Accept-Ranges`, adapt strong ETags, or
synthesize a `Content-Type`. The `header Content-Type` lines inside the
`encode` block's `match` are response matchers: they select which response
content types are compressed. They are not request matchers and are not a
response mutation. The file is therefore a direct-to-Caddy baseline. Do not use
its `{remote_host}` forwarding lines unchanged behind a CDN: all clients would
be attributed to a CDN egress address, collapsing rejection aggregation and the
invalid-auth limiter onto unrelated users.

The bundled Caddy configuration leaves `flush_interval` unset so Caddy can
automatically flush `text/event-stream` responses while still propagating
client cancellation upstream. Do not set it globally: positive values can add
streaming latency, while Caddy 2.6.2's special `-1` mode also causes
reverse-proxied requests to continue after clients disconnect. The
configuration uses an explicit response content-type list for compression. Do
not replace that list with `text/*` or the shorthand `encode gzip zstd`: both
match `text/event-stream` and can buffer SSE until the response ends. Keep
streaming responses uncompressed while retaining compression for the web UI,
JSON, and static assets.

For a CDN deployment, first firewall the origin so only current CDN egress
CIDRs can connect. Then configure those exact ranges as Caddy trusted proxies
and derive upstream headers from Caddy's parsed `{client_ip}`. For example:

```caddyfile
{
	servers {
		trusted_proxies static 192.0.2.0/24 2001:db8:1234::/48
		trusted_proxies_strict
		client_ip_headers CF-Connecting-IP X-Forwarded-For
	}
}

api.example.com {
	reverse_proxy 127.0.0.1:8080 {
		header_up X-Real-IP {client_ip}
		header_up X-Forwarded-For {client_ip}
	}
}
```

Replace the documentation ranges with the CDN's published, automatically
maintained egress ranges. `CF-Connecting-IP` is safe here only because direct
origin access is blocked and Caddy trusts only those TCP peers. Configure
Sub2API `server.trusted_proxies` with the Caddy address/private subnet so the
application accepts only Caddy's rewritten headers.

Caddy core does not provide a general request-rate limiter; use a trusted
CDN/WAF, a supported rate-limit module, or host firewall controls.

At a CDN/WAF, configure connection limits, header/body limits, bot challenges,
and per-IP/ASN rates before traffic reaches the origin. Allow origin ingress
only from CDN egress CIDRs or a private load balancer. Keep the application port
off the public Internet.

### Caching and cache policy

The Go origin is the sole owner of all client-visible response headers
relevant to delivery and cache policy: it emits `Cache-Control` and validators
itself. The edge must not use header rewrite rules to add, delete, or override
any such origin header; Caddy and any CDN must forward those headers verbatim
and must not add directives of their own. Compression behavior — including the
correctly scoped `Vary: Accept-Encoding` that Caddy's `encode` attaches to the
responses it actually encodes (see below) — is the explicit narrow exception.
For the bundled `deploy/Caddyfile` the constraint is absolute at the directive
level: it contains no response-header mutation directives — no `header` outside
the canonical `encode` matcher and no `header_down`. The `encode` middleware's
representation transformations are the expected exception. The origin's exact
behavior:

- **Release-owned, content-addressed assets** (`/assets/*-<hash>.*`) are served
  with `public, max-age=31536000, immutable`. A CDN may cache them long-term.
  Never create a file under the same path in `data/public`: the origin rejects
  overrides of immutable Vite-hashed assets because the file name is the
  content hash, so any "override" would be a lie. Do not purge these entries on
  a normal deploy — old HTML shells can still reference old hashes, and an
  immutable name is never served twice with different bytes.
- **Nonce-bearing dynamic HTML** (`/`, `/index.html`, and the SPA fallback)
  carries `private, no-store, no-cache, must-revalidate`. The nonce is
  per-response, so this body must never be stored or reused. Do not configure
  cache-everything rules, an edge minimum TTL, `s-maxage`, stale-while-
  revalidate or stale-if-error, or any CDN rule that stores this response.
- **Mutable root/static files and allowed `data/public` overrides** are served
  with `no-cache` plus origin validators (a strong `ETag` derived from content
  for release-owned files, `Last-Modified` for on-disk overrides). A browser or
  CDN may store them but must revalidate with the origin on every use; never
  impose a minimum TTL on these paths.
- **Unknown `/assets/` paths** are never served the SPA HTML shell. The origin
  returns an explicit `404` with `Cache-Control: no-store`, no static ETag,
  and a non-HTML body, so a browser, proxy, or CDN cannot store or
  negative-cache the response under an asset-style key. Do not configure edge
  rules that cache or synthesize responses for unknown asset paths.
- **APIs, auth, and user-specific routes** must be CDN cache-bypassed
  regardless of what each origin response happens to emit: there is no global
  origin API cache policy to rely on, and per-endpoint `Cache-Control` values
  vary and may be absent. Bypass caching for every application route outside
  the static namespace — `/api/`, `/v1/`, `/v1beta/`, `/backend-api/`,
  `/antigravity/`, `/setup/`, `/health`, `/models`, `/responses`,
  `/alpha/search`, `/images/`, `/videos/`, and auth/user-specific paths. Never
  add a generic edge cache rule for them; a blanket rule can store tokens,
  quota state, or one user's data for another.

Origin headers are authoritative:

- The origin is the sole owner of all client-visible response headers relevant
  to delivery and cache policy. Do not have Caddy or the CDN inject, strip, or
  override any of them on responses — `Cache-Control`, `Expires`,
  `Surrogate-Control`, `CDN-Cache-Control`, and every other delivery-relevant
  header. For the bundled Caddyfile the constraint is not limited to
  cache-related fields: it must not add, delete, or rewrite ANY response header
  via Caddy's response-side `header` directive (both `header <field>` and
  `header { ... }` forms, including the `+`/`-`/`?`/`>` field operations and
  trailing-colon spellings such as `-Cache-Control:`) or the reverse_proxy
  `header_down` subdirective. The strict guard in
  `deploy/test-caddyfile-cache.sh` rejects every such directive: any `header`
  outside the canonical `encode` block and every `header_down`, whatever field
  name or value it carries.
- The only `header` syntax permitted in the bundled Caddyfile is the
  `Content-Type` line inside the canonical `encode` block's `match`. It is a
  response matcher: it selects which response content types are compressed. It
  is not a response-header rewrite directive and does not violate the
  response-header policy above. There are no other `header` directives in the
  file.
- Caddy `header_up` and `request_header` modify only the upstream request; they
  are request-side and are deliberately exempt from the guard.
- The guard also rejects Caddy config dynamic environment substitutions
  (`{$...}`), `import` directives, and line continuations so static checks
  cannot be bypassed by deferring or splitting header directives elsewhere in
  the configuration. Runtime placeholders such as `{remote_host}` remain
  supported and are unaffected.
- Altering this policy means updating the guard in
  `deploy/test-caddyfile-cache.sh` and performing a cache/security review
  before deployment.
- Compression must respect `Vary: Accept-Encoding`. Caddy's `encode`
  subdirective adds `Vary: Accept-Encoding` only to the responses it actually
  encodes (and to 304 Not Modified responses), not to every response, and it
  does not remove an origin-supplied `Vary`. A CDN must likewise keep the
  header when serving compressed variants and must not strip or broaden an
  origin `Vary`.
- Never configure high-cardinality `Vary` keys such as `Cookie`,
  `User-Agent`, or `Referer`: they fragment cache entries, can leak a
  user-specific copy across tenants, and invite cache-poisoning probes.

Cache-poisoning controls:

- Cache only canonical hostname/path GET/HEAD representations of the static
  assets above — never key, store, or reflect untrusted forwarding or rewrite
  headers such as `X-Forwarded-Host`, `X-Original-URL`, or `X-Rewrite-URL`.
- Firewall the origin to the CDN/load balancer (as above) and canonicalize the
  host and path at the CDN: normalize scheme and host case, collapse dot
  segments, and strip query strings the origin does not use before anything is
  cached or forwarded.

Deployment checklist for avoiding stale content:

- Before a deploy, confirm the CDN has no cache-everything rule, no edge
  minimum TTL, and no `s-maxage`/stale-serving override that could hold the
  nonce-bearing HTML or API responses.
- A warm CDN alone does not make a new release safe. The Go binary embeds
  exactly one `dist` release, so the new process can serve only the new
  content-addressed asset set. Before shifting traffic, either retain and
  serve the complete old AND new `assets/*-<hash>.*` sets from a shared
  versioned static origin (a bucket or CDN path that both releases keep
  referencing), or perform an atomic/drained frontend cutover with verified
  availability of the new assets. Old HTML shells may still reference old
  hashes, so both sets must stay fetchable until no old shell remains in the
  wild. Never negative-cache unknown `/assets/` paths (see above): an edge
  rule that caches a `404` for a not-yet-served hash turns a harmless old
  reference into a permanently broken asset.
- After a deploy, purge only mutable paths whose bytes actually changed
  (`/index.html`-adjacent mutable files, or `data/public` overrides you
  replaced). Leave `assets/*-<hash>.*` untouched; old shells still reference
  the old hashes.
- Distinguish origin-side convergence from edge caching: admin setting changes
  that alter generated HTML are propagated across replicas by the origin's
  invalidation bus (CSP refresh, HTML-cache invalidation). That bus does not
  reach the CDN, so a CDN-stored mutable copy only updates once it revalidates
  or is purged. If an admin-visible change does not appear, check the CDN's
  stored copy before assuming the origin failed to converge.

## DDoS boundary

Application checks reduce amplification after a connection reaches Go. They
cannot absorb volumetric attacks, TLS floods, bandwidth saturation, or a large
distributed source set. Those require upstream network capacity, CDN/WAF
filtering, provider firewall rules, and origin isolation. Avoid high-cardinality
metrics or per-request database security logs during rejection storms.
