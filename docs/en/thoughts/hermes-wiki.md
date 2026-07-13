---
title: "Hermes Wiki: a site-agent on any server"
free: true
lang: en
lang_redirect: "[[ru/thoughts/hermes-wiki]]"
---

# Hermes Wiki: a site-agent on any server

For a while I've wanted to build the smallest possible version of what we
do. Not the full box with Nomad, Vault and network isolation — the bare
minimum: two containers on any five-dollar server.

The first container is trip2g. It keeps your notes in a single SQLite file
and serves them to the world as a website: a plain note becomes a page, and
if you drop an HTML template next to it, it becomes any page you want,
landing pages included. Open source, one process, no database server to
run.

The second is [hermes](https://github.com/NousResearch/hermes-agent), an
open LLM agent from NousResearch. Its brain is your own Codex subscription:
you put your auth.json on your server, and the agent thinks on the plan you
already pay for. No third-party API keys, no middleman between you and the
model.

Between them: one volume and one token. Hermes keeps notes in a folder, a
sync tool pushes them to trip2g, trip2g shows them to the world. That's the
whole architecture.

What it feels like in practice. You message the agent on Telegram: "note
this — the call with Igor moved to Thursday." It appends a line to today's
note. A week later you ask "what was the deal with Igor?" and it finds the
answer. Then you say "make a page for my course, here's a picture of the
design I want" — and a minute later you get a link to a finished landing
page on your own domain. Notes and website live in the same place; backing
it up means copying one file.

Here's the part that matters: you can just tell your site what to do now. You
drop a line into Telegram — "make me a page like this" — and that's it:
hermes reads it and lays it out, trip2g puts it on your domain. They split
the work between themselves.

![[tg_publish.webp]]
*One line to the bot in Telegram, and a page appears on your domain.*

And you don't have to type. Send the agent a voice message — "make me a page
about a space café, dark theme" — and it transcribes the clip on the server
itself (a local Whisper, nothing leaves the box) and builds the page from
what you said. Talking your website into existence turns out to be the most
natural interface it has: you speak a sentence into Telegram on the way
somewhere, and a page shows up on your domain.

![[hermes-wiki-kosmicheskoe-kafe.webp]]
*The finished page on your own domain.*

I call it a site-agent. Not an assistant in someone else's cloud that
changes its pricing tomorrow, but a tenant on your machine. Its memory is a
SQLite file you can copy anywhere. Its pages are your domain. Its brain is
your subscription. The same argument I made in
[[en/thoughts/digital-sovereignty|digital sovereignty]] for notes, extended
to the agent that keeps them.

To be honest about the limits: this is not a fortress. In the full box the
agent never sees the Telegram token or the Codex credentials — there are
proxies and a broker between it and every secret, and that is a separate,
much bigger piece of work. In the two-container version the auth.json sits
right next to the agent, protected only by the fact that it is your server
and your agent. For a personal wiki and a website that's a fair trade. For
a company running a dozen agents it isn't — that's what the box is for.

Everything needed to set this up lives in one public repo:
[trip2g/hermes_wiki](https://github.com/trip2g/hermes_wiki). One
`curl | sh` line from its README brings up both containers. There's also a
version I like even more: the same recipe written as an instruction for an
agent, not for a human. You paste it into your own Claude Code or Cursor,
and it installs Docker, brings up the containers, converts the auth.json,
runs the acceptance checks and hands you the link to your first page. An
installer that ships as text and is executed by your own AI — I like how
that sounds in 2026.

Tested on a bare VM: from empty Ubuntu to a working site-agent in minutes,
most of which is Docker pulling images.
