---
title: Типизированная база знаний по образцу ontoship
free: true
lang_redirect: "[[en/thoughts/typed-knowledge-base]]"
---

[ontoship](https://github.com/vakovalskii/ontoship) Валерия Ковальского — база знаний из трёх вещей: markdown-файлов, индекса `README.md` в каждой папке и git. Его CLI, `gitmark`, строит из тех же файлов поисковый индекс, HTML-граф и отчёты. Сверху лежит необязательная онтология, и именно её стоит позаимствовать: у каждого документа есть тип, а линтер следит, чтобы база не теряла форму по мере роста. Это эссе показывает, сколько из этого хранилище trip2g умеет уже сегодня и где оно останавливается.

## Идея

Онтология взята у Palantir, который описывает организацию объектами, их свойствами и связями между ними. ontoship применяет те же три сущности к документации о коде:

- **Объект** — это документ. Его `node_type` берётся из фиксированного списка: `service`, `reference`, `runbook`, `gotcha`, `decision`, `plan`, `guide`, `report`, `index`.
- **Свойства** — это frontmatter: `title`, `description`, `service`, `status`, `updated` и поля доверия из Open Knowledge Format от Google: `generated` (кто написал), `verified` (кто сверил с реальностью) и `stale_after`.
- **Связи** типизированы и лежат под `links:`: `documents`, `depends_on`, `supersedes`, `relates_to`, `implemented_by`, `part_of`.

`gitmark lint` проверяет восемь инвариантов:

| | Инвариант |
|---|---|
| I1 | у каждого несущего документа есть допустимый `node_type` |
| I2 | `node_type`, `service` и `status` берутся из своих словарей |
| I3 | нет сирот: у документа есть хотя бы одна связь, входящая или исходящая |
| I4 | нет битых ссылок |
| I5 | в каждой папке внутри `docs/` есть `README.md` |
| I6 | цель `supersedes` помечена `deprecated` или `archived` |
| I7 | цель `documents` или `implemented_by` существует в коде |
| I8 | срок `stale_after` не прошёл |

Смысл тот же, что у любой схемы. Документ с типом ещё до чтения говорит человеку и агенту, что он такое, а проверяемая база не превращается в кучу файлов.

## То же на trip2g

Хранилище trip2g — тоже папка с markdown, поэтому части ложатся почти один к одному. Примеры ниже взяты из маленького демо-хранилища, которое загружают тесты trip2g: шесть заметок в `docs/`, один патч, один валидатор и один запрос.

### Тип по папке

ontoship держит каждый тип в своей папке: решения в `docs/decisions/`, runbook-и в `docs/ops/`. Тогда тип может задавать папка, и писать его руками не нужно. [[ru/user/frontmatter_patches|Патч frontmatter]] делает это для каждой заметки в `docs/`:

````
---
type: frontmatter-patch
include: ["docs/**"]
---

A note's folder gives its default node_type, and every note under docs/ is
checked by the ontology validator.

```jsonnet
local byFolder = {
  decisions: 'decision', ops: 'runbook', plans: 'plan', reference: 'reference',
};
local parts = std.split(path, '/');
local file = parts[std.length(parts) - 1];
local folder = if std.length(parts) > 2 then parts[1] else '';
local own = std.get(meta, 'validate', []);

{
  validate: (if std.isString(own) then [own] else own) + ['[[ontology]]'],
} + (
  if std.objectHas(meta, 'node_type') then {}
  else if file == 'README.md' then { node_type: 'index' }
  else if std.objectHas(byFolder, folder) then { node_type: byFolder[folder] }
  else {}
)
```
````

`node_type`, записанный в самой заметке, важнее. `README.md` становится `index`. Тот же патч подключает к каждой заметке в `docs/` валидатор ниже и добавляет его к валидаторам, которые у заметки уже есть.

### I1 и I2: валидатор

Сами правила лежат в [[ru/user/validators|валидаторе]] `validators/ontology.md`:

```jsonnet
local re = std.native('regexMatch');
local nodeTypes = [
  'service', 'reference', 'runbook', 'gotcha', 'decision',
  'plan', 'guide', 'report', 'index',
];
local bearing = ['service', 'reference', 'runbook', 'plan', 'decision'];
local statuses = ['active', 'draft', 'deprecated', 'archived'];
local linkTypes = [
  'documents', 'depends_on', 'supersedes',
  'relates_to', 'implemented_by', 'part_of',
];
local err(msg) = {level: 'error', msg: msg};
local warn(msg) = {level: 'warning', msg: msg};
local type = std.get(meta, 'node_type', '');

if type == '' then [err('missing node_type')]
else if !std.member(nodeTypes, type) then [
  err('node_type "%s" is not one of: %s' % [type, std.join(', ', nodeTypes)]),
] else [
  err('a %s needs %s' % [type, f])
  for f in ['title', 'description', 'status', 'updated']
  if std.member(bearing, type) && !std.objectHas(meta, f)
] + [
  warn('status "%s" is not one of: %s' % [s, std.join(', ', statuses)])
  for s in [std.get(meta, 'status', 'active')]
  if !std.member(statuses, s)
] + [
  warn('unknown link type "%s"' % k)
  for k in std.objectFields(std.get(meta, 'links', {}))
  if !std.member(linkTypes, k)
] + [
  err('verified.by "%s" must be human:<id>, process:<id> or tool/version'
      % v.by)
  for v in std.get(meta, 'verified', [])
  if !re('^((human|process):[a-z0-9_-]+|[a-z0-9_-]+/[a-z0-9._-]+)$',
         std.get(v, 'by', ''))
]
```

Он проверяет, что у заметки есть тип из списка; что у несущего типа есть заголовок, описание, статус и дата обновления; что `status` и типы связей берутся из своих словарей; что каждая запись в `verified` называет человека, процесс или инструмент с версией. Неверный статус или неизвестный тип связи — предупреждение, отсутствующий тип или поле — ошибка.

В демо runbook без описания, со `status: live`, связью `see_also` и `verified: [{ by: claude }]` получает при синхронизации четыре сообщения:

```
[CRITICAL] validate validators/ontology.md: a runbook needs description
[WARNING] validate validators/ontology.md: status "live" is not one of: active, draft, deprecated, archived
[WARNING] validate validators/ontology.md: unknown link type "see_also"
[CRITICAL] validate validators/ontology.md: verified.by "claude" must be human:<id>, process:<id> or tool/version
```

ontoship хранит словари во frontmatter своего `ontology.md`. Здесь они живут в валидаторе: валидатор видит только проверяемую заметку и прочитать другую не может. Словарь `service` у каждого проекта свой, поэтому в демо его нет; это был бы ещё один список того же вида.

### I4: уже есть

trip2g разрешает каждую wikilink-ссылку в тексте заметки, а ссылку в никуда показывает как `broken link` — в админке и в `trip2g lint`. Правило не нужно.

### I5, I6 и сводка здоровья: запрос

`gitmark health` — сводка по всей базе: доля документов с типом, документы без описания и так далее. В trip2g это [[ru/user/note-queries|запрос к заметкам]]: заметка с программой на Jsonnet, которая получает путь и frontmatter (после патчей) каждой заметки, доступной вызывающему, и возвращает любой JSON. Демо запускает его от админа, которому доступны все. `queries/kb_health.md`:

````
---
type: note-query
---

```jsonnet
local docs = [n for n in notes if std.startsWith(n.path, 'docs/')];
local parts(p) = std.split(p, '/');
local dir(p) = std.join('/', parts(p)[:std.length(parts(p)) - 1]);
local name(p) = std.strReplace(parts(p)[std.length(parts(p)) - 1], '.md', '');
local target(link) = std.split(std.stripChars(link, '[]'), '|')[0];
local byName = { [name(n.path)]: n for n in docs };
local paths = [n.path for n in docs];
local done = ['deprecated', 'archived'];

{
  docs: std.length(docs),
  typed: std.length([n for n in docs if std.objectHas(n.meta, 'node_type')]),
  no_description: [
    n.path for n in docs
    if std.objectHas(n.meta, 'node_type')
    && n.meta.node_type != 'index'
    && !std.objectHas(n.meta, 'description')
  ],
  no_readme: [
    d for d in std.set([dir(p) for p in paths])
    if !std.member(paths, d + '/README.md')
  ],
  superseded_but_active: [
    { note: byName[t].path, replaced_by: n.path }
    for n in docs
    for t in [
      target(l)
      for l in std.get(std.get(n.meta, 'links', {}), 'supersedes', [])
    ]
    if std.objectHas(byName, t)
    && !std.member(done, std.get(byName[t].meta, 'status', ''))
  ],
}
```
````

На демо-хранилище он возвращает:

```json
{
  "docs": 6,
  "typed": 5,
  "no_description": ["docs/ops/restart-db.md"],
  "no_readme": ["docs/decisions", "docs/ops", "docs/plans"],
  "superseded_but_active": [
    {
      "note": "docs/decisions/old-sandbox.md",
      "replaced_by": "docs/decisions/drop-firecracker.md"
    }
  ]
}
```

`no_readme` — это I5, `superseded_but_active` — I6: старую схему песочницы заменили, но никто не пометил её `deprecated`. Страница показывает отчёт через `noteQuery("[[kb_health]]")` в шаблоне, а скрипт получает его GraphQL-запросом `noteQuery`. Тот же механизм отвечает на вопросы в духе `gitmark query`, например «все решения, кроме устаревших»: это фильтр по `notes` и их `meta`.

Для I3 нужны ссылки, а запрос их не получает. Их получает [[ru/user/templates|шаблон]]: `nvs.BackLinks(note)` и `nvs.OutLinks(note)` перечисляют входящие и исходящие ссылки заметки, так что страница-отчёт может вывести заметки, у которых пусто и там, и там.

## Где это останавливается

**I3, I5 и I6 — отчёт, а не пометка.** Валидатор помечает проверяемую заметку, а CLI синхронизации при ошибке завершается с кодом 1, так что агент видит проблему сразу. Запрос или шаблон отвечают, только когда кто-то открывает страницу. Чтобы помечать заметки по правилам над всем хранилищем, trip2g нужен валидатор, который видит все заметки. Он запланирован, обсуждение — в [issue #436](https://github.com/trip2g/trip2g/issues/436).

**I8 пока невозможен.** Ни валидатор, ни запрос не знают текущую дату. У валидаторов это сделано намеренно: их результаты кешируются, а результат, который зависит от часов, кешировать нельзя. Время запланировано для валидаторов хранилища из того же issue.

**I7 неприменим.** В хранилище trip2g лежат заметки, а не код, который они описывают, так что путь из `documents` сверять не с чем.

## Что это даёт

Правила живут в хранилище, рядом с заметками, которые проверяют: патч, валидатор и запрос — тоже заметки, они синхронизируются и версионируются вместе с остальными. Новая заметка в `docs/decisions/` получает тип и проверку при первой же синхронизации, настраивать ничего не нужно. Агент, который написал runbook без описания, получает ошибку в выводе синхронизации, а человек видит тот же список в админке.
