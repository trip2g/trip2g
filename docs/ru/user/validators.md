---
title: Валидаторы заметок
free: true
lang_redirect: "[[en/user/validators]]"
---

trip2g хранит каждую заметку в базе и работает с ней как с данными. Шаблоны, списки, поиск и API берут поля из frontmatter и разделы из текста. Карточке нужна картинка, списку — дата, сниппету в поиске — описание. Если поля нет или раздел назван иначе, страница ломается без ошибки: пустая карточка, список не в том порядке, обрезанный сниппет.

Нейросети хорошо пишут markdown, но каждый раз по-разному. В одном запуске раздел называется `## Steps`, в другом — `## How to`. В одном `tags` — список, в другом — строка.

Валидатор — это заметка с правилами на [Jsonnet](https://jsonnet.org). Вы подключаете её к другим заметкам, и при каждой синхронизации trip2g проверяет их по этим правилам. Синхронизация проходит как обычно. Заметка, которая нарушила правило, получает предупреждение, и видно, что именно исправить.

### Быстрый старт

1\. Создайте заметку `validators/required_frontmatter.md`. Правила пишутся в блоке кода `jsonnet`:

````markdown
Every article needs a title and a description, both strings.

```jsonnet
local required = ['title', 'description'];

[
  {level: 'error', msg: 'missing frontmatter field: ' + f}
  for f in required
  if !std.objectHas(meta, f)
] + [
  {level: 'error', msg: f + ' must be a string'}
  for f in required
  if std.objectHas(meta, f) && !std.isString(meta[f])
] + (
  if std.objectHas(meta, 'tags') && !std.isArray(meta.tags)
  then [{level: 'error', msg: 'tags must be a list'}]
  else []
)
```
````

2\. Подключите её к заметке через поле `validate`:

```yaml
---
title: Publishing a vault
validate: "[[required_frontmatter]]"
---
```

3\. Синхронизируйте хранилище. Если у заметки нет `description`, в выводе синхронизации появится:

```
notes/article.md
  [CRITICAL] validate validators/required_frontmatter.md: missing frontmatter field: description
```

Заметка всё равно сохранится и опубликуется. Предупреждение висит на ней, пока вы её не исправите.

### Где виден результат

- **CLI синхронизации.** Сообщения выводятся под каждой заметкой. Если есть хотя бы один `error`, CLI завершается с кодом 1, и скрипт или ИИ-агент понимает, что нужно что-то исправить. Сообщения приходят с синхронизацией, которая отправляет заметки: если ничего не изменилось, CLI ничего не отправляет и ничего не показывает. См. [[ru/user/cli|CLI для синхронизации]].
- **Админка.** Dashboard → «Предупреждения в заметках» всегда показывает все заметки с их сообщениями.
- **Локально, без сервера.** `trip2g lint <папка>` выводит те же сообщения. Он строже синхронизации: завершается с кодом 1 и на `error`, и на `warning`. Поставьте его в CI, чтобы невалидные заметки не попадали на сайт. См. [[ru/user/lint|Линтер для хранилища]].

### Как подключать валидаторы

В `validate` можно указать один валидатор или список:

```yaml
validate:
  - "[[required_frontmatter]]"
  - "[[seo]]"
  - validators/session_has_image.md
```

Валидатор можно указать вики-ссылкой или путём, с `.md` или без. Поиск работает так же, как для вики-ссылок в тексте; часть `#заголовок` игнорируется. Заметка проходит проверку, только если ни один её валидатор не вернул `error`.

#### Один валидатор на много заметок

Пишите каждое правило один раз и переиспользуйте. Держите валидаторы в одной папке, например `validators/`, по одному правилу на заметку: «обязательные поля», «SEO», «у каждой сессии есть картинка». Каждому типу заметок — свой набор. Воркшопу — `session_has_image` и `seo`. Статье — `required_frontmatter`, `seo` и `sections_order`.

Чтобы проверять целую папку, не правьте каждую заметку, а задайте `validate` [[ru/user/frontmatter_patches|frontmatter-патчем]]:

````markdown
---
type: frontmatter-patch
include: ["blog/**"]
---

```jsonnet
{ validate: ['[[required_frontmatter]]', '[[seo]]'] }
```
````

Патч заменяет собственный `validate` заметки. Чтобы дополнить список, а не заменить его, объедините их, как в эссе [[ru/thoughts/file-naming-convention|Имя файла, которое объясняет файл]]:

```jsonnet
local own = std.get(meta, 'validate', []);
{
  validate: (if std.isString(own) then [own] else own)
    + ['[[file_name]]'],
}
```

Валидаторы видят frontmatter уже после патчей, поэтому поле, которое добавил патч, считается заполненным.

### Как написать валидатор

В каждом блоке `jsonnet` валидатора доступны:

| Имя | Что в нём |
|---|---|
| `meta` | frontmatter заметки после патчей |
| `path` | путь заметки, например `blog/post.md` |
| `nodes` | текст заметки списком элементов, сверху вниз |
| `std.native('regexMatch')(pattern, text)` | `true`, если текст подходит под регулярное выражение |

У каждого элемента в `nodes` есть `type`, `line` (строка в файле, считая frontmatter) и `depth` (глубина вложенности):

| `type` | Дополнительные поля |
|---|---|
| `heading` | `level` (1–6), `text` |
| `paragraph` | `text` (текст пункта плотного списка тоже приходит как `paragraph`) |
| `image` | `src`, `alt` (и `![alt](src)`, и `![[src\|alt]]`) |
| `link` | `url`, `text` |
| `wikilink` | `target`, `embed` |
| `code_block` | `lang`, пустой, если язык не указан |
| `list` | `ordered` |
| `list_item`, `blockquote`, `table` | — |

Блок возвращает список сообщений. Пустой список значит, что с заметкой всё в порядке:

```jsonnet
[
  {level: 'error', msg: 'what is wrong', line: 12},
  {level: 'warning', msg: 'worth a look'},
]
```

- `error` помечает заметку как невалидную: `CRITICAL` в CLI, CLI завершается с кодом 1.
- `warning` — совет: `WARNING`, код выхода CLI синхронизации остаётся 0. `trip2g lint` на нём всё равно падает.
- `line` необязателен. Если он указан, сообщение его покажет.

Сломанный валидатор никогда не проходит молча. Опечатка в Jsonnet, результат не в виде списка, неизвестный `level` или ненайденный валидатор дают заметке сообщение `CRITICAL`.

В валидаторе может быть несколько блоков `jsonnet`. Выполняется каждый, их сообщения складываются.

### Примеры

Эти валидаторы взяты из демо-хранилища, которое тесты trip2g прогоняют при каждой сборке.

#### Обязательные поля

```jsonnet
local required = ['title', 'description'];

[
  {level: 'error', msg: 'missing frontmatter field: ' + f}
  for f in required
  if !std.objectHas(meta, f)
] + [
  {level: 'error', msg: f + ' must be a string'}
  for f in required
  if std.objectHas(meta, f) && !std.isString(meta[f])
] + (
  if std.objectHas(meta, 'tags') && !std.isArray(meta.tags)
  then [{level: 'error', msg: 'tags must be a list'}]
  else []
)
```

#### Разделы в заданном порядке

В статье ровно эти разделы второго уровня и в этом порядке:

```jsonnet
local want = ['Intro', 'Steps', 'FAQ'];
local got = [n.text for n in nodes if n.type == 'heading' && n.level == 2];

if got == want then []
else [{
  level: 'error',
  msg: 'sections must be %s, got %s' % [
    std.join(' → ', want),
    std.join(' → ', got),
  ],
}]
```

#### У каждой сессии есть картинка

Заметка делится на разделы по каждому `##`, и в каждом разделе считаются картинки. `std.foldl` проходит по списку элементов и несёт разделы с собой:

```jsonnet
local last(xs) = xs[std.length(xs) - 1];
local init(xs) = xs[:std.length(xs) - 1];

local sections = std.foldl(
  function(acc, n)
    if n.type == 'heading' && n.level == 2 then
      acc + [{title: n.text, line: n.line, images: 0}]
    else if n.type == 'image' && std.length(acc) > 0 then
      init(acc) + [last(acc) + {images: super.images + 1}]
    else acc,
  nodes,
  [],
);

[
  {level: 'error', line: s.line, msg: '"%s" has no image' % s.title}
  for s in sections
  if std.startsWith(s.title, 'Session') && s.images == 0
]
```

#### Картинки и код

Нет alt-текста — предупреждение. Блок кода без языка — ошибка:

```jsonnet
[
  {level: 'warning', line: n.line, msg: 'image %s has no alt text' % n.src}
  for n in nodes
  if n.type == 'image' && n.alt == ''
] + [
  {level: 'error', line: n.line, msg: 'code block without a language'}
  for n in nodes
  if n.type == 'code_block' && n.lang == ''
]
```

#### Длина описания и формат даты

```jsonnet
local re = std.native('regexMatch');
local desc = std.get(meta, 'description', '');
local date = std.toString(std.get(meta, 'date', ''));

(
  if std.length(desc) > 160
  then [{
    level: 'warning',
    msg: 'description is %d chars, keep it under 160' % std.length(desc),
  }]
  else []
) + (
  if !re('^\\d{4}-\\d{2}-\\d{2}$', date)
  then [{level: 'error', msg: 'date must be YYYY-MM-DD, got "%s"' % date}]
  else []
)
```

#### Формат имени файла

Некоторые команды называют заметки так, чтобы человек или агент понимал, что в файле, не открывая его: `{project} {type} title – YYYY-MM-DD – tool.md`. Например, `{blog} {article} Weekly summary – 2026-10-18 – Cursor.md`. Дата всегда стоит ближе к концу, а последняя часть называет инструмент, который написал заметку.

Валидатор берёт имя файла из `path`, делит его по ` – ` и проверяет каждую часть. Если во frontmatter есть `date`, она должна совпадать с датой в имени:

```jsonnet
local re = std.native('regexMatch');
local parts = std.split(path, '/');
local file = parts[std.length(parts) - 1];
local name = std.substr(file, 0, std.length(file) - 3);
local chunks = std.split(name, ' – ');
local tools = ['Claude Code', 'Cursor', 'ChatGPT', 'Manual'];

local problems =
  if std.length(chunks) != 3 then [
    'name must be "{project} {type} title – YYYY-MM-DD – tool"',
  ] else
    local head = chunks[0], date = chunks[1], tool = chunks[2];
    (
      if re('^\\{[a-z0-9]+\\} \\{[a-z]+\\} .+$', head) then []
      else ['name must start with {project} {type} and a title']
    ) + (
      if re('^\\d{4}-\\d{2}-\\d{2}$', date) then []
      else ['date "%s" must be YYYY-MM-DD' % date]
    ) + (
      if std.member(tools, tool) then []
      else ['tool "%s" must be one of: %s' % [tool, std.join(', ', tools)]]
    ) + (
      local fmDate = std.toString(std.get(meta, 'date', date));
      if fmDate == date then []
      else ['frontmatter date %s differs from %s in the name' % [fmDate, date]]
    );

[{level: 'error', msg: p} for p in problems]
```

`{blog} {2026-10-18} {article} Weekly summary.md` не проходит: дата стоит посередине, и имя не делится на три части.

Такое правило подключайте к папке frontmatter-патчем, чтобы проверялись и новые файлы. Иначе файл, созданный без `validate`, не проверяется.

### Что ещё можно проверять

- **Договор с шаблоном.** Layout читает `cover`, `description` и раздел `## Summary`. Валидатор следит, чтобы они были у каждой заметки с этим layout.
- **Заметки, которые пишут агенты.** Агент пишет заметку, запускает синхронизацию, получает код 1 со списком сообщений и исправляет заметку. Такой цикл ловит то, что модель сделала не так в конкретном запуске.
- **Данные в заметках.** События, товары, люди: `price` — число, `status` — одно из `draft`, `review`, `published`, дата в формате ISO. Списки, графики и API получают чистые данные.
- **SEO и доступность.** Один заголовок `#`, без пропуска уровней (`##`, а сразу за ним `####`), alt у картинок, описание короче 160 символов.
- **Редакционный стиль.** Запрещённые слова, ссылки с текстом «нажмите здесь», слишком длинные предложения. Текст абзаца лежит в поле `text` у элементов `paragraph`. Текста из ячеек таблиц в `nodes` нет.
- **Структура курса.** В каждом уроке есть видео и раздел `## Quiz`.
- **Посты в Telegram.** Текст укладывается в лимит, картинок не больше одной.

### Ограничения

- **Валидатор видит одну заметку.** Он не проверит, что slug уникален во всём хранилище или что на заметку ссылается другая.
- **Валидаторы не блокируют синхронизацию.** Невалидная заметка сохраняется и публикуется с предупреждением. Чтобы заблокировать её, запускайте `trip2g lint` в CI до синхронизации.
- **Заметка-валидатор публикуется, как любая другая.**
- **Результаты запоминаются между синхронизациями.** Заметка проверяется заново, только если изменилась она сама, изменился один из её валидаторов или стал указывать на другую заметку, или изменился любой frontmatter-патч. Запуск сервера, сохранение патча в админке или синхронизация с заметкой-патчем проверяют все заметки с `validate` ещё раз. С семью валидаторами на ~850 заметок такой проход занимает около 4 секунд.

Как это устроено внутри и сколько стоит: [[dev/note_validators|валидаторы заметок (dev)]].
