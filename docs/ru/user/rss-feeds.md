---
free: true
title: RSS-ленты
lang_redirect: "[[en/user/rss]]"
---

RSS в trip2g — это заметка, которую рендерит Jet-шаблон. Никакого встроенного модуля нет: вся реализация — около 20 строк разметки, которые можно прочитать, скопировать и переделать под любой формат.

Это принцип «всё есть заметка» в действии. Заметка-фид содержит frontmatter, который управляет тем, какие заметки попадают в ленту и сколько их. Движок передаёт эти поля в `_layouts/rss.html` — Jet-layout, который лежит прямо в вашем хранилище Obsidian.

Редактируйте `_layouts/rss.html` как любой другой файл и синхронизируйте. Шаблон поставляется в [[ru/user/onboarding-vault|стартовом архиве]], а исходный код опубликован на [github.com/trip2g/trip2g — rss.html](https://github.com/trip2g/trip2g/blob/main/onboarding-vault/_layouts/rss.html).

[[ru/user/kanban|Kanban]], [[ru/user/theme-editor|редактор тем]] и RSS устроены одинаково: layout-файлы внутри хранилища, которые редактируются прямо в Obsidian.

### Настройка фида

Создайте заметку — в стартовом архиве она уже есть (`feed.md`) — с таким frontmatter:

```yaml
---
slug: /feed.xml
content_type: application/rss+xml; charset=utf-8
layout: rss
free: true
rss_title: Лента моего сайта
rss_description: Последние заметки
rss_glob: "**"
rss_limit: 20
---
```

Откройте `/feed.xml` — фид готов. В него попадают только публично читаемые заметки: без sign-in-гейта, без системных, только с `free: true`.

| Поле | Что делает | По умолчанию |
|------|------------|--------------|
| `rss_glob` | Какие заметки включать (glob, например `"blog/**"`) | `**` (все) |
| `rss_limit` | Максимум элементов | `20` |
| `rss_title` | Заголовок ленты | Заголовок заметки |
| `rss_description` | Описание ленты | Описание заметки |

### Кастомизация шаблона

Откройте `_layouts/rss.html` в Obsidian и редактируйте. Текущий шаблон генерирует RSS 2.0 с полным телом в `<content:encoded>`. Добавьте `<author>`, `<category>` или `<enclosure>` для подкаст-аудио. Чтобы получить другой формат — Atom, JSON Feed, карту сайта — напишите новый layout-файл и укажите его в `layout:` frontmatter заметки-фида.

Чтобы вести несколько фидов с разным охватом, создайте несколько заметок. Например, `/blog-feed.xml` с `rss_glob: "blog/**"` и `/podcast.rss` с `rss_glob: "episodes/**"`.
