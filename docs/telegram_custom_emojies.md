# Telegram Custom Emoji в Obsidian

## Проблема

Telegram custom emoji (`tg://emoji?id=...`) не поддерживаются в Obsidian и других markdown редакторах. Нужно было найти способ отображать их в заметках.

## Попытка 1: Base64 + Obsidian плагин

**Подход:**
- Загружать emoji через Telegram API
- Хранить как base64 на сервере
- Obsidian плагин (`telegramEmoji.ts`) заменяет `![](tg://emoji?id=...)` на inline контент

**Проблемы:**
- TGS (Lottie) требует `lottie-player` для рендера
- WEBM видео плохо работает inline
- Сложная логика с кешированием и CodeMirror decorations
- Live Preview в Obsidian не поддерживает custom protocols
- Большой размер base64 данных

**Результат:** Слишком сложно, много edge cases.

## Попытка 2: Микросервис lottie-converter

**Подход:**
- Отдельный микросервис конвертирует TGS/WEBM/WEBP → анимированный WEBP
- Простые URL: `https://ce.trip2g.com/{id}.webp`
- CSS в Obsidian для размера 20x20

**Реализация:**

1. **Telegram бот** получает custom emoji и возвращает markdown:
   ```
   ![emoji](https://ce.trip2g.com/5373112999076699207.webp)
   ```

2. **HTTP API** отдаёт WEBP по ID:
   ```
   GET https://ce.trip2g.com/{id}.webp
   ```

3. **При публикации в Telegram** происходит трансформация:
   ```
   ![emoji](https://ce.trip2g.com/4983684955683947251.webp)
   ↓
   ![](tg://emoji?id=4983684955683947251)
   ```

4. **CSS в Obsidian** фиксирует размер:
   ```css
   img[src*="ce.trip2g.com"] {
       width: 20px !important;
       height: 20px !important;
   }
   ```

## Технические решения

### Конвертация форматов

| Формат | Описание | Конвертация |
|--------|----------|-------------|
| TGS | Gzip Lottie JSON | DotLottie → PNG frames → WEBP |
| WEBM | VP9 видео | ffmpeg libwebp |
| WEBP | Статичный/анимированный | ffmpeg resize + optimize |

### Почему WEBP вместо GIF

- Лучшее качество при меньшем размере
- Полная прозрачность (не binary как в GIF)
- Нативная поддержка в Obsidian и браузерах

### Инфраструктура

- Docker образ: `alexes/lottie:v0.1`
- Домен: `ce.trip2g.com` (custom emoji)
- Кеш: `/tmp/lottiecache` (файловый)
- Webhook: `POST /{bot_token}`

## Файлы

- `lottie-converter/` - микросервис
- `obsidian-sync/styles.css` - CSS для размера emoji
- `infra/lottie.yml` - Ansible playbook

## Итог

Простое решение: микросервис + обычные URL + CSS. Работает везде где поддерживается WEBP.
