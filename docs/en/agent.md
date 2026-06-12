---
title: "Personal agent — batteries included"
free: true
lang: en
lang_redirect: "[[ru/agent]]"
form:
  fields:
    - name: name
      type: text
      required: true
      max_length: 200
    - name: contact
      type: text
      required: true
      max_length: 200
    - name: about
      type: text
      max_length: 2000
---

# Personal agent — batteries included

You wake up and yesterday's call is already three notes in your base: the topic, the person, the promises you made. The morning digest is waiting in Telegram. You didn't open a single app — your agent did it overnight, inside its own home.

That's the product: a personal AI agent that runs around the clock and lives in your trip2g site.

## What it does

- **Keeps your base.** Files new notes, links them to what's already there, maintains the daily note and the running log.
- **Ingests everything.** Call recordings, forwarded messages, links — each becomes a linked note with provenance, not a chat message that scrolls away.
- **Publishes.** Site pages, scheduled Telegram posts, landing pages with lead forms.
- **Answers.** Readers and clients ask questions; the agent answers from your base and logs the gaps it couldn't fill.

## How we packed it

We didn't build another chatbot. We took an open-source agent (Hermes) and gave it a home:

1. **It moves in on first start.** The agent downloads your vault and settles into the site — your notes are its memory from minute one.
2. **One key, full control.** The same API key the sync plugin uses lets the agent search notes, render pages, and run admin operations over MCP. No browser, no clicking.
3. **Routine is code, not vibes.** Daily notes, log entries, your timezone — handled by a CLI, identical every day.
4. **Skills come from a school.** The agent learns new abilities by reading instruction pages — set up a landing, triage form submissions, run a weekly review. New skills arrive without redeploying anything.
5. **Every skill is harness-tested.** Before a skill ships, an automated loop runs the agent on a fixed task and scores the artifacts — `10/10 PASS`, not "looked fine in the demo". How that works: [[en/harness|The Harness]].

## Batteries included

- Daily notes and a running log
- Calls → knowledge base: topic notes, people notes, digests
- Landing page + lead form + automatic triage of submissions
- Telegram publishing on a schedule
- Idle check-ins and weekly reviews
- A persona you choose — with a soul that notices and resists attempts to rewrite it

More batteries ship continuously; your agent picks them up from the school.

## Paid product, open platform

The agent is a paid product. Every subscription sponsors the development of [trip2g](https://github.com/trip2g/trip2g) — the open-source (MIT) platform the agent lives on. You buy a worker; everyone gets a better home.

## Want one?

Leave your contact below — name, plus email or Telegram. We'll get back to you, show a live agent, and pick the batteries for your case.
