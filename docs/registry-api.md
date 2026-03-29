# Registry API Specification

**Стабильные ссылки на эту спецификацию**

- Файл в репозитории: [`docs/registry-api.md`](https://github.com/getskillpack/registry/blob/main/docs/registry-api.md)
- Развёрнутый реестр (Markdown, `GET`): `https://registry.skpkg.org/docs/registry-api` (или ваш `REGISTRY_PUBLIC_URL` + `/docs/registry-api`)

Корневой [`API.md`](../API.md) оставлен как короткий указатель для старых ссылок.

---

## Compiled core: источник правды (HTTP)

Для **скомпилированных клиентов** (CLI **skillget**, smoke **core-spike-go**, любые SDK поверх HTTP) канон контракта — **этот документ** плюс типы JSON в исходниках реестра [`internal/store/store.go`](https://github.com/getskillpack/registry/blob/main/internal/store/store.go) (`SkillSummary`, `SkillDetail`, `VersionPublicInfo`, `VersionRecord`). Реализация маршрутов: [`internal/api/server.go`](https://github.com/getskillpack/registry/blob/main/internal/api/server.go).

- **Базовый URL API** — `{origin}/api/v1` без завершающего слэша в переменной окружения **`SKILLGET_REGISTRY_URL`** (клиенты склеивают пути вида `/skills`, `/skills/{name}/versions/{version}`).
- **`archive_url`** в ответах — абсолютный URL вида `{publicBase}/downloads/{sha256_hex}.tar.gz`, где `sha256_hex` — 64 hex-символа (имя файла на диске реестра), **не** человекочитаемое имя скилла.
- **`checksum`** — строка вида `sha256:` + тот же hex.
- Регрессия контракта в репозитории: `go test ./...` (в т.ч. `TestRegistryFlow`, `TestPublishYankWriteContract`, `TestGETSkillDetailContract`), опционально удалённый smoke при `REGISTRY_SMOKE_BASE_URL` + `REGISTRY_SMOKE_WRITE_TOKEN`.
- **Скомпилированный клиент (Go):** поведение библиотеки `github.com/getskillpack/skillget-manager` и таблица **переменных окружения клиента** (`SKILLGET_REGISTRY_URL`, алиас `SKPKG_REGISTRY_URL`; `SKILLGET_REGISTRY_READ_TOKEN` при включённом на сервере `REGISTRY_READ_TOKEN`; запись — `SKILLGET_REGISTRY_TOKEN` / `SKILLGET_TOKEN`) — в [`REGISTRY_CLIENT_CONTRACT.md`](https://github.com/getskillpack/skillget-manager/blob/main/docs/REGISTRY_CLIENT_CONTRACT.md). Там же зафиксировано, что канон HTTP-сервера — этот файл (раздел **Compiled core**).

### Статус-коды по маршрутам (ожидания клиента)

| Метод | Путь (относительно `/api/v1`) | Успех | Типичные ошибки |
|--------|-------------------------------|-------|------------------|
| `GET` | `/skills` | `200` + JSON list | `401` если задан read-token |
| `GET` | `/skills/{name}` | `200` + JSON detail | `404`, `401` |
| `GET` | `/skills/{name}/versions/{version}` | `200` + JSON version | `404`, **`410`** после yank, `401` |
| `POST` | `/skills` | **`201`** пустое тело | `400`, `401`, `409` дубликат версии, `503` без write token |
| `DELETE` | `/skills/{name}/versions/{version}` | **`204`** | `401`, `404` |

Тела ошибок — короткий `text/plain` от reference-сервера (не JSON).

### Публикация: multipart

`POST /skills`, `Content-Type: multipart/form-data`:

| Поле | Содержимое |
|------|------------|
| `manifest` | JSON-строка; обязательны **`name`**, **`version`** (semver); используются также **`description`**, **`author`**. Дополнительные поля остаются в сохранённом manifest как JSON. |
| `archive` | Файл `.tar.gz` (байты не пустые). |

Заголовок: `Authorization: Bearer <write token>`.

### Согласование с инженерными требованиями

Детальный чеклист из `plans/ENGINEERING_REQUIREMENTS_SKPKG.md` живёт в onboarding / CTO workspace. **Расхождения с этим файлом** фиксируем в рабочем тикете компании (Paperclip) и при необходимости краткой сноской в этом разделе после review.

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

**Response (shape):**
```json
{
  "name": "para-memory-files",
  "description": "File-based memory system using PARA",
  "author": "company",
  "created_at": "2026-03-27T00:00:00Z",
  "versions": {
    "1.0.4": {
      "manifest": { "name": "para-memory-files", "version": "1.0.4" },
      "checksum": "sha256:<64 hex chars>",
      "archive_url": "https://registry.skpkg.org/downloads/<64 hex chars>.tar.gz",
      "published_at": "2026-03-27T00:00:00Z",
      "yanked": false
    }
  }
}
```

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
  "archive_url": "https://registry.skpkg.org/downloads/<64 hex chars>.tar.gz",
  "checksum": "sha256:<64 hex chars>"
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
