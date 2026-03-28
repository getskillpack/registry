# Registry API Specification

**Стабильные ссылки на эту спецификацию**

- Файл в репозитории: [`docs/registry-api.md`](https://github.com/getskillpack/registry/blob/main/docs/registry-api.md)
- Развёрнутый реестр (Markdown, `GET`): `https://registry.skpkg.org/docs/registry-api` (или ваш `REGISTRY_PUBLIC_URL` + `/docs/registry-api`)

Корневой [`API.md`](../API.md) оставлен как короткий указатель для старых ссылок.

---

Base URL: `https://registry.skpkg.org/api/v1`

## Authentication

**Write** (publish, yank): the server operator sets `REGISTRY_WRITE_TOKEN`; clients send `Authorization: Bearer <token>`. If the token is not configured on the server, mutating routes return `503`.

**Read** (optional): if the operator sets `REGISTRY_READ_TOKEN`, every **`GET`/`HEAD`** to **`/api/v1/*`** and **`/downloads/*`** must include `Authorization: Bearer <read token>` or the server responds with **`401`**. **`POST`** and **`DELETE`** on `/api/v1/*` still use only the **write** token.

The following paths are **not** protected by the read token: **`/healthz`**, **`/readyz`**, and (when enabled) **`/metrics`** — restrict them at the network or ingress layer if needed.

CLI publishing uses `SKILLGET_REGISTRY_TOKEN` / `SKILLGET_TOKEN` for the write Bearer (see repository docs).

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
Requires multipart/form-data with `manifest` (JSON) and `archive` (file).

### 5. `DELETE /skills/:name/versions/:version` (Yank)
Soft-delete (yank) a specific version, making it unavailable for new installations.

---

## Operational endpoints (outside `/api/v1`)

These paths are served by the reference server for probes and do not use the `Base URL` above:

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/healthz` | Liveness — process is up (does not verify storage). |
| `GET` | `/readyz` | Readiness — data directory layout is usable (`skills/`, `archives/`). Returns `503` if not. |
| `GET` | `/version` | Plain-text build version (`text/plain`). Same value as `registry -version` (override with `-ldflags` — see repository README). Not under read-token auth. |
| `GET` | `/metrics` | Prometheus scrape endpoint when `REGISTRY_ENABLE_METRICS` is enabled (not under read-token auth). |

Optional rate limiting, structured request logging, and other server knobs are documented in the repository [README](../README.md).
