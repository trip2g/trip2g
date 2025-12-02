# Lottie Converter

Микросервис для конвертации Telegram custom emoji (TGS/WEBM/WEBP) в оптимизированный анимированный WEBP для использования в Obsidian и других markdown редакторах.

## TODO

 - [ ] Сделать защиту от спама, иначе можно перебрать все emoji и забить диск.

## Возможности

- ✅ Конвертация TGS (Lottie) → WEBP
- ✅ Конвертация WEBM → WEBP
- ✅ Оптимизация WEBP → WEBP
- ✅ Telegram бот для получения markdown кодов
- ✅ HTTP API для прямого доступа к изображениям
- ✅ Файловый кеш для быстрого доступа
- ✅ Webhook поддержка

## Преимущества WEBP

- Лучшее качество при меньшем размере файла
- Полная поддержка прозрачности (alpha channel)
- Поддержка анимации
- Нативная поддержка в Obsidian и современных браузерах

## Запуск

### Docker Compose (локальная разработка)

```bash
# Из корня проекта
docker-compose up -d lottie-converter

# Проверка здоровья
curl http://localhost:3000/health
```

### Docker (production)

```bash
docker run -d \
  --name lottie-converter \
  -p 3000:3000 \
  -v /var/lib/lottie-converter/cache:/tmp/lottiecache \
  --env-file /etc/lottie.env \
  alexes/lottie:v0.1
```

Файл `/etc/lottie.env`:
```
TELEGRAM_BOT_TOKEN=your_bot_token_here
SERVER_URL=https://your-domain.com
NODE_ENV=production
```

## Использование

### Telegram Bot

1. Отправьте боту сообщение с custom emoji
2. Получите markdown коды для Obsidian:

```markdown
![emoji](https://your-domain.com/5373112999076699207.webp)
```

### HTTP API

**GET /:id.webp**
- Возвращает анимированный WEBP для custom emoji
- Автоматически конвертирует и кеширует при первом запросе
- Cache-Control: 1 год

**GET /health**
- Response: `{"ok": true, "cache": {"count": 42, "sizeBytes": 1234567}}`

**POST /:bot_token** (webhook)
- Telegram webhook endpoint

## Технологии

- Node.js + Express
- @lottiefiles/dotlottie-web для рендеринга TGS
- @napi-rs/canvas для рендеринга кадров
- FFmpeg (libwebp) для конвертации в WEBP
- Telegraf для Telegram бота
- Файловый кеш в `/tmp/lottiecache`

## Deployment

См. [`infra/lottie.yml`](../infra/lottie.yml) для Ansible playbook
