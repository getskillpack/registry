# Публичный registry: runbook, SLO и go-live

Операционная документация для **публичного** инстанса [getskillpack/registry](https://github.com/getskillpack/registry). Связанный эпик: планирование инфраструктуры и запуска на стороне команды.

Контракт API и пример базового URL: [API.md](../API.md) (`https://registry.skpkg.org/api/v1`).

---

## 1. Архитектура (кратко)

- Один бинарь `registry` (`cmd/registry`), Go 1.22+.
- **Файловое хранилище** в каталоге `REGISTRY_DATA_DIR` (по умолчанию `./data`): метаданные и архивы скиллов.
- HTTP: API под префиксом `/api/v1`, выдача архивов `GET /downloads/{file}`, liveness `GET /healthz` → `200` и тело `ok\n`, readiness `GET /readyz` (проверка `skills/` и `archives/`), `GET /version` — текстовая версия билда. Опционально `GET /metrics` при `REGISTRY_ENABLE_METRICS`.
- TLS и публичный DNS обычно на **обратном прокси** (nginx, Caddy, cloud LB); приложение слушает внутренний `REGISTRY_LISTEN_ADDR`.

---

## 2. Окружения: prod и stage (согласование с FE2)

Зафиксировать в инфраструктуре и у FE2 **до go-live**:

| Решение | Stage (черновик) | Production |
|--------|-------------------|------------|
| Публичный базовый URL API | *TBD, например `https://stage.registry.skpkg.org`* | `https://registry.skpkg.org` (как в [API.md](../API.md)) |
| Значение `REGISTRY_PUBLIC_URL` | Канонический HTTPS URL **без** завершающего `/` | То же для prod |
| Источник TLS-сертификатов | По политике FE2 / платформы | То же |
| Кто владеет DNS-записями | FE2 / DevOps | То же |

**Почему важен `REGISTRY_PUBLIC_URL`:** в ответах API подставляются ссылки на архивы (`archive_url`). Если переменная не задана, база берётся из заголовков запроса (`X-Forwarded-Proto`, `X-Forwarded-Host`, `Host`). Для продакшена задайте явный канонический URL, чтобы не утекали внутренние имена хостов.

**Прокси:** пробрасывайте `X-Forwarded-Proto` и `X-Forwarded-Host` (или терминируйте TLS на прокси с корректным `Host`), иначе ссылки в JSON могут быть некорректны при отсутствии `REGISTRY_PUBLIC_URL`.

---

## 3. Секреты

| Секрет | Назначение | Где хранить |
|--------|------------|-------------|
| `REGISTRY_WRITE_TOKEN` | Bearer для `POST /api/v1/skills` и `DELETE` (публикация / yank) | Менеджер секретов (Vault, SSM, K8s Secret); **не** в git |
| `REGISTRY_READ_TOKEN` | Если задан — для `GET`/`HEAD` на `/api/v1/*` и `/downloads/*` нужен `Authorization: Bearer` с этим значением; `POST`/`DELETE` не затрагивается | Тот же класс хранилищ, что и write-токен; для **публичного** каталога обычно **не** задаётся |
| Токены CI для публикации | Клиенты используют `SKILLGET_REGISTRY_TOKEN` / `SKILLGET_TOKEN` | CI secrets, выдаются ограниченному кругу |

**Публичный read-path:** по умолчанию чтение без токена. Если включён **`REGISTRY_READ_TOKEN`**, все клиенты (включая skillget при установке) должны передавать read-токен; иначе получат `401`. Эндпоинт **`/metrics`** read-токеном не защищается — ограничивайте доступ на сети / ingress.

Инстанс **только для чтения** снаружи: не задавайте `REGISTRY_WRITE_TOKEN` в окружении публичного сервиса; публикацию выполняйте с отдельного защищённого хоста/джобы.

---

## 4. Переменные окружения (эксплуатация)

| Переменная | Обязательно | Описание |
|------------|-------------|----------|
| `REGISTRY_LISTEN_ADDR` | Нет | `:8080` по умолчанию; в k8s часто `:8080` |
| `REGISTRY_DATA_DIR` | Нет | Путь к персистентному тому |
| `REGISTRY_PUBLIC_URL` | **Да для prod/stage** | Канонический публичный базовый URL (см. §2) |
| `REGISTRY_WRITE_TOKEN` | По политике | Только если на этом же процессе нужны write-операции |
| `REGISTRY_READ_TOKEN` | Нет | Опционально закрыть чтение API и скачивания (см. §3) |
| `REGISTRY_ENABLE_METRICS` | Нет | `1`/`true`/`yes` — `GET /metrics` (Prometheus) |
| `REGISTRY_LOG_FORMAT` | Нет | `json` — JSON-логи запросов на stderr |
| `REGISTRY_RATE_LIMIT_RPS` / `REGISTRY_RATE_LIMIT_BURST` | Нет | Per-IP лимит; см. корневой [README.md](../README.md) |
| `REGISTRY_TRUST_FORWARDED_FOR` | Нет | Только за **доверенным** прокси |
| `REGISTRY_HTTP_READ_HEADER_TIMEOUT_SEC` | Нет | По умолчанию `10` (секунды); `0` — отключить защиту от slowloris |
| `REGISTRY_HTTP_READ_TIMEOUT_SEC` | Нет | Если >0 — лимит чтения тела запроса (секунды) |
| `REGISTRY_HTTP_WRITE_TIMEOUT_SEC` | Нет | Если >0 — лимит записи ответа; для больших `GET /downloads/*` задайте достаточное значение (например `600`) |
| `REGISTRY_HTTP_IDLE_TIMEOUT_SEC` | Нет | Если >0 — `IdleTimeout` сервера |

---

## 5. SLO / SLI (предложение v1)

**SLI**

- **Доступность API чтения:** доля успешных (`2xx`) запросов к `GET /api/v1/skills`, `GET /healthz` и (при использовании) `GET /readyz` за окно.
- **Доступность скачиваний:** доля успешных `GET /downloads/{file}` (ожидается много кэша на CDN/прокси — измерять и на edge, и на origin).
- **Латентность p95:** время ответа для `GET /api/v1/skills` (без учёта огромных offset) и для `GET /downloads/*` на origin.

**SLO (черновик для согласования)**

- Доступность read-path: **99.5%** месячного окна (или выше по договорённости с FE2).
- `healthz` — liveness; для Kubernetes при желании используйте `readyz` как readiness probe (проверка хранилища).

Пороги алёртов закладывать **жёстче** SLO (например, 99.0% за 30 минут) — см. §7.

---

## 6. Резервное копирование и восстановление

- **Бэкап:** каталог `REGISTRY_DATA_DIR` целиком (согласованные снапшоты диска или `rsync`/объектное хранилище по расписанию).
- **RPO/RTO:** зафиксировать с FE2 (например, RPO ≤ 24 ч для v1, RTO ≤ 4 ч).
- **Восстановление:** остановить процесс, восстановить данные в `REGISTRY_DATA_DIR`, запустить процесс, проверить `GET /healthz` и выборочно `GET /api/v1/skills`.

---

## 7. Наблюдаемость и алёрты

- Встроенные логи: **`slog`** на stderr (`REGISTRY_LOG_FORMAT=json` для JSON): метод, путь, статус, длительность, клиентский IP (и при необходимости объём ответа).
- При **`REGISTRY_ENABLE_METRICS`**: scrape **`GET /metrics`** (серии `registry_http_requests_total`, `registry_http_request_duration_seconds` и др.) — защищайте endpoint на сети.
- Инфраструктура: размер диска под `REGISTRY_DATA_DIR`, ошибки `5xx` на прокси, RPS/latency с edge.

Алёрты (минимум): рост доли `5xx`, недоступность `healthz`, при использовании `readyz` — провал readiness, заполнение диска > порога, истечение TLS-сертификата.

---

## 8. Развёртывание (типовой сценарий)

1. Собрать бинарь: `go build -o registry ./cmd/registry` (или образ на базе `scratch`/distroless — по политике FE2).
2. Примонтировать персистентный том на `REGISTRY_DATA_DIR`.
3. Выставить `REGISTRY_PUBLIC_URL` для данного окружения.
4. Запустить за reverse proxy с TLS; проверить с внешней сети `GET /healthz`, `GET /readyz`, `GET /api/v1/skills` (и при включённом read-токене — с заголовком `Authorization`).
5. Убедиться, что в JSON ответов `archive_url` указывает на ожидаемый публичный хост.
6. Если задан `REGISTRY_HTTP_WRITE_TIMEOUT_SEC`, убедиться, что большие скачивания `GET /downloads/*` укладываются в лимит.

---

## 9. Чеклист go-live

- [ ] DNS для prod (и stage, если нужен) указывает на балансировщик / FE2 edge.
- [ ] TLS валиден, автообновление включено или алерт на expiry.
- [ ] `REGISTRY_PUBLIC_URL` совпадает с публичным каноном; smoke-тест `archive_url` из API.
- [ ] Персистентный том и расписание бэкапов; пробное восстановление на стенде.
- [ ] Алёрты по SLO/инфра подключены; дежурство/маршрут эскалации известен.
- [ ] Публичный сервис без `REGISTRY_WRITE_TOKEN` **или** write отключён сетевой политикой (предпочтительно отдельный publish-пайплайн).
- [ ] Решение по `REGISTRY_READ_TOKEN` (публичный анонимный read vs закрытый каталог) задокументировано; клиенты/CI обновлены при включении.
- [ ] При `REGISTRY_ENABLE_METRICS` — scrape настроен, `/metrics` не торчит в публичный интернет без фильтра.
- [ ] HTTP-таймауты (`REGISTRY_HTTP_*`) согласованы с размером архивов и прокси.
- [ ] Документирован контакт FE2 и владелец DNS/TLS для registry.
- [ ] Согласованы имена хостов stage/prod с FE2 (таблица в §2 заполнена).

---

## 10. Инциденты (краткий runbook)

| Симптом | Действия |
|---------|----------|
| `healthz` не отвечает | Проверить процесс/под, прокси, диск, недавние деплои; откат к предыдущему образу при подозрении на регрессию. |
| `readyz` = 503 | Права и монтирование `REGISTRY_DATA_DIR`, наличие подкаталогов `skills/`, `archives/`. |
| Массовые `401` на чтении | Проверить `REGISTRY_READ_TOKEN` и заголовок `Authorization` у клиентов. |
| Массовые `5xx` | Логи приложения и прокси; проверить место на диске и права на `REGISTRY_DATA_DIR`. |
| Неверные ссылки в API | Проверить `REGISTRY_PUBLIC_URL` и заголовки `X-Forwarded-*` на прокси. |
| Компрометация write-токена | Ротировать `REGISTRY_WRITE_TOKEN`, обновить CI и секреты; при необходимости yank затронутых версий через API. |
| Компрометация read-токена | Ротировать `REGISTRY_READ_TOKEN`, обновить всех потребителей read-path (и зеркала/кэши). |

---

## 11. Связанные документы

- [docs/registry-api.md](../docs/registry-api.md) — канонический контракт HTTP; [API.md](../API.md) — краткий указатель.
- [REFERENCE_REGISTRY_OPERATIONS_RU.md](./REFERENCE_REGISTRY_OPERATIONS_RU.md) — краткий операционный runbook (деплой/обновление, health, логи, rollback, эскалация; плейсхолдеры для board).
- [E2E_SMOKE.md](./E2E_SMOKE.md) — смоук skillget ↔ registry.
- [README.md](../README.md) — локальный запуск и переменные (кратко).
