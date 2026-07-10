---
title: "Хаб"
free: true
lang: ru
lang_redirect: "[[en/hub/_index]]"
---

# Хаб

Базы знаний, подключённые к этому хабу через федерацию.

- [[ru/hub/nicksenin_journal|Журнал Ника Сенина]]
- [[ru/hub/markavrelii|Марк Аврелий — Размышления]]
- [[ru/hub/minionschool|Школа миньонов]]
- [[ru/hub/philosophers|Хаб философов]] — слой маршрутизации над 21 корпусом философов

## Текущая схема хаба

Корневой хаб подключает `philosophers` одним пиром; тот хаб уже федерирует
21 корпус. Пунктирные линии — связи влияния между корпусами: корпус указывает
на соседей, с которыми спорит, и агент может пройти по идее через базы.

```mermaid
graph TD
  HUB["trip2g.com hub"]
  HUB --> J["nicksenin_journal"]
  HUB --> MA["markavrelii"]
  HUB --> MS["minionschool"]
  HUB --> PH["philosophers"]

  PH --> nietzsche
  PH --> schopenhauer
  PH --> goethe
  PH --> pascal
  PH --> montaigne
  PH --> larochefoucauld
  PH --> confucius
  PH --> laozi
  PH --> epictetus
  PH --> tolstoy
  PH --> ignatius
  PH --> lebon
  PH --> adler
  PH --> machiavelli
  PH --> franklin
  PH --> smiles
  PH --> ford
  PH --> rockefeller
  PH --> hill
  PH --> wattles
  PH --> jamesallen["james-allen"]

  nietzsche -.->|отвечает| schopenhauer
  pascal -.->|против| montaigne
  epictetus -.->|Стоя| MA
  confucius -.->|порядок vs путь| laozi
  tolstoy -.->|опирается| epictetus
  tolstoy -.->|опирается| MA
```

Слепой веер с корневого хаба видит `philosophers` как **одну** базу. Чтобы дойти
до конкретного мыслителя, спрашивают хаб философов по `kb_id` его корпуса
(например, `federated_search(kb_id="nietzsche")` к `philosophers.2pub.me`).

Что такое федеративный поиск и как он работает — [[ru/user/federation|MCP Federation]].
