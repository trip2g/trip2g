# Changelog

## 2026-01-20

- **GitHub OAuth**: исправлена ошибка "small read buffer" при валидации credentials
  - fasthttp требует явно указывать `ReadBufferSize` для API с большими заголовками (GitHub CSP)
