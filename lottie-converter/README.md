# Lottie to WebM Converter

Микросервис для конвертации Lottie анимаций в WebM видео с поддержкой прозрачности.

## Запуск

```bash
# Из корня проекта
docker-compose up -d lottie-converter

# Проверка здоровья
curl http://localhost:3000/health
```

## Использование

```bash
# Тест с anim.json
curl -X POST http://localhost:3000/convert \
  -H "Content-Type: application/json" \
  -d @anim.json \
  --output test.webm

# Или с inline JSON
curl -X POST http://localhost:3000/convert \
  -H "Content-Type: application/json" \
  -d '{"animation": {...}, "width": 512, "height": 512, "fps": 30}' \
  --output output.webm
```

## API

**POST /convert**
- Body:
  - `animation` (required) - Lottie JSON объект
  - `width` (optional, default: 800) - ширина видео
  - `height` (optional, default: 600) - высота видео
  - `fps` (optional, default: 30) - FPS
- Response: video/webm файл

**GET /health**
- Response: `{"ok": true}`

## Технологии
- Node.js + Express
- Puppeteer для рендеринга Lottie
- FFmpeg для конвертации в WebM с VP9
- Поддержка прозрачности (alpha channel)
