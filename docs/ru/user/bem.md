---
free: true
title: "BEM-именование в шаблонах"
lang_redirect: "[[en/user/bem]]"
---

BEM — соглашение об именовании CSS-классов, которое не даёт стилям одного компонента случайно затирать стили другого. В шаблонах trip2g BEM — рекомендуемый способ именовать классы внутри блоков `_style_*`.

### Что такое BEM

BEM расшифровывается как Блок, Элемент, Модификатор. Каждый класс — один из трёх видов:

- **Блок** — самостоятельный компонент: `.card`, `.hero`, `.button`
- **Элемент** — часть блока, соединяется через `__`: `.card__title`, `.card__image`
- **Модификатор** — вариант блока или элемента, соединяется через `--`: `.card--featured`, `.button--primary`

Вся система — только это. Никакого контекстного вложения в именах.

Ключевые правила:

- **Плоские имена элементов** — глубина DOM не отражается в имени класса: `.card__body` не `.card__content__body`
- **Никаких глобальных модификаторов** — `.card--hidden` не `.hidden` (глобальные классы конфликтуют между компонентами)
- **Миксин** — один элемент может нести классы двух блоков: `class="button nav__item"` применяет стили обоих блоков

### Почему шаблоны trip2g используют BEM

Компоненты шаблонов загружаются независимо, а их CSS собирается постранично через `yield_blocks`. Без соглашения об именовании два компонента легко могут объявить `.title` или `.wrapper` и перезаписать стили друг друга.

BEM решает это: каждое имя класса уникально для своего блока. `.card__title` и `.hero__title` не конфликтуют, даже если оба блока оказываются на одной странице.

### Именование в файлах компонентов

Каждый файл компонента в `_layouts/ваша-тема/components/` следует такому шаблону:

```
_style_имяблока   ← CSS-блок, который собирает yield_blocks
имяблока          ← HTML-блок, который рендерит yield
```

Имя блока в CSS-классах совпадает с именем компонента. Если компонент называется `card`, все его классы начинаются с `card`:

```html
{{block _style_card()}}
.card { border: 1px solid #eee; border-radius: 6px; padding: 16px; }
.card__title { font-size: 1.25rem; font-weight: bold; margin-bottom: 8px; }
.card__body { color: #555; line-height: 1.5; }
.card--featured { border-color: #0070f3; }
{{end}}

{{block card(title="", body="", featured=false)}}
<div class="card{{if featured}} card--featured{{end}}">
  <h2 class="card__title">{{title}}</h2>
  <p class="card__body">{{body}}</p>
</div>
{{end}}
```

Блок стилей называется `_style_card` — префикс `_style_` говорит `yield_blocks`, что его нужно собрать. HTML-блок называется `card` — суффикс совпадает, и загрузчик автоматически связывает их между собой.

### Соглашение `_style_имяблока`

Префикс `_style_` не случаен. `yield_blocks("_style_")` находит каждый блок, чьё имя начинается с `_style_`, и записывает их CSS в один тег `<style>`. Имя `_style_card` вместо `card_css` или `styles_card` делает блок обнаруживаемым загрузчиком без дополнительной настройки.

BEM-именование классов внутри этих блоков гарантирует отсутствие коллизий, когда CSS разных компонентов оказывается на одной странице.

### Полный пример

Три компонента на одной странице, все с BEM:

```html
{{block _style_hero()}}
.hero { padding: 80px 24px; text-align: center; background: #f5f5f5; }
.hero__title { font-size: 2.5rem; font-weight: 700; }
.hero__subtitle { color: #666; margin-top: 8px; }
{{end}}

{{block _style_button()}}
.button { display: inline-flex; padding: 10px 20px; border-radius: 4px; }
.button--primary { background: #0070f3; color: #fff; }
.button--ghost { border: 1px solid #0070f3; color: #0070f3; }
{{end}}

{{block _style_card()}}
.card { border: 1px solid #eee; border-radius: 6px; padding: 16px; }
.card__title { font-weight: bold; }
.card--featured { border-color: #0070f3; }
{{end}}
```

`yield_blocks("_style_")` собирает все три в один тег `<style>`. Поскольку каждый класс ограничен именем своего блока, конфликтов нет.

### Что не стоит делать

Избегайте общих имён классов внутри CSS-блоков компонентов — они столкнутся, как только два компонента окажутся на одной странице:

```html
{{block _style_card()}}
/* Плохо: .title, .body, .wrapper — слишком общие */
.title { font-weight: bold; }
.wrapper { padding: 16px; }

/* Хорошо: имена привязаны к блоку */
.card__title { font-weight: bold; }
.card { padding: 16px; }
{{end}}
```

### `@lid` / `@did` для уникальности имён блоков

Имена блоков в Jet глобальны. Если два файла компонентов оба определяют блок с именем `hero`, один молча перезапишет другой. BEM делает уникальными имена CSS-классов, но не имена блоков Jet.

`@lid` решает проблему на уровне имён блоков. Перед разбором шаблона движок заменяет `@lid` на значение с подчёркиваниями, производное от пути к файлу. `@did` делает то же самое с дефисами — для CSS-классов:

- `components/button.html` → `@lid` = `components_button`, `@did` = `components-button`
- `ui/nav/header.html` → `@lid` = `ui_nav_header`, `@did` = `ui-nav-header`

Рекомендуемый паттерн сочетает BEM-именование классов через `@did` с `@lid` в именах блоков:

```html
{{block _style_@lid()}}
.@did { display: inline-flex; padding: 8px 20px; }
.@did--primary { background: #0070f3; color: #fff; }
{{end}}

{{block @lid(label="Нажмите", variant="")}}
<button class="@did{{if variant}} @did--{{variant}}{{end}}">{{label}}</button>
{{end}}
```

BEM делает уникальными CSS-классы на странице. `@lid` делает уникальными имена блоков Jet во всех файлах компонентов. Используйте оба подхода в каждом файле компонента.

### Смотрите также

- [[ru/user/yield_blocks|yield_blocks: CSS и JS по страницам]] — как собираются блоки стилей и полная документация по `@lid`/`@did`
- [[ru/user/templates|Шаблоны]] — основы шаблонов и синтаксис Jet
