---
title: "Alerts belong in a knowledge base"
free: true
lang_redirect: "[[ru/thoughts/alerts-in-a-knowledge-base]]"
---

*What this is: we wrote [alert-sink](https://github.com/trip2g/alert-sink), a small open-source service that turns Alertmanager webhooks into incident notes in a trip2g instance. Every alert becomes a markdown note, the notes add up to a permanent searchable incident history, and the knowledge base can narrate incidents to a Telegram group on its own. Read it if your incident history currently lives in a chat scrollback.*

Alerts today are fire-and-forget. Prometheus notices a problem, Alertmanager pushes a message to Telegram or email, someone reacts or doesn't, and that's the end of the record. A month later you're asking "when did this node last flap?" and the honest answer is: scroll the chat and hope. Alertmanager itself only shows what is firing right now. History survives as a Prometheus time series, bounded by retention, or as a message that scrolled past.

That gap annoyed me for a while, because the fix is not a monitoring product. It's a place to write things down.

### One incident, one note

alert-sink is a single Go binary that sits behind Alertmanager as a webhook receiver. When an alert fires, the sink creates a markdown note in a trip2g knowledge base: `incidents/YYYY/MM/<timestamp>-<alertname>-<fingerprint>.md`, with the full label set and timing in frontmatter. When the alert resolves, the same note flips to resolved.

The path is a pure function of the alert identity. Re-deliveries and retries land on the same note, so an alert that fires, gets re-notified five times and resolves is still exactly one note. Not six messages in a chat.

What that buys you:

- **Permanent history.** Notes don't expire with Prometheus retention and don't scroll away.
- **Search.** Incident history is just notes in a knowledge base, so full-text search over it comes for free.
- **Postmortems that link back.** Every incident note carries a wikilink to a sibling postmortem note. A human or an agent writes the postmortem later, and the sink is careful never to touch that path.
- **Narration.** trip2g already knows how to publish notes to Telegram chats by tag. The sink stamps every incident note with the publish fields, so a wired-up knowledge base can post the incident to a group and edit the same message in place when it resolves. One incident, one message, flipping from firing to resolved on its own.

That last point is the part I find interesting. The knowledge base isn't just an archive that happens to hold incidents. It's the thing doing the talking. The sink writes a note; everything downstream, the magazine page, the search, the Telegram post, is the knowledge base behaving the way it behaves with any other note.

### Try it

The repo ships a self-contained demo: Prometheus, Alertmanager, alert-sink and a trip2g instance in one compose file.

```bash
git clone https://github.com/trip2g/alert-sink
cd alert-sink/demo
docker compose up --build
```

Wait a minute, then open http://localhost:8080/incidents. That's the incident magazine, a trip2g page listing every incident newest first. A demo alert is already there. To watch a full lifecycle:

```bash
docker compose stop demo-target    # DemoTargetDown fires after ~1 minute
docker compose start demo-target   # the same note flips to resolved
```

### A few engineering notes

Three decisions I'd defend.

**Resolve is a patch, not a rewrite.** When an alert resolves, the sink edits exactly two adjacent frontmatter lines in the note, `status` and `ends_at`, with a single find and replace. It never regenerates the note body. So if someone added context to the note while the incident was live ("this is the flaky switch again, see last week"), that text survives the resolution. The machine and the humans share a document without clobbering each other.

**Self-mint auth.** trip2g has no long-lived scoped write tokens, so the sink holds the instance's JWT secret and signs itself a fresh 5-minute token for every write. No credential longer-lived than 5 minutes ever exists outside the secret. The honest caveat: that token is admin on its instance, trip2g has no path-level scoping. Least privilege comes from blast radius, so point the sink at a dedicated instance that holds only alert history.

**Alertmanager never waits.** The webhook handler validates, enqueues, and answers 200 in milliseconds. A worker goroutine does the actual writes with exponential backoff, so a trip2g outage doesn't back-pressure your alerting pipeline. Which matters, because the moments when trip2g might be down are exactly the moments alerts are firing.

The whole thing is Go stdlib plus one JWT library. No database, no state outside the knowledge base itself. MIT licensed: https://github.com/trip2g/alert-sink
