# Registry API Specification

Base URL: `https://registry.skpkg.org/api/v1`

## Health and readiness

- `GET /healthz` — процесс жив (`200`, тело `ok`).
- `GET /readyz` — хранилище готово: подкаталоги `skills/` и `archives/` в `REGISTRY_DATA_DIR` существуют и являются директориями (`200` или `503`).

## Metrics (optional)

When the operator sets **`REGISTRY_ENABLE_METRICS`** (`1` / `true` / `yes`), the server exposes **`GET /metrics`** in Prometheus text format. Typical series: `registry_http_requests_total` and `registry_http_request_duration_seconds` with labels `method`, `route`, `code`. The `/metrics` path is not subject to optional read-token auth (protect it at the network or ingress layer if needed).

## Authentication

All endpoints requiring write access (publish, yank) require a Bearer token in the `Authorization` header. The value must match the registry operator’s `REGISTRY_WRITE_TOKEN`.

If **`REGISTRY_READ_TOKEN`** is set, **read** access to **`GET`/`HEAD`** on `/api/v1/*` and `/downloads/*` requires `Authorization: Bearer <read token>`. `POST` and `DELETE` continue to use only the write token for mutating routes.

**CLI clients** (`skillget publish`): set **`SKILLGET_REGISTRY_TOKEN`** (canonical) or **`SKILLGET_TOKEN`** (short alias) in the environment so the client sends `Authorization: Bearer <token>`.

## Endpoints

### 1. `GET /skills`
Search and list skills.

**Query Parameters:**
- `q` (string, optional) - Search query.
- `author` (string, optional) - Filter by author.
- `limit` (number, optional, default: 20)
- `offset` (number, optional, default: 0)

**Response:**
```json
{
  "data": [
    {
      "name": "para-memory-files",
      "description": "File-based memory system using PARA",
      "author": "company",
      "latest_version": "1.0.4",
      "created_at": "2026-03-27T00:00:00Z"
    }
  ],
  "meta": { "total": 1, "limit": 20, "offset": 0 }
}
```

### 2. `GET /skills/:name`
Get detailed information about a specific skill, including all published versions.

### 3. `GET /skills/:name/versions/:version`
Get metadata and download URL for a specific version.

**Response:**
```json
{
  "name": "para-memory-files",
  "version": "1.0.4",
  "manifest": {
    "dependencies": {}
  },
  "archive_url": "https://registry.skpkg.org/downloads/para-memory-files-1.0.4.tar.gz",
  "checksum": "sha256:abc123def456..."
}
```

### 4. `POST /skills`
Publish a new skill or a new version of an existing skill.
Requires `multipart/form-data` with:

- **`manifest`** (string field): JSON object with at least **`name`** and **`version`**; optional **`description`**, **`author`**, and any extra keys stored verbatim.
- **`archive`** (file field): gzip tarball (`.tar.gz`) of the skill payload.

On success the server responds with **`201 Created`** and an empty body.

### 5. `DELETE /skills/:name/versions/:version` (Yank)
Soft-delete (yank) a specific version, making it unavailable for new installations.
