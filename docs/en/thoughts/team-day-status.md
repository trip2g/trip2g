---
title: The manager reads the team's day from their agents
free: true
lang_redirect: "[[ru/thoughts/team-day-status]]"
---

Every employee works with their own hermes agent, and each agent has its own trip2g. In the morning the agent asks its person what they plan, and in the evening how the day went. It keeps the answer as a note. The manager's agent copies these notes from every employee into its own base and builds one page of the team's day.

There is no stand-up call and no form to fill. Nobody sends anybody a report.

![[team-day-dashboard.png|The manager's page: one row per employee, assembled from their agents' notes]]

## Two questions a day

The questions cost nothing. A hermes cron job with `no_agent` runs a script that prints a fixed text, and the job delivers that text to the person's Telegram. No model runs, so no tokens are spent:

```
09:00  daily-morning  → «Что сегодня в планах?»
18:00  daily-evening  → «Как прошёл день? Что сделано, где нужна помощь?»
```

One flag matters: `attach_to_session`. Without it the question lives only in the cron job's own session, and the person's answer reaches the agent with no question in front of it. With it, the question is in the chat history, and the answer is understood as an answer.

## The note of the day

The agent turns each answer into a few short lines of its own. It never keeps the chat itself:

```markdown
# 2026-10-18 — Маша

## План
- Дописать коммерческое предложение для «Альфа-Логистик».
- В 15:00 провести созвон с «Альфа-Логистик».

## Итог
- Коммерческое предложение отправлено.
- Созвон проведён; клиент запросил скидку 10%.

## Нужна помощь
- Требуется согласование руководителя на скидку 10%.
```

Morning fills «План». Evening fills «Итог» and «Нужна помощь». One file per day: `daily-sync/YYYY-MM-DD.md`.

![[team-day-note.png|One employee's day, as their agent keeps it]]

## Opened to the manager, and only that

A frontmatter patch puts the folder into a subgraph of its own:

```jsonnet
{ subgraph: "daily_sync" }
```

The employee's agent then mints a federation key with `federation_share`, scoped to `daily_sync` alone, and hands it to the manager's agent, which connects it once. The manager sees the days and nothing else of the employee's base: no notes, no memory, no other subgraph. Revoking that one key closes it.

## A fan-out, then a copy

`federated_note_html` reads one note from one base, so thirty employees take thirty calls. The path is fixed by convention, which means nothing has to be searched:

```
for each connected base:
    federated_note_html(kb_id, "daily-sync/<today>.md")
    → write team/<person>/<today>.md if it changed
```

A script does this hourly, from another `no_agent` cron job. Search would not serve here: `federated_search` without an explicit `kb_ids` list asks at most seven bases (`--federated-fanout-limit`), five at a time, five seconds each, and skips the rest.

The manager keeps a copy rather than reading live. That is deliberate. A day's status is a dated fact that nobody edits afterwards, so the copy cannot go stale. If an employee's base is offline in the evening, the page still shows that employee's day as last copied. And the manager's model reads local notes, not thirty peers.

## The page

One more cron job, this one with the model, reads today's copies and writes `dashboards/team-today.md`: a row per employee, a list of who needs help, and who has no status today. An empty day is information too, so the agent does not chase anyone.

## What the first run taught

- **The question has to be in the history.** Without `attach_to_session` the answer «дописываю КП» reaches the agent as a message out of nowhere.
- **Say which skill takes the key.** A message carrying a handover block matches a generic "connect a knowledge base" skill first. The manager's agent needs a standing instruction that this kind of message belongs to the team status.
- **The name comes from the person.** The agent takes the name from the Telegram profile until told otherwise.
- **Privacy is part of the design.** The person knows the manager reads their day, and the note holds a summary, never the conversation.

## Related

- [[en/thoughts/agent-team-digest|The agent keeps a team journal]]
- [[en/thoughts/agent-team-reporting|The agent reports to the team on its own]]
- [[en/thoughts/federation-of-gardens|A federation of gardens]]
