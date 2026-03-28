# Registry API Specification

**Стабильные ссылки на эту спецификацию**

- Файл в репозитории: [`docs/registry-api.md`](https://github.com/getskillpack/registry/blob/main/docs/registry-api.md)
- Развёрнутый реестр (Markdown, `GET`): `https://registry.skpkg.org/docs/registry-api` (или ваш `REGISTRY_PUBLIC_URL` + `/docs/registry-api`)

Корневой [`API.md`](../API.md) оставлен как короткий указатель для старых ссылок.

---

Base URL: `https://registry.skpkg.org/api/v1`

## Authentication

All endpoints requiring write access (publish, yank) require a Bearer token in the `Authorization` header.

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
