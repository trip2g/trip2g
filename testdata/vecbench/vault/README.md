---
title: Vecbench test vault
lang: en
free: true
---

This is the isolated corpus for the vector-search benchmark (`docker-compose.vecbench.yml`,
`scripts/vecbench.sh`). It is deliberately small, bilingual, and topic-disjoint so retrieval
relevance is unambiguous and reproducible.

Six topics, each written once in English (`en/`) and once in Russian (`ru/`):

| Topic | English | Russian |
|-------|---------|---------|
| Exoplanets | `en/exoplanets` | `ru/ekzoplanety` |
| Sourdough bread | `en/sourdough` | `ru/zakvaska` |
| Go goroutines | `en/goroutines` | `ru/goroutines` |
| Vector search | `en/vector-search` | `ru/vektornyy-poisk` |
| Photosynthesis | `en/photosynthesis` | `ru/fotosintez` |
| Green tea | `en/green-tea` | `ru/zelenyy-chay` |

Because every topic exists in both languages, the golden set can exercise all four
retrieval directions: ru→ru, en→en, ru→en (cross-lingual), en→ru (cross-lingual).
Each note sets `lang:` explicitly and lives in an `en/`/`ru/` folder, which also lets us
demonstrate setting `lang` per folder via frontmatter patches.

Extend this vault during golden-set construction (more notes / near-duplicate distractors
sharpen the benchmark), then re-run `./scripts/vecbench.sh sync`.
