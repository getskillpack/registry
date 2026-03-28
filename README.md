# registry

Центральный реестр и маркетплейс скиллов для экосистемы [getskillpack](https://github.com/getskillpack).

## Содержимое

- **[docs/registry-api.md](./docs/registry-api.md)** — канонический контракт HTTP API (`/api/v1`); на развёрнутом реестре тот же текст: `GET /docs/registry-api`.
- **[API.md](./API.md)** — короткий указатель на файл выше (совместимость со старыми ссылками).
- **[docs/PUBLISH.md](./docs/PUBLISH.md)** — как собрать архив и опубликовать пакет (curl, секреты, CI).
- **Пример skill-пакета** — [`examples/hello-skill/`](./examples/hello-skill/) и готовый архив [`examples/dist/hello-skill-0.1.0.tar.gz`](./examples/dist/hello-skill-0.1.0.tar.gz) (пересборка: `scripts/pack-example.sh`).
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
| `REGISTRY_PUBLIC_URL` | Публичный базовый URL для ссылок на архивы (если не задан — берётся из запроса) |

Проверка: `GET /healthz` → `200 ok`.

### Тесты

```bash
go test ./... -count=1
```

## Связанные репозитории

| Репозиторий | Роль |
|-------------|------|
| [skillget-manager](https://github.com/getskillpack/skillget-manager) | Ядро менеджера пакетов (lockfile, клиент реестра, установка) |
| [cli](https://github.com/getskillpack/cli) | Тонкая обёртка CLI (`skillget`) над менеджером |

## Лицензия

MIT
