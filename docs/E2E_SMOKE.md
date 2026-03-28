# E2E smoke: skillget ↔ registry

Сквозной сценарий: опубликовать скилл в реестре и скачать его **skillget**, как это делает внешний пользователь.

Связанные репозитории: [getskillpack/registry](https://github.com/getskillpack/registry), [getskillpack/cli](https://github.com/getskillpack/cli) (бинарь `skillget`), [getskillpack/skillget-manager](https://github.com/getskillpack/skillget-manager).

## Сборка skillget

Нужен **Go 1.22+**. Из репозитория CLI:

```bash
git clone https://github.com/getskillpack/cli.git
cd cli
go build -o skillget ./cmd/skillget
```

Либо положить бинарь в `PATH` и вызывать `skillget`.

## Переменные окружения

| Переменная | Назначение |
|------------|------------|
| `SKILLGET_REGISTRY_URL` | Базовый URL API реестра, **с суффиксом `/api/v1`** |
| `SKPKG_REGISTRY_URL` | Устаревший fallback (тот же смысл) |

Пример для локального сервера на `8080`:

```bash
export SKILLGET_REGISTRY_URL=http://127.0.0.1:8080/api/v1
```

Публичный инстанс из спецификации: `https://registry.skpkg.org/api/v1` (см. [API.md](../API.md)).

## Ожидаемые команды и вывод

```bash
skillget config
```

Ожидаемо: строки `registry URL: …` и `source: …` (при переопределении — не `default`).

```bash
skillget search e2e-smoke
```

Ожидаемо: в списке есть строка с именем скилла (и при наличии — `@версия`) либо `No skills found.` если скилла ещё нет.

```bash
skillget install e2e-smoke-skill@1.0.0
```

Ожидаемо (успех):

- `Wrote …/e2e-smoke-skill-1.0.0.tar.gz` (путь к архиву под `.skillget/skills/…` или `-o`)
- `Updated …/skills.lock`
- при ответе реестра с checksum — строка `Checksum (registry): sha256:…`
- финальная подсказка про `tar -xzf`

**Важно:** для текущего JSON `GET /skills/:name` в реестре используется объект `versions`; клиент при **установке без `@version`** опирается на другой формат. В смоке и в документации используйте **явную версию**: `name@semver`.

## Автоматизация в этом репозитории

Локально (нужны `go`, `curl`, `python3`, собранный или доступный `skillget`):

```bash
# вариант A: указать корень исходников CLI
export SKILLGET_SRC=/path/to/cli
make e2e-smoke

# вариант B: бинарь уже в PATH
make e2e-smoke
```

Скрипт поднимает реестр на свободном порту, публикует тестовый скилл `e2e-smoke-skill@1.0.0`, затем вызывает `skillget search` и `skillget install …@1.0.0`.

В CI см. job `e2e-smoke` в [.github/workflows/go.yml](../.github/workflows/go.yml): второй checkout клонирует **приватный** `getskillpack/cli` с токеном из Actions secret **`GETSKILLPACK_ORG_PAT`** (его нужно завести в настройках репозитория/организации для `registry`, с правом чтения **и `cli`, и `skillget-manager`** — иначе `go build ./cmd/skillget` не скачает приватный модуль). Дополнительно в job выставляются `GOPRIVATE` / `GONOSUMDB` и `git config url.insteadOf` с тем же PAT для `go`/`git` при загрузке модулей. Job не запускается для PR **из форка** (секреты недоступны). После настройки секрета workflow **Go** можно перезапустить вручную (**Actions** → **Go** → **Run workflow**), не делая пустой коммит.

## Публичный happy path (вручную)

Если в продакшене опубликован скилл (пример из [API.md](../API.md): `para-memory-files`):

```bash
export SKILLGET_REGISTRY_URL=https://registry.skpkg.org/api/v1
skillget search para
skillget install para-memory-files@1.0.4
```

Фактические имена и версии зависят от каталога в проде; при 404 уточните актуальные данные через `skillget search` или `GET /api/v1/skills`.
