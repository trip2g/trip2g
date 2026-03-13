---
free: true
title: Самостоятельная установка
---
## Требования

**Минимальные ресурсы:**
- 128 МБ оперативной памяти
- 500 МБ свободного места на диске

**Установите Docker:**
- Следуйте инструкции: https://docs.docker.com/engine/install/
- После установки проверьте: `docker --version`

**Docker Compose:**
- В новых версиях Docker (20.10+) входит в состав: `docker compose`
- Если у вас старая версия, установите отдельно: https://docs.docker.com/compose/install/
- Проверьте: `docker compose version` или `docker-compose --version`

В инструкции ниже используется команда `docker-compose`. Если у вас новая версия Docker, замените на `docker compose` (без дефиса).

## Три шага

### 1. Создайте docker-compose.yml

Создайте файл `docker-compose.yml` со следующим содержимым:

```yaml
version: '3.8'

services:
  app:
    image: alexes/trip2g:v10
    container_name: trip2g-app
    restart: unless-stopped
    ports:
      - "7777:7777"
    env_file:
      - .env
    environment:
      - LISTEN_ADDR=:7777
      - INTERNAL_LISTEN_ADDR=:7778
      - DB_FILE=/app/data/data.sqlite3
      - MINIO_ENDPOINT=minio:9000
      - MINIO_USE_SSL=false
    volumes:
      - app-data:/app/data
    depends_on:
      minio:
        condition: service_healthy
    networks:
      - trip2g-network

  minio:
    image: minio/minio:latest
    container_name: trip2g-minio
    restart: unless-stopped
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    command: server /data --console-address ":9001"
    volumes:
      - minio-data:/data
    networks:
      - trip2g-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s

volumes:
  app-data:
    driver: local
  minio-data:
    driver: local

networks:
  trip2g-network:
    driver: bridge
```

### 2. Создайте .env

Создайте файл `.env` и укажите настройки:

```bash
# Сгенерируйте секретный ключ: openssl rand -hex 32
JWT_SECRET=ваш-секретный-ключ-32-символа

# Придумайте логин и пароль для MinIO (минимум 8 символов)
MINIO_ROOT_USER=admin
MINIO_ROOT_PASSWORD=ваш-пароль-минимум-8-символов

# Скопируйте сюда те же значения
MINIO_ACCESS_KEY_ID=admin
MINIO_SECRET_KEY=ваш-пароль-минимум-8-символов

# Укажите ваш email для входа
OWNER_EMAIL=admin@example.com

# Для отправки писем зарегистрируйтесь на https://resend.com/ (бесплатный тариф)
RESEND_API_KEY=
MAIL_FROM=no-reply@example.com
```

### 3. Запустите

```bash
docker-compose up
```

Откройте:
- Приложение: http://localhost:7777
- MinIO Console: http://localhost:9001

## Управление

Запустить в фоне:
```bash
docker-compose up -d
```

Посмотреть логи:
```bash
docker-compose logs
```

Остановить:
```bash
docker-compose down
```

Перезапустить:
```bash
docker-compose restart
```

## Резервные копии

Сохранить базу:
```bash
docker cp trip2g-app:/app/data/data.sqlite3 ./backup-$(date +%Y%m%d).sqlite3
```

Сохранить файлы:
```bash
docker run --rm \
  -v simple_minio-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/minio-backup-$(date +%Y%m%d).tar.gz /data
```

Восстановить базу:
```bash
docker cp ./backup.sqlite3 trip2g-app:/app/data/data.sqlite3
docker-compose restart app
```

Восстановить файлы:
```bash
docker run --rm \
  -v simple_minio-data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/minio-backup.tar.gz -C /
docker-compose restart minio
```

## Настройки

Все настройки в `.env`. Обязательные:

```bash
JWT_SECRET=секрет-32-символа

MINIO_ROOT_USER=admin
MINIO_ROOT_PASSWORD=пароль-минимум-8
MINIO_ACCESS_KEY_ID=admin
MINIO_SECRET_KEY=пароль-минимум-8

OWNER_EMAIL=admin@example.com

# для отправки писам, регистрация на resend.com (бесплатный тариф)
RESEND_API_KEY= 
```

## Порты

- 7777 — приложение
- 9000 — MinIO API
- 9001 — MinIO Console

Изменить порт приложения в `docker-compose.yml`:
```yaml
ports:
  - "8080:7777"
```

## Production

1. Смените все пароли
2. Сгенерируйте JWT_SECRET: `openssl rand -hex 32`
3. Делайте бэкапы
