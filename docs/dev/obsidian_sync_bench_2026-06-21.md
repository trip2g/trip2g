# Obsidian Sync — benchmark (2026-06-21)

Замеры времени синка на ~2000 заметках через obsidian-sync CLI (тот же
`classify`/`execute`, что и в плагине), стек `docker-compose.syncperf.yml`
без векторного поиска. Каждый прогон `bench.mjs` дописывает секцию ниже —
так baseline и «после фикса» сравниваются в одном файле.

Сценарии: **cold** (первый пуш всех заметок), **noop** (повтор без изменений —
idle, где мёртвый кэш пере-хэширует всё), **dry-noop** (только classify),
**small-change** (правка K заметок), **twoway-noop** (`--two-way` без изменений).

---
## Run: baseline (2026-06-21 09:13:32)

- commit `14b57753`, notes 2002, repeats 3, touch 5, node v24.15.0
- stack: docker-compose.syncperf.yml (vector search OFF), CLI via tsx
- decomposition: noop − dry-noop ≈ execute/asset overhead = **0.04s**

| scenario | median | min | max | runs | counts |
|---|---|---|---|---|---|
| cold | 3.30s | 3.30s | 3.30s | 1 | push=0, pushed=2000, assetsUp=0, err=0 |
| noop | 0.36s | 0.36s | 0.36s | 3 | push=0, pushed=0, assetsUp=0, err=0 |
| dry-noop | 0.32s | 0.32s | 0.35s | 3 | push=0, pushed=0, assetsUp=0, err=0 |
| small-change | 0.44s | 0.43s | 0.45s | 3 | push=5, pushed=5, assetsUp=0, err=0 |
| twoway-noop | 0.37s | 0.36s | 0.39s | 3 | push=0, pushed=0, assetsUp=0, err=0 |
## Run: assets-unique (2026-06-21 09:28:26)

- commit `14b57753`, notes 2002, repeats 3, touch 5, node v24.15.0
- stack: docker-compose.syncperf.yml (vector search OFF), CLI via tsx
- decomposition: noop − dry-noop ≈ execute/asset overhead = **0.11s**

| scenario | median | min | max | runs | counts |
|---|---|---|---|---|---|
| cold | 231.79s | 231.79s | 231.79s | 1 | push=0, pushed=2000, assetsUp=2000, err=0 |
| noop | 0.67s | 0.66s | 1.01s | 3 | push=0, pushed=0, assetsUp=0, err=0 |
| dry-noop | 0.55s | 0.54s | 0.57s | 3 | push=0, pushed=0, assetsUp=0, err=0 |
| small-change | 1.05s | 0.93s | 1.12s | 3 | push=5, pushed=5, assetsUp=0, err=0 |
| twoway-noop | 0.73s | 0.72s | 0.76s | 3 | push=0, pushed=0, assetsUp=0, err=0 |

### Findings (baseline)

**Главное: ассеты — это вся боль cold-sync.** Cold push 2000 заметок: без ассетов
**3.3s**, с ассетами (2000 × 1×1 PNG по 69 байт) — **231.8s**. Рост ×70 при том,
что суммарный объём картинок ~138 КБ. Цена не в байтах — **~116ms на ассет** чистого
оверхеда.

**Корень: CLI/browser не шлют `skipCommit:true` при загрузке ассета.** Сервер в
`uploadnoteasset/resolve.go:108` делает `PrepareLatestNotes` (полный reload всех
заметок) на КАЖДУЮ загрузку, если не передан `skipCommit`. → 2000 reload-ов.

| фронтенд | `skipCommit` при upload | cold-push ассетов |
|---|---|---|
| **plugin** (`src/env.ts:308`) | **есть** ✓ | не страдает (reload один раз в commit) |
| **CLI** (`src/sync/cli/env.ts:404-411`) | нет ✗ | ×N reload → 231s |
| **browser** (`src/sync/browser/index.ts:574-581`) | нет ✗ | ×N reload |

**Важный вывод:** baseline через CLI **переоценивает** cold-push живого плагина по
ассетам — у плагина `skipCommit` уже есть. Чтобы CLI-бенч отражал плагин (и чтобы
почистить реальный баг CLI/docs-sync), нужно добавить `skipCommit:true` в
`cli/env.ts` и `browser/index.ts`. Ожидание после фикса: cold ≈ 3–10s.

**Второй баг (дедуп ассетов):** дедуп по `(noteId, localPath)` — если N заметок
ссылаются на один файл, `uploadAsset` зовётся N раз для одного файла (N round-trip
+ N локальных пере-хэшей; сервер контент-адресный → блоб пишется один раз, но
вызовы идут). Замеряется режимом `--asset-mode shared` (виден после фикса
`skipCommit`, иначе маскируется reload-ом).

**idle (noop/dry/small) — дёшево даже с ассетами** (0.5–1.1s, из них ~0.34s —
стартап tsx на запуск; в плагине его нет). Мёртвый mtime/hash-кэш на этом размере
заметок не виден; будет заметен на крупных заметках или по сети с задержкой.
## Run: assets-unique-skipcommit (2026-06-21 09:37:30)

- commit `14b57753`, notes 2002, repeats 3, touch 5, node v24.15.0
- stack: docker-compose.syncperf.yml (vector search OFF), CLI via tsx
- decomposition: noop − dry-noop ≈ execute/asset overhead = **0.07s**

| scenario | median | min | max | runs | counts |
|---|---|---|---|---|---|
| cold | 8.80s | 8.80s | 8.80s | 1 | push=0, pushed=2000, assetsUp=2000, err=0 |
| noop | 0.41s | 0.40s | 0.42s | 3 | push=0, pushed=0, assetsUp=0, err=0 |
| dry-noop | 0.34s | 0.33s | 0.34s | 3 | push=0, pushed=0, assetsUp=0, err=0 |
| small-change | 0.52s | 0.51s | 0.52s | 3 | push=5, pushed=5, assetsUp=0, err=0 |
| twoway-noop | 0.39s | 0.39s | 0.40s | 3 | push=0, pushed=0, assetsUp=0, err=0 |

### Подтверждение фикса skipCommit

`cli/env.ts` + `browser/index.ts`: добавлен `skipCommit: true` в `uploadNoteAsset`
(как уже было в плагине `env.ts:308`). **cold: 231.79s → 8.80s (×26).** Гипотеза
«per-upload `PrepareLatestNotes`» подтверждена эмпирически. Прогноз 3–10s сбылся.
Остаток 8.8s = 20 push-батчей (по 100, reload каждый) + 2000 upload round-trip
(теперь дёшево) + один финальный commit. Следующая цель cold — инкрементальный
reload на push (`pushnotes/resolve.go:83`).

Замечание: это **косвенно** подтверждает, что живой плагин в ~8.8s-режиме (у него
`skipCommit` есть), но прямой замер плагина ещё нужен (см. план).
## Run: assets-shared-skipcommit (2026-06-21 09:39:11)

- commit `14b57753`, notes 2002, repeats 3, touch 5, node v24.15.0
- stack: docker-compose.syncperf.yml (vector search OFF), CLI via tsx
- decomposition: noop − dry-noop ≈ execute/asset overhead = **0.05s**

| scenario | median | min | max | runs | counts |
|---|---|---|---|---|---|
| cold | 5.97s | 5.97s | 5.97s | 1 | push=0, pushed=2000, assetsUp=2000, err=0 |
| noop | 0.38s | 0.36s | 0.38s | 3 | push=0, pushed=0, assetsUp=0, err=0 |
| dry-noop | 0.33s | 0.33s | 0.34s | 3 | push=0, pushed=0, assetsUp=0, err=0 |
| small-change | 0.52s | 0.51s | 0.54s | 3 | push=5, pushed=5, assetsUp=0, err=0 |
| twoway-noop | 0.38s | 0.38s | 0.39s | 3 | push=0, pushed=0, assetsUp=0, err=0 |

shared cold **5.97s** < unique cold **8.80s**: сервер контент-адресный, 1999 minio-записей
пропущены. Но `assetsUp=2000` — дедуп-баг: один файл загружается 2000 раз (2000
round-trip + 2000 локальных пере-хэшей одного файла). Чистая трата ≈ 5.97 − 3.3 ≈ **2.7s**.
Фикс дедупа (грузить уникальный блоб один раз) свёл бы shared cold к ~3.3s.

### Прямой замер живого плагина (Obsidian)

Открыт `tmp/syncperf-vault` в Obsidian, сервер пуст, добавлена команда `trip2g:sync`,
триггер через `~/.local/bin/obsidian command id=trip2g:sync`, длительность —
по числу `notePaths` (0 → 2000 на финальном commit), механизм — по docker-логам.

**cold push 2000 заметок + 2000 ассетов: 6.0s.** Из логов сервера за окно синка:
- `pushNotes: insert note` — 2000 (все заметки),
- полных reload (`latest noteloader: notes indexed`) — **20** (= 2000/100 push-батчей),
- reload на загрузку ассета — **0** (плагин шлёт `skipCommit:true`).

| фронтенд | cold (2000 + assets) | reload-ов | механизм |
|---|---|---|---|
| CLI (баг) | 231.79s | ~2020 | `PrepareLatestNotes` на каждый upload |
| CLI (фикс `skipCommit`) | 8.80s | ~20 | skip per-upload reload |
| **plugin (живой)** | **6.0s** | **20** | `skipCommit` уже был (`env.ts:308`) |

**Вывод (эмпирический, не инференс):** живой плагин per-asset-reload багом **не страдает**.
Оставшаяся цена cold (20 reload на push-батчи) — общая для всех; цель — инкрементальный
reload (`pushnotes/resolve.go:83`). Также в логах: `write pool exhausted ... will block` ×2
(контеншн на write-пуле под всплеском — отдельный мелкий сигнал).

### Warm in-process (без стартапа tsx, 15 итераций)

`scripts/syncperf/warmbench.mts` — времена самого `classify` без ~0.34s оверхеда node/tsx
на запуск (плагин его не платит — долгоживущий процесс):

| операция (2000 заметок) | median | min | max |
|---|---|---|---|
| classify (idle, full) | **38.0ms** | 32.0 | 67.9 |
| local hash all 2000 («мёртвый кэш») | **17.9ms** | 15.6 | 20.8 |
| fetch server hashes | 10.5ms | 9.6 | 14.8 |

**Вывод:** на 2000 мелких заметок (localhost) idle-классификация ~38ms, пере-хэш всех
файлов ~18ms. Мёртвый mtime/hash-кэш здесь **незначим**. Он станет заметен на крупных
заметках (стоимость ∝ байтам) или по сети с задержкой (10ms × RTT-фактор). Числа из
CLI-бенча (noop ~0.4s) — это почти целиком стартап tsx, а не работа синка.
