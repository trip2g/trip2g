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
- [Хаб философов](https://philosophers.2pub.me) — слой маршрутизации над 21 корпусом философов

## Текущая схема хаба

```mermaid
graph TD
  HUB["trip2g.com hub"]
  HUB --> J["nicksenin_journal"]
  HUB --> MA["markavrelii"]
  HUB --> PH["philosophers.2pub.me<br/>карточки · матрица тем · противоречия"]

  subgraph CORPORA["21 корпус философов"]
    direction LR
    C1["nietzsche · schopenhauer · goethe<br/>pascal · montaigne · larochefoucauld"]
    C2["confucius · laozi · epictetus<br/>tolstoy · ignatius · lebon · adler"]
    C3["machiavelli · franklin · smiles<br/>ford · rockefeller · hill<br/>wattles · james-allen"]
  end

  PH --> C1
  PH --> C2
  PH --> C3
```

Слепой federated-веер с этого хаба видит хаб философов как **одну** базу
(его карточки и матрицу тем). Глубина корпусов достигается адресно через
составной id: `federated_search(kb_id="philosophers/<slug>")`.
