---
title: "robots.txt и llms.txt"
description: "Публикуйте robots.txt и llms.txt как обычные заметки с полем content_type во frontmatter. Никаких специальных настроек не требуется."
free: true
lang_redirect: "[[en/user/robots_and_llm]]"
---

`robots.txt` и `llms.txt` — это текстовые файлы, которые поисковые роботы и AI-агенты ищут по стандартным путям. В trip2g их публикуют как обычные заметки с полем [[ru/user/content_type|content_type]] во frontmatter.

### robots.txt

Создайте заметку с таким frontmatter:

```yaml
---
slug: /robots.txt
content_type: text/plain; charset=utf-8
free: true
search: false
---
User-agent: *
Disallow:

Sitemap: https://yourdomain.com/sitemap.xml
```

Замените `https://yourdomain.com` на публичный URL вашего сайта. Строка `Sitemap:` подсказывает роботам, где найти карту сайта — используйте абсолютный URL, а не относительный путь. trip2g генерирует sitemap автоматически по адресу `/sitemap.xml`.

`slug: /robots.txt` переопределяет URL заметки, чтобы она была доступна по тому пути, который роботы проверяют первым.

Сайты без заметки `robots.txt` возвращают 404 для этого пути. Поисковики трактуют отсутствующий `robots.txt` как «всё разрешено», поэтому без заметки всё индексируется.

### llms.txt

`llms.txt` — текстовое описание сайта для AI-агентов и языковых моделей. Создайте заметку:

```yaml
---
slug: /llms.txt
content_type: text/plain; charset=utf-8
free: true
search: false
---
# Мой сайт

Краткое описание, чему посвящён сайт.

## Ключевые страницы
- Документация: https://yourdomain.com/docs
- О проекте: https://yourdomain.com/about

## Факты для AI-ассистентов
- О чём сайт и для кого.
```

Напишите всё, что AI-агентам нужно знать о вашем сайте. Тело заметки отдаётся как есть.

### Как это работает

Подробное объяснение — в [[ru/user/content_type|заметках с произвольным Content-Type]]. Коротко: `content_type` во frontmatter устанавливает HTTP-заголовок `Content-Type` и заставляет тело заметки отдаваться дословно, без HTML-обёртки.
