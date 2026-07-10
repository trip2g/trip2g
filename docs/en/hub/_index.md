---
title: "Hub"
free: true
lang: en
lang_redirect: "[[ru/hub/_index]]"
---

# Hub

Knowledge bases federated into this hub.

- [[en/hub/nicksenin_journal|Nick Senin Journal]]
- [[en/hub/markavrelii|Marcus Aurelius — Meditations]]
- [Philosophers Hub](https://philosophers.2pub.me) — routing layer over 21 philosopher corpora

## Current hub topology

```mermaid
graph TD
  HUB["trip2g.com hub"]
  HUB --> J["nicksenin_journal"]
  HUB --> MA["markavrelii"]
  HUB --> PH["philosophers.2pub.me<br/>cards · topic matrix · contradictions"]

  subgraph CORPORA["21 philosopher corpora"]
    direction LR
    C1["nietzsche · schopenhauer · goethe<br/>pascal · montaigne · larochefoucauld"]
    C2["confucius · laozi · epictetus<br/>tolstoy · ignatius · lebon · adler"]
    C3["machiavelli · franklin · smiles<br/>ford · rockefeller · hill<br/>wattles · james-allen"]
  end

  PH --> C1
  PH --> C2
  PH --> C3
```

A blind federated fan-out from this hub sees the philosophers hub as **one**
base (its cards and topic matrix). Corpus depth is reached deliberately via a
composite id: `federated_search(kb_id="philosophers/<slug>")`.

What federated search is and how it works — [[en/user/federation|MCP Federation]].
