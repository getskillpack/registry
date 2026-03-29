# Reference-инстанс registry: краткий runbook эксплуатации

Одностраничный чеклист для **публичного (reference/production)** инстанса [getskillpack/registry](https://github.com/getskillpack/registry). Детали SLO, бэкапов и go-live — в [PUBLIC_REGISTRY_RUNBOOK.md](./PUBLIC_REGISTRY_RUNBOOK.md). Контракт HTTP — [registry-api.md](./registry-api.md) и [API.md](../API.md).

**Плейсхолдеры `TBD_*` ниже должен заполнить board / владелец инфраструктуры** (конкретный хостинг, секреты и URL пайплайнов в этом репозитории не фиксируются).

---

## 1. Поднять и обновить

| Шаг | Действие |
|-----|----------|
| Артефакт | Собрать бинарь из `main`/релизного тега `v*` или использовать образ, согласованный с FE2/board (**TBD_IMAGE_OR_BINARY_SOURCE**). |
| Данные | Смонтировать персистентный том на **`REGISTRY_DATA_DIR`** (см. раздел про бэкап/восстановление в [PUBLIC_REGISTRY_RUNBOOK.md](./PUBLIC_REGISTRY_RUNBOOK.md)). |
| Конфиг | Задать обязательные переменные из §2; прокси с TLS и заголовками `X-Forwarded-Proto` / `X-Forwarded-Host` — см. раздел про окружения prod/stage в [PUBLIC_REGISTRY_RUNBOOK.md](./PUBLIC_REGISTRY_RUNBOOK.md). |
| Проверка после выката | `GET /healthz`, `GET /readyz`, выборочно `GET /api/v1/skills` и поле `archive_url` в ответе. |

**CI в этом репозитории:** [`.github/workflows/go.yml`](../.github/workflows/go.yml) — тесты и e2e-smoke; **автоматического деплоя продакшена из GitHub Actions здесь нет**. Канонический способ выката для вашего окружения: **TBD_DEPLOY_RUNBOOK_URL** (внутренняя ссылка или runbook платформы).

**Обновление (типовой сценарий):** выкатить новую версию артефакта с тем же `REGISTRY_DATA_DIR`, перезапустить процесс/под; затем § «Проверка после выката».

---

## 2. Обязательные переменные окружения (prod)

| Переменная | Обязательно | Заметка |
|------------|-------------|---------|
| `REGISTRY_PUBLIC_URL` | **Да** | Канонический HTTPS base URL **без** завершающего `/` (иначе в JSON могут попасть внутренние хосты). |
| `REGISTRY_DATA_DIR` | Практически да | Путь к персистентному каталогу данных. |

Остальные переменные (таймауты, rate limit, метрики, read/write токены) — по политике окружения; полная таблица — в [PUBLIC_REGISTRY_RUNBOOK.md](./PUBLIC_REGISTRY_RUNBOOK.md) (раздел про переменные окружения) и корневом [README.md](../README.md).

**Публичный read-only сервис:** не задавайте **`REGISTRY_WRITE_TOKEN`** на боевом чтении; публикацию выполняйте с отдельного хоста/джобы. Секреты — только в менеджере секретов (**TBD_SECRET_STORE**), не в git.

---

## 3. Health и readiness

| Probe | Запрос | Ожидание |
|-------|--------|----------|
| Liveness | `GET /healthz` | `200`, тело `ok` |
| Readiness | `GET /readyz` | `200` если доступны `skills/` и `archives/` под `REGISTRY_DATA_DIR`; иначе `503` |

Пример снаружи (подставьте публичный хост):

```bash
curl -fsS "https://registry.example.com/healthz"
curl -fsS "https://registry.example.com/readyz"
```

Версия билда: `GET /version` (текст).

---

## 4. Где смотреть логи

Приложение пишет запросы в **stderr** через `slog` (поля: метод, путь, статус, длительность, IP и т.д.). Для агрегации в платформе включите **`REGISTRY_LOG_FORMAT=json`**.

| Среда | Где смотреть |
|-------|----------------|
| Kubernetes | `kubectl logs` для пода сервиса (**TBD_NAMESPACE_DEPLOYMENT**) |
| systemd | `journalctl -u TBD_UNIT_NAME -f` |
| Docker | `docker logs TBD_CONTAINER` |

Прокси (nginx, ingress, LB) — отдельные access/error логи для корреляции `5xx` и TLS.

---

## 5. Rollback

1. **Код/образ:** вернуть предыдущий известный рабочий **тег/дижест** образа или бинарь (**TBD_PREVIOUS_ARTIFACT_POLICY**). Перезапуск сервиса без изменения данных обычно достаточен при регрессии в релизе.
2. **Данные:** при порче хранилища — восстановить `REGISTRY_DATA_DIR` из бэкапа (процедура в [PUBLIC_REGISTRY_RUNBOOK.md](./PUBLIC_REGISTRY_RUNBOOK.md)), затем перезапуск и `readyz`.
3. **Пайплайн:** если в CI/CD есть откат на предыдущий релиз — **TBD_ROLLBACK_PROCEDURE_URL**; иначе ручной шаг из пункта 1.

Таблица инцидентов (симптом → действие) — в [PUBLIC_REGISTRY_RUNBOOK.md](./PUBLIC_REGISTRY_RUNBOOK.md) (раздел про инциденты).

---

## 6. Эскалация (секреты и инфраструктура)

Факты о прод-хостинге, именах секретов, дежурстве и канонических URL деплоя **не описаны в коде** — их задаёт board / DevOps.

**Попросите board** завести внутренний тикет с назначением на board и чеклистом минимум:

- [ ] **TBD_DEPLOY_RUNBOOK_URL** — как именно выкатывается prod/stage.
- [ ] **TBD_IMAGE_OR_BINARY_SOURCE** — откуда брать артефакт (registry образов, артефакты сборки).
- [ ] **TBD_SECRET_STORE** и имена ключей для `REGISTRY_*_TOKEN` (если используются).
- [ ] **TBD_ONCALL** / канал эскалации при недоступности `registry.skpkg.org` (или актуального публичного URL).

В репозитории остаются только плейсхолдеры; после заполнения их можно перенести во внутреннюю wiki, а здесь оставить ссылку.
