---
description: "Krisp transcript → topic-boundary segmentation (step 1 of 2)"
model: qwen/qwen3-14b
tools: [write_note]
read_patterns: ["transcripts/**"]
write_patterns: ["segments/**"]
mode: change
trigger_on: [create, update]
trigger_include: ["transcripts/**"]
trigger_exclude: ["segments/**", "wiki/**"]
for_each: changed_files
max_depth: 3
concurrency: skip
max_steps: 5
max_tokens: 16000
---
Тебе дан транскрипт записи (созвон ИЛИ голосовая заметка одного человека), формат строк «Имя | MM:SS» или «Speaker N | MM:SS». Сырой ASR может быть рваным.

Файл-источник: `{{ change_file.Path }}`

Задача: **разметить смысловые границы** — разбить запись на сегменты по смене темы/мысли. Это карта для последующего оконного сканирования, НЕ полный пересказ. Под каждым сегментом — одна-две строки сути, не больше.

Правила:
- Заголовок сегмента = **вывод/суть куска (Minto), а не ярлык темы**. Плохо: «Обсуждение бюджета». Хорошо: «Бюджет режут — фокус на одном канале».
- Каждый сегмент помечай диапазоном времени `[MM:SS–MM:SS]` из транскрипта.
- Число сегментов ≈ длительность_мин / 5 (минимум 3, максимум 15).
- Границы ставь там, где реально меняется тема/направление мысли.
- `type` определи сам (sales-call, planning, brainstorm, interview, voice-note, lecture, support — или своё точное слово).
- `participants` — перечисли говорящих; для сольной заметки `-`.

Когда разметка готова, **запиши результат** инструментом `write_note`. Путь = `segments/` + basename исходного файла (то есть для `{{ change_file.Path }}` это `segments/<имя-без-папки>`). Строгий формат:

---
id: <basename без .md>
source_transcript: "{{ change_file.Path }}"
type: <классификация>
participants: <список или ->
kind: transcript-segments
---

## [MM:SS–MM:SS] Заголовок = суть сегмента
Одна-две строки сути.

## [MM:SS–MM:SS] Следующий сегмент …
…

Транскрипт:
{{ change_file.Content }}
