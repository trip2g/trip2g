---
title: "Непрерывный бэкап и read-реплики через Litestream"
free: true
lang_redirect: "[[en/user/litestream]]"
---

[Litestream](https://litestream.io) реплицирует вашу SQLite-базу в S3 **непрерывно** — каждая запись стримится в объектное хранилище — вместо периодических полных снапшотов, которые делает встроенный [[backup|простой бэкап]] trip2g. Упал сервер — теряете секунды, а не час. Запускается маленьким **sidecar** рядом с trip2g; внутрь ничего не вкомпилировано.

## Запуск рядом с trip2g

Litestream следит за WAL вашего SQLite-файла и отгружает его в S3. Два флага trip2g делают их совместимыми:

```sh
trip2g --vacuum-cron=false --simple-backup=false ...
```

- `--vacuum-cron=false` — **самый важный** (и это значение по умолчанию). Опциональная maintenance-задача trip2g делает `wal_checkpoint(TRUNCATE)` + `VACUUM`. Litestream **должен быть единственным, кто трогает WAL**: `TRUNCATE`-чекпойнт может срезать WAL-фреймы, которые Litestream ещё не реплицировал, а `VACUUM` перезаписывает всю базу в новую генерацию Litestream. Поэтому с Litestream никогда не включайте `--vacuum-cron`. (Litestream сам чекпойнтит; vacuum он **не** делает, и он вам не нужен.)
- `--simple-backup=false` — не запускайте ещё и S3-снапшот-бэкап trip2g. Litestream **и есть** ваш бэкап; держать оба избыточно.

Минимальный конфиг (`litestream.yml`):

```yaml
dbs:
  - path: /data/data.sqlite3
    replicas:
      - type: s3
        endpoint: https://your-s3-or-minio
        bucket: trip2g-backups
        path: data.sqlite3
```

Запустите `litestream replicate -config litestream.yml` рядом с trip2g. Восстановление на свежем сервере: `litestream restore -config litestream.yml /data/data.sqlite3` **до** старта trip2g.

## Шифрование

Litestream 0.3.13+ поддерживает нативное клиентское **age-шифрование** (E2E) — простейший способ закрыть требование «бэкапы зашифрованы at-rest и in-transit» для комплаенса. Добавьте `age`-ключ в конфиг реплики; бакет видит только шифртекст.

## Дальше бэкапа: read-реплики

Новый **[VFS](https://fly.io/blog/litestream-vfs/)** в Litestream умеет читать базу *прямо из S3-бэкапа* без скачивания — это near-realtime **read-only реплика**, удобно для «читать прод не трогая прод» и point-in-time запросов, хотя per-page S3-запросы дают латентность (не для горячего чтения).

Для живых низколатентных read-реплик между машинами (чтение локально, записи форвардятся на primary, авто-выбор primary) есть соседний проект **[LiteFS](https://fly.io/docs/litefs/)** ([введение](https://fly.io/blog/introducing-litefs/)) — FUSE-ФС, реплицирующая SQLite по кластеру. Ложится естественно на single-writer модель trip2g и на [[zerodowntime|деплой без простоя]] (реплики держат чтение, пока primary заменяется). Это основа SQLite-приложений на [Fly.io](https://fly.io/docs/litefs/).

## Какой бэкап выбрать?

Полное сравнение — в [[backup]]. Коротко: **простой бэкап** (встроенные S3-снапшоты) — дефолт без настройки; **Litestream** — апгрейд, когда нужна непрерывная репликация / минимальная потеря данных, и предпосылка для read-реплик выше.
