---
title: Имя файла, которое объясняет файл
free: true
lang_redirect: "[[en/thoughts/file-naming-convention]]"
---

Команда [AI Mindset](https://aimindset.org/) называет заметки в своих хранилищах по одной схеме:

```
{project} {type} title – YYYY-MM-DD – tool.md
```

Например, `{blog} {article} Weekly summary – 2026-10-18 – Cursor.md`. Проект и тип текста идут первыми, в фигурных скобках. Дата всегда стоит ближе к концу, после тире. Последняя часть называет инструмент, который написал заметку: `Cursor`, `Claude Code`, `ChatGPT` или `Manual`, если заметку писали руками.

## Почему это работает

Имя отвечает на три вопроса до того, как файл открыли: к чему он относится, что это за текст и когда его написали. Это видит человек, который листает папку. Это видит и агент, которому дали список файлов и нужно выбрать один: пока он не открыл ни одного файла, кроме имён у него часто ничего нет.

Инструмент в конце говорит, откуда взялся текст. Заметке из Claude Code и заметке, написанной руками, доверяешь по-разному.

## Почему это ломается

Правила самого соглашения называют дату в середине имени критической ошибкой и возвращаются к этому в обоих неправильных примерах. Так громко правило пишут, когда его постоянно нарушают.

Нарушают его агенты: модель каждый раз пишет имя чуть иначе. `{2026-10-18}` в фигурных скобках, обычный дефис вместо тире, подчёркивания вместо пробелов. В моём собственном хранилище по этому соглашению названы четыре заметки, и три из них получились такими: `{rule}_…_–_2025_12_18_–_Cursor.md`. Соглашение, которое живёт только в заметке, — это пожелание.

## Проверка в trip2g

В trip2g такое правило становится [[ru/user/validators|валидатором]]: заметкой с блоком Jsonnet, который при каждой синхронизации проверяет имя каждой заметки. Положите его в `validators/file_name.md`:

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

Валидатор делит имя по ` – `, то есть по тире с пробелами с обеих сторон, поэтому обычный дефис сразу не проходит. Если во frontmatter заметки есть `date`, она должна совпадать с датой в имени.

Чтобы соглашению следовало всё хранилище, не добавляйте `validate` в каждую заметку. Один [[ru/user/frontmatter_patches|frontmatter-патч]] подключает валидатор ко всем заметкам, включая новые:

````markdown
---
type: frontmatter-patch
include: ["**/*.md"]
exclude: ["validators/**", "patches/**"]
---

Все заметки хранилища следуют соглашению об именах.

```jsonnet
local own = std.get(meta, 'validate', []);
{
  validate: (if std.isString(own) then [own] else own)
    + ['[[file_name]]'],
}
```
````

Патч добавляет `file_name` к валидаторам, которые у заметки уже есть, а не заменяет их. `exclude` исключает сами валидаторы и патчи, а заодно служебные заметки, имена которых живут по другому правилу.

При следующей синхронизации заметка с неправильным именем получит сообщение:

```
[CRITICAL] validate validators/file_name.md: name must be "{project} {type} title – YYYY-MM-DD – tool"
```

CLI синхронизации завершится с кодом 1, и агент, который написал заметку, увидит, что её нужно переименовать. Админка покажет все такие заметки, а `trip2g lint` найдёт их в CI ещё до синхронизации.

## Чего он не делает

Валидатор помечает заметку, но не переименовывает её и не блокирует синхронизацию. Список инструментов вы правите сами. Заметки, которые живут по другим правилам, например ежедневные `2026-10-18.md`, идут в `exclude` или получают свой валидатор.
