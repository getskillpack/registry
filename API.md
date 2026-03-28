# Registry API Specification

The canonical specification lives in **[docs/registry-api.md](./docs/registry-api.md)** — a stable path for links from README, marketing, and the site.

| Where | URL |
|-------|-----|
| GitHub (`main`) | https://github.com/getskillpack/registry/blob/main/docs/registry-api.md |
| Deployed registry | `GET /docs/registry-api` on your instance (e.g. `https://registry.skpkg.org/docs/registry-api`) |

## Health and readiness

- `GET /healthz` — liveness (`200`, body `ok`).
- `GET /readyz` — storage readiness: data root and `skills/` + `archives/` exist and are directories (`200` or `503`, body `not ready` on failure).
- `GET /version` — build version as plain text (same as `registry -version`; see root README for `-ldflags`).

## Metrics (optional)

When **`REGISTRY_ENABLE_METRICS`** is `1` / `true` / `yes`, the server exposes **`GET /metrics`** in Prometheus text format. Typical series: `registry_http_requests_total` and `registry_http_request_duration_seconds` with labels `method`, `route`, `code`. The `/metrics` path is not subject to optional read-token auth (protect at the network or ingress if needed).

## Authentication

Write routes (`POST /skills`, yank) require `Authorization: Bearer` matching **`REGISTRY_WRITE_TOKEN`**.

If **`REGISTRY_READ_TOKEN`** is set, **read** access to **`GET`/`HEAD`** on `/api/v1/*` and `/downloads/*` requires `Authorization: Bearer <read token>`. `POST` and `DELETE` continue to use the write token on mutating routes.

**CLI clients** (`skillget publish`): set **`SKILLGET_REGISTRY_TOKEN`** (canonical) or **`SKILLGET_TOKEN`** (short alias) so the client sends `Authorization: Bearer <token>`.

Operational summary: root [README](./README.md) lists environment variables and rate limiting.
