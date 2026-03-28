# registry

Central registry and skill marketplace for the [getskillpack](https://github.com/getskillpack) ecosystem.

## Contents

- **[docs/ORG_README_POLICY.md](./docs/ORG_README_POLICY.md)** — English-only root `README.md` policy for public repos in the getskillpack org.
- **[docs/registry-api.md](./docs/registry-api.md)** — canonical HTTP API contract (`/api/v1`); on a deployed registry the same document is served at `GET /docs/registry-api`.
- **[API.md](./API.md)** — short pointer plus health, metrics, and auth notes (backward compatibility for older links).
- **[docs/PUBLISH.md](./docs/PUBLISH.md)** — how to build an archive and publish a package (curl, secrets, CI).
- **[docs/PUBLIC_REGISTRY_RUNBOOK.md](./docs/PUBLIC_REGISTRY_RUNBOOK.md)** — public instance: SLO/SLI, secrets, backups, alerts, go-live checklist (RU; FE2 alignment).
- **[docs/PUBLIC_ECOSYSTEM.md](./docs/PUBLIC_ECOSYSTEM.md)** — which getskillpack repos are public, and what replaced the removed `skillget-manager` prototype.
- **[docs/SKPKG_VS_PACKAGE_MANAGERS_RU.md](./docs/SKPKG_VS_PACKAGE_MANAGERS_RU.md)** — factual comparison of skpkg vs npm / Packagist / PyPI (Russian; for Growth/CMO messaging).
- **Example skill package** — [`examples/hello-skill/`](./examples/hello-skill/) and a prebuilt archive [`examples/dist/hello-skill-0.1.0.tar.gz`](./examples/dist/hello-skill-0.1.0.tar.gz) (rebuild: `scripts/pack-example.sh`).
- **Go server** (`cmd/registry`) — file-backed store under `./data`, routes from `docs/registry-api.md` plus archive delivery at `GET /downloads/{sha256}.tar.gz`.

### Build and run

Requires **Go 1.22+**.

```bash
go build -o registry ./cmd/registry
./registry -version
```

Release / image builds should stamp the version (also served as `GET /version`):

```bash
go build -ldflags "-X github.com/getskillpack/registry.Version=1.2.3" -o registry ./cmd/registry
```

Run with write token:

```bash
REGISTRY_WRITE_TOKEN='secret' ./registry
```

Environment variables:

| Variable | Purpose |
|----------|---------|
| `REGISTRY_LISTEN_ADDR` | Listen address (default `:8080`) |
| `REGISTRY_DATA_DIR` | Data directory (default `data`) |
| `REGISTRY_WRITE_TOKEN` | Bearer token for `POST /api/v1/skills` and `DELETE ...` (if empty — read-only) |
| `REGISTRY_READ_TOKEN` | If set, `GET`/`HEAD` on `/api/v1/*` and `/downloads/*` require `Authorization: Bearer` with this value; `POST`/`DELETE` still use the write token |
| `REGISTRY_PUBLIC_URL` | Public base URL for archive links (if unset — derived from the request) |
| `REGISTRY_LOG_FORMAT` | Set to `json` for JSON request logs on stderr (default: text `slog`) |
| `REGISTRY_ENABLE_METRICS` | `1` / `true` / `yes` — enable `GET /metrics` (Prometheus) and `registry_http_*` series |
| `REGISTRY_RATE_LIMIT_RPS` | Per-IP rate limit (token bucket). If unset or `0`, limiting is **disabled** |
| `REGISTRY_RATE_LIMIT_BURST` | Burst size for the limiter (default `1` when RPS is enabled) |
| `REGISTRY_TRUST_FORWARDED_FOR` | If `true`/`1`/`yes`/`on`, use `X-Forwarded-For` for client IP (only behind a **trusted** proxy) |
| `REGISTRY_HTTP_READ_HEADER_TIMEOUT_SEC` | HTTP server read-header timeout in seconds (default `10`; set `0` to disable) |
| `REGISTRY_HTTP_READ_TIMEOUT_SEC` | If >0, total request read timeout (seconds) |
| `REGISTRY_HTTP_WRITE_TIMEOUT_SEC` | If >0, response write timeout (seconds); use a large value if serving big downloads |
| `REGISTRY_HTTP_IDLE_TIMEOUT_SEC` | If >0, idle connection timeout (seconds) |

Health check: `GET /healthz` → `200 ok` (liveness; does not verify storage).

Readiness: `GET /readyz` → `200 ok` when the data directory and required subfolders are reachable; `503` if storage is not usable.

Build id: `GET /version` (plain text) or `registry -version` — same string as `registry.Version` (default `0.0.0-dev` until set via `-ldflags`).

**Observability:** each response is logged to stderr via `slog` (`method`, `path`, `status`, `duration`, `remote`, `bytes`). With `REGISTRY_ENABLE_METRICS`, also scrape `GET /metrics` (exempt from read-token auth; protect at the edge).

**Rate limit:** when `REGISTRY_RATE_LIMIT_RPS` is set, `/healthz`, `/readyz`, and `/metrics` are skipped. Use `REGISTRY_TRUST_FORWARDED_FOR` only behind a trusted ingress.

### Tests

```bash
go test ./... -count=1
```

### E2E: skillget ↔ registry

See [docs/E2E_SMOKE.md](./docs/E2E_SMOKE.md). Short version: `make e2e-smoke` (needs Go, `curl`, `python3`, and CLI sources in `SKILLGET_SRC` or `skillget` on `PATH`). CI: `e2e-smoke` job in `.github/workflows/go.yml`.

## Related repositories

See **[docs/PUBLIC_ECOSYSTEM.md](./docs/PUBLIC_ECOSYSTEM.md)** for the full public surface (including retired repos).

| Repository | Role |
|------------|------|
| [cli](https://github.com/getskillpack/cli) | `skillget` CLI and **Go** package-manager core (lockfile, registry client, installation) |

## License

MIT
