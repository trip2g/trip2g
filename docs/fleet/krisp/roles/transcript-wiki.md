---
description: "Segments → knowledge note with [[WikiLinks]] (step 2 of 2)"
model: qwen/qwen3-14b
tools: [read_note, write_note]
read_patterns: ["segments/**", "transcripts/**"]
write_patterns: ["qwen3-14b/wiki/**"]
fleet_id: llm
mode: change
trigger_on: [create, update]
trigger_include: ["segments/**"]
trigger_exclude: ["wiki/**"]
for_each: changed_files
max_depth: 3
concurrency: skip
max_steps: 8
max_tokens: 16000
---
Дана карта сегментов записи (по темам), полученная на шаге сегментации: `{{ change_file.Path }}`. Извлеки из записи пригодное для базы знаний сырьё. Выход — markdown-заметка для vault, где все сущности оформлены как `[[WikiLinks]]` для графа.

Карта сегментов задаёт структуру (темы и временные окна). Для деталей **прочитай исходный транскрипт** инструментом `read_note` по пути из фронтматтера `source_transcript` карты сегментов.

Когда готово, **запиши результат** инструментом `write_note` по пути `qwen3-14b/wiki/` + basename файла сегментов (модель namespace'ится для сравнения прогонов). Строгий формат:

---
id: <basename без .md>
source_segments: "{{ change_file.Path }}"
kind: transcript-wiki
---

## Суть
1–3 предложения: о чём запись, главный итог.

## Решения и идеи
- **[Решение/идея одним предложением]** — контекст/обоснование, цифры если есть.

## Открытые вопросы
- Что осталось нерешённым / на проверку.

## Действия
- [ ] Что наметили сделать (с владельцем, если ясно).

## Сущности
Все упомянутые проекты, люди, инструменты, каналы, концепции — как `[[WikiLinks]]` с коротким контекстом, для графа vault.
`[[Название]]` — что это / почему всплыло.

Карта сегментов:
{{ change_file.Content }}
