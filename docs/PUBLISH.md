# Как опубликовать skill в реестре

Публикация — это `POST /api/v1/skills` с multipart: поле **`manifest`** (JSON) и файл **`archive`** (`.tar.gz`). См. [registry-api.md](./registry-api.md).

## Манифест

Обязательные поля в JSON манифеста (как в теле multipart `manifest`):

| Поле | Описание |
|------|----------|
| `name` | Имя пакета (совпадает с логикой клиента / lockfile) |
| `version` | Semver, например `1.2.3` |
| `description` | Краткое описание |
| `author` | Автор или организация |

Дополнительные поля (например `dependencies`) сохраняются в записи версии как часть manifest.

## Архив

- Формат: **gzip-сжатый tar** (`.tar.gz`).
- В корне архива обычно лежит `SKILL.md` (и при необходимости другие файлы скилла).
- Пример исходников: каталог [`examples/hello-skill`](../examples/hello-skill/).
- Сборка готового архива: из корня репозитория выполните `scripts/pack-example.sh` — результат в `examples/dist/hello-skill-0.1.0.tar.gz`.

## Локальная публикация (curl)

1. Запустите сервер с включённой записью:

   ```bash
   export REGISTRY_WRITE_TOKEN='ваш-секрет-только-в-окружении'
   go run ./cmd/registry
   ```

2. Опубликуйте (подставьте путь к архиву и манифесту):

   ```bash
   MANIFEST="$(cat examples/hello-skill/manifest.json)"
   curl -sS -X POST "http://127.0.0.1:8080/api/v1/skills" \
     -H "Authorization: Bearer $REGISTRY_WRITE_TOKEN" \
     -F "manifest=$MANIFEST;type=application/json" \
     -F "archive=@examples/dist/hello-skill-0.1.0.tar.gz"
   ```

Ожидаемый ответ: HTTP **201 Created**.

## Секреты и CI

- **Токен записи реестра** (`REGISTRY_WRITE_TOKEN` на стороне сервера; для клиента — Bearer в `Authorization`) **не** коммитьте и не вставляйте в тикеты.
- **GitHub PAT для org getskillpack** (`GETSKILLPACK_ORG_PAT`) задаётся только в окружении раннера/оператора — см. workflow «getskillpack org (manual)» и внутреннюю документацию skpkg-cli.
- В URL `git remote` **не** вшивайте токены; для HTTPS-пуша используйте кратковременно `GH_TOKEN` в сессии или SSH-ключ.

## Готовый пример в репозитории

В репозитории лежит собранный архив [`examples/dist/hello-skill-0.1.0.tar.gz`](../examples/dist/hello-skill-0.1.0.tar.gz) — его можно загрузить в свой стенд тем же `curl`, не собирая tarball заново.
