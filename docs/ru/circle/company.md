---
title: "Распределённая база знаний компании"
label: "сеть баз компании"
order: 6
motion_visualization: federation
link: "[[ru/user/team-knowledge-base-mcp]]"
related:
  - "[[ru/user/hub]]"
  - "[[ru/user/federation]]"
  - "[[ru/thoughts/public-hub]]"
  - "[[ru/user/subgraph_for_agents]]"
layout: mesh/circle
lang: ru
free: true
noindex: true
lang_redirect: "[[en/circle/company]]"
---

Каждый ведёт свою базу и пишет только в неё. Чтобы дотянуться до знаний соседа, не нужно сваливать всё в одну общую вики или подключать агента к десяти базам: запрос маршрутизирует ваш собственный хаб. База соседа — одна заметка в вашем хабе с её адресом; federated_search отправляет вопрос туда или сразу во все базы, и каждый ответ помнит, из какой базы пришёл. Это data mesh для знаний, а круг друзей — то же самое, только меньше.
