# Участие в разработке `getskillpack/registry`

Спасибо за интерес к центральному реестру экосистемы [getskillpack](https://github.com/getskillpack). Ниже — минимум, чтобы собрать сервер, вносить правки в контракт и документацию и открывать осмысленный PR; длинные процедуры эксплуатации — только по ссылкам.

## Что почитать в первую очередь

- **Корневой обзор, сборка, переменные окружения:** [README.md](README.md)
- **Канонический HTTP API (`/api/v1`):** [docs/registry-api.md](docs/registry-api.md) — же содержимое отдаётся с живого инстанса как `GET /docs/registry-api`
- **Краткий указатель на health/metrics и совместимость со старыми ссылками:** [API.md](API.md)
- **Публикация пакета (curl, токены, CI):** [docs/PUBLISH.md](docs/PUBLISH.md)
- **Матрица совместимости** (CLI, skillget-manager, registry): [COMPATIBILITY_MATRIX_RU.md](https://github.com/getskillpack/cli/blob/main/docs/COMPATIBILITY_MATRIX_RU.md)
- **Клиентский контракт менеджера к реестру:** [REGISTRY_CLIENT_CONTRACT.md](https://github.com/getskillpack/skillget-manager/blob/main/docs/REGISTRY_CLIENT_CONTRACT.md)
- **Трассировка требований и тикетов:** [ENGINEERING_REQUIREMENTS_TRACEABILITY_RU.md](https://github.com/getskillpack/cli/blob/main/docs/ENGINEERING_REQUIREMENTS_TRACEABILITY_RU.md)

## Эксплуатация и runbook (не дублировать в PR)

Изменения в **операционных** процедурах публичного инстанса оформляйте в репозитории (или согласуйте с владельцем окружения):

- **SLO, секреты, бэкапы, алерты, go-live:** [docs/PUBLIC_REGISTRY_RUNBOOK.md](docs/PUBLIC_REGISTRY_RUNBOOK.md)
- **Краткий runbook: деплой, health/readiness, логи, откат, эскалация:** [docs/REFERENCE_REGISTRY_OPERATIONS_RU.md](docs/REFERENCE_REGISTRY_OPERATIONS_RU.md)

## Требования к окружению

- **Go 1.22+** (см. `go.mod` / `toolchain` в корне).

## Сборка и тесты

Из корня репозитория:

```bash
go build -o registry ./cmd/registry
./registry -version
go test ./... -count=1
```

Поведение CI — в [.github/workflows/go.yml](.github/workflows/go.yml) (в т.ч. smoke **skillget ↔ registry** на PR из этого репозитория).

### Изменения контракта API и «схемы»

1. Обновите **[docs/registry-api.md](docs/registry-api.md)** — он **встроен** в бинарь (`embed_docs.go`) и отдаётся как документация с сервера.
2. При необходимости синхронизируйте [API.md](API.md), [README.md](README.md) и связанные указатели в `docs/`.
3. Если меняется поведение, видимое клиентам CLI/менеджера, проверьте [REGISTRY_CLIENT_CONTRACT.md](https://github.com/getskillpack/skillget-manager/blob/main/docs/REGISTRY_CLIENT_CONTRACT.md) и при необходимости **отдельный PR** в [`getskillpack/skillget-manager`](https://github.com/getskillpack/skillget-manager); матрицу совместимости — в [`getskillpack/cli`](https://github.com/getskillpack/cli).

## Issues и pull requests

1. **Issue** — версия/образ registry при необходимости, шаги воспроизведения, ожидаемое и фактическое поведение, фрагмент запроса/ответа или лога.
2. **PR** — ветка **`main`**, одна логическая тема на PR; пользовательски заметные изменения поведения API сопровождайте обновлением `docs/registry-api.md` (и связанных доков), без копирования runbook-прозы из ops-документов.
3. Агентам Paperclip и автоматизации в org **getskillpack**: гигиена PAT, ветки и push — [AGENT_GITHUB_REPO_WORKFLOW_RU.md](https://github.com/getskillpack/cli/blob/main/docs/AGENT_GITHUB_REPO_WORKFLOW_RU.md).

Смежные репозитории: пользовательский CLI — [`getskillpack/cli`](https://github.com/getskillpack/cli); библиотека ядра установки — [`getskillpack/skillget-manager`](https://github.com/getskillpack/skillget-manager).

## Безопасность

Сообщения об уязвимостях — по [SECURITY.md](SECURITY.md).
