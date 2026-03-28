# registry

Центральный реестр и маркетплейс скиллов для экосистемы [getskillpack](https://github.com/getskillpack).

## Содержимое

- **[API.md](./API.md)** — контракт HTTP API (`/api/v1`).
- **[docs/PUBLIC_REGISTRY_RUNBOOK.md](./docs/PUBLIC_REGISTRY_RUNBOOK.md)** — публичный инстанс: SLO/SLI, секреты, бэкапы, алёрты, чеклист go-live и границы prod/stage для согласования с FE2.
- **Сервер на Go** (`cmd/registry`) — файловое хранилище под `./data`, маршруты из `API.md` + выдача архивов по `GET /downloads/{sha256}.tar.gz`.

### Сборка и запуск

Требуется **Go 1.22+**.

```bash
go build -o registry ./cmd/registry
REGISTRY_WRITE_TOKEN='секрет' ./registry
```

Переменные окружения:

| Переменная | Назначение |
|------------|------------|
| `REGISTRY_LISTEN_ADDR` | Адрес прослушивания (по умолчанию `:8080`) |
| `REGISTRY_DATA_DIR` | Каталог данных (по умолчанию `data`) |
| `REGISTRY_WRITE_TOKEN` | Bearer-токен для `POST /api/v1/skills` и `DELETE ...` (если пусто — только чтение) |
| `REGISTRY_READ_TOKEN` | Если задан — для `GET`/`HEAD` на `/api/v1/*` и `/downloads/*` нужен `Authorization: Bearer` с этим значением; `POST`/`DELETE` не затрагивается |
| `REGISTRY_PUBLIC_URL` | Публичный базовый URL для ссылок на архивы (если не задан — берётся из запроса) |
| `REGISTRY_LOG_FORMAT` | `json` — логи в JSON на stderr; иначе текстовый формат `slog` |
| `REGISTRY_ENABLE_METRICS` | `1` / `true` — включить `GET /metrics` (Prometheus) и счётчики `registry_http_*` |
| `REGISTRY_RATE_LIMIT_RPS` | Если задано число > 0 — лимит запросов на IP (см. выше) |
| `REGISTRY_RATE_LIMIT_BURST` | Размер burst для лимитера (опционально) |
| `REGISTRY_TRUST_FORWARDED_FOR` | `1` / `true` — брать IP из первого `X-Forwarded-For` (только за доверенным прокси) |

Проверка: `GET /healthz` → `200 ok`. Готовность: `GET /readyz` — проверка `skills/` и `archives/` под `REGISTRY_DATA_DIR` (`200` или `503`).

**Наблюдаемость:** ответы (включая `401`/`429`) пишутся в stderr через `slog` (`method`, `path`, `status`, `duration_ms`, `remote_ip`). При `REGISTRY_ENABLE_METRICS` — ещё `GET /metrics` (без `REGISTRY_READ_TOKEN`) и счётчики Prometheus на тех же ответах.

**Rate limit (опционально):** задайте `REGISTRY_RATE_LIMIT_RPS` (например `50`) — per-IP token bucket для всех путей, кроме `/healthz`, `/readyz` и `/metrics`. Дополнительно: `REGISTRY_RATE_LIMIT_BURST` (если не задан — `max(10, 2×RPS)`). За прокси с корректным `X-Forwarded-For` установите `REGISTRY_TRUST_FORWARDED_FOR=1` (доверяйте только своему ingress).

### Тесты

```bash
go test ./... -count=1
```

### E2E: skillget ↔ registry

См. [docs/E2E_SMOKE.md](./docs/E2E_SMOKE.md). Кратко: `make e2e-smoke` (нужен Go, `curl`, `python3` и исходники CLI в `SKILLGET_SRC` или бинарь `skillget` в `PATH`). В CI — job `e2e-smoke` в `.github/workflows/go.yml`.

## Связанные репозитории

| Репозиторий | Роль |
|-------------|------|
| [skillget-manager](https://github.com/getskillpack/skillget-manager) | Ядро менеджера пакетов (lockfile, клиент реестра, установка) |
| [cli](https://github.com/getskillpack/cli) | Тонкая обёртка CLI (`skillget`) над менеджером |

## Лицензия

MIT
