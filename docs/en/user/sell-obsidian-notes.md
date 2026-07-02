---
title: "Sell access to your Obsidian notes"
description: "Put a paywall on part of your Obsidian vault and sell access from your own site. Subscriptions, paid Telegram groups, and crypto, with no per-sale platform cut."
free: true
home_position: 42
lang_redirect: "[[ru/user/sell-obsidian-notes]]"
---

Obsidian Publish lets you share your vault, but not sell it. Publish your vault with trip2g instead, mark some notes public and leave the rest gated, and sell access from your own domain. Readers who pay through Patreon, Boosty, a paid Telegram group, or crypto get the full text; everyone else sees a preview. You keep the vault, the site, the subscriber list, and every payment method's full amount minus that platform's own fee.

### Why Obsidian Publish can't sell your notes

Obsidian Publish has no login and no paywall. Every published page is public to anyone with the link. The workarounds people share on the forum are client-side JavaScript gates, which anyone can bypass by viewing source, so they do not actually protect paid content. The recurring forum question "does Obsidian Publish support paywalls?" has a plain answer: no.

The usual pivots are to package the notes into a PDF and sell it on Gumroad, or start a paid newsletter on Substack. Both work, and both take a cut of every sale and own the reader relationship. A one-time PDF also goes stale the day you sell it; the buyer got a frozen snapshot, and you got a single payment.

Selling access to a living knowledge base is different. The notes keep updating, the buyer keeps getting the current version, and you charge for ongoing access instead of a frozen file. trip2g runs that on a server you control, with a real login and a real paywall, so no marketplace sits between you and your subscribers.

### Four ways to monetize notes, compared

| Model | Typical platform | Platform cut | Owns the reader | Content stays live |
|---|---|---|---|---|
| One-time file sale | Gumroad, Lemon Squeezy | around 10% per sale | the platform | no, a frozen export |
| Membership community | Patreon, Memberful | 5 to 12% | the platform | posts, not a base |
| Paid newsletter | Substack | around 10% | the platform | issues, not a base |
| Self-hosted paywalled site | trip2g | 0% to trip2g (only the payment platform's own fee) | you | yes, a living vault |

For a single file sold to a cold audience, Gumroad is genuinely faster to launch, concede that. trip2g wins when you sell ongoing access to a base that keeps changing and you want to keep the whole relationship and the whole payment.

### How the paywall works

A note is private unless it has `free: true` in its frontmatter. So you publish two tiers from one vault:

- Notes with `free: true` are public and discoverable.
- Notes without it are visible only to admins and authenticated subscribers.

A visitor who is not signed in sees a preview and a sign-in prompt. A signed-in subscriber with active access sees the full note. Nothing else to configure per note.

Details and the exact access flow are in [[en/user/monetization|Monetization]].

### Ways to charge

trip2g does not run its own payment processor. It connects to platforms your audience may already use, plus direct crypto:

| Method | How access is granted | Best for |
|---|---|---|
| Patreon | Existing patrons sign in via Patreon; access follows their tier | Creators with a Patreon audience already |
| Boosty | Same as Patreon, via Boosty | Russian-speaking audiences |
| Paid Telegram group | The bot checks group membership | Channel owners monetizing a community |
| Crypto | Reader pays directly, then gets access | No third-party platform, no chargebacks |

You can combine them. A note can be unlocked by any active method, so a Patreon patron and a paid-group member both get in.

### Set it up

1. Get a running instance. Any of the [[en/user/hosting|hosting options]] works; self-hosting keeps the whole stack yours.
2. Publish your vault. Follow [[en/user/getting-started|Getting started]] to connect the sync plugin and push notes.
3. Decide your tiers. Add `free: true` to the notes you want to give away (an intro, a table of contents, your best public piece) and leave it off the paid ones.
4. Connect a payment method. In the admin panel, link Patreon, Boosty, a paid Telegram group, or enable crypto. See [[en/user/monetization|Monetization]].
5. Verify. Open a gated note in a private browser window. You should see the preview and a sign-in prompt, not the full text.

### Give subscribers an AI consultant too

Because the same notes are also an [[en/user/mcp|MCP endpoint]], you can sell more than reading. Subscribers can point their AI client at your knowledge base and ask it questions, and the answers stay grounded in your notes and cite them. You can gate a premium agent persona behind the same subscription using named MCP entry points. This is a product a static PDF cannot be.

### The hard part is not the tech, it is the audience

Standing up the paywall takes an afternoon. Getting people to pay is the real work. The pattern that works: give away a genuinely useful free tier as a lead magnet (mark your best public notes `free: true`), then convert readers who want the depth. Distribute where your topic already lives, an Obsidian community, a subreddit, a Telegram channel, before expecting search traffic to arrive.

The rough math helps set expectations. At $5 to $15 for one-time access, or $5 to $15 per month for a subscription, 100 subscribers at $5 is around $500 a month, and 200 at $15 is around $3,000 a month. Those are audience-size problems, not tooling problems, and no platform solves them for you.

### A note on protection

Any content a paying reader can see, a determined reader can copy. A PDF can be shared; a rendered page can be scraped. DRM mostly annoys honest customers. The durable posture is a fair price plus the convenience of always-current, well-organized access, which is exactly what a living vault offers and a leaked PDF does not. trip2g's paywall keeps out the casual freeloader, which is most of them; it is not, and nothing is, an anti-piracy vault.

### What you keep, and the honest tradeoffs

You keep the markdown files (export or `git clone` any time), the domain, the subscriber list, and every payment method's full cut minus that platform's own fee. There is no trip2g per-sale commission because there is no trip2g in the payment path.

The tradeoffs are real, so weigh them:

- **You run a server** (or rent a managed instance). Gumroad and Substack need nothing to run.
- **No built-in checkout.** Payment happens on Patreon, Boosty, or on-chain, not in a native cart. If you want one-click card checkout for strangers with no account, a marketplace is smoother.
- **Discovery is on you.** A marketplace brings some browse traffic. Your own site does not, until your SEO and audience do.

Pick trip2g when you are selling ongoing access to knowledge that keeps changing and you want to own the relationship. Pick a marketplace when you are selling a one-time file to a cold audience.

### Related

- [[en/user/monetization|Monetization]] — the full paywall reference: tiers, Patreon, Boosty, Telegram groups, crypto
- [[en/user/telegram|Telegram publishing]] — publish teasers to a channel, keep the full version paid
- [[en/user/mcp|MCP server]] — sell an AI consultant over your knowledge base
- [[en/user/getting-started|Getting started]] — connect Obsidian and publish your first notes
