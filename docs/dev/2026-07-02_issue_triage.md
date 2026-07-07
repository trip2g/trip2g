# Open issue triage — 2026-07-02

Read-only triage of all open issues in `trip2g/trip2g`. Judged against the current direction: Markdown OS positioning (README), fleet agent runtime + MCP memory, and this week's SEO/distribution push (`2026-07-02_seo_growth_plan.md`, `2026-07-02_launch_kit.md`). No issue was closed or edited — recommendations only.

## TL;DR

6 open issues. **Implement 1** (#6 InstantView — small remaining scope, feeds the SEO/Telegram distribution push). **Backlog 2** (#5 calendar/recurring, #26 OKF export). **Close 3** (#9, #11, #12 — creator-economy monetization and typography polish from March, misaligned with the fleet/corp direction, zero demand since filing).

---

## Do-now shortlist (1 issue)

### #6 — Telegram InstantView Support

**Why now.** The only issue that intersects the current push. The SEO growth plan and launch kit are about distribution; Telegram is trip2g's flagship output (the `telegram-blog-from-obsidian` SEO page shipped this week). Links shared in Telegram opening instantly inside the app is a direct conversion win for exactly that use case — and it's dogfood: the founder's own channel posts trip2g links daily.

**Why it's small.** Your own 2026-06-23 status comment already did the hard triage: OG tags, `og:image` resolution, Twitter Card, and JSON-LD all shipped. What remains:

1. `telegram:channel` meta tag + config (env or admin setting) — hours.
2. Semantic article structure — `views.html:286` renders a plain `article.content` block; InstantView's parser wants `<header>`, `<time pubdate>`, `rel=author`, `<figure>`. This also helps SEO parsers generally, so it double-counts toward the SEO plan's "structurally sound pages" goal.
3. Register the rhash template with Telegram — manual external step, ~1–2 hours, human-in-the-loop.

**Effort:** ~1–2 days total. **Risk:** the rhash registration is per-domain and Telegram-side; scope it to the canonical docs domain first, don't build multi-domain template management.

Nothing else makes the cut. The fleet pipeline and the SEO batch are the week's real work; every other issue below would be scatter.

---

## Close list (3 issues)

Ready for a `gh issue close -c "<comment>"` batch after approval:

| # | Title | Close reason | Suggested closing comment |
|---|-------|--------------|---------------------------|
| #11 | Referral System and Promo Codes | Misaligned + stale — creator-economy payments feature from March, no demand since (only comment is a drive-by /apply), large payments-surface scope while the direction is agent runtime / corp | "Closing: no demand since March and current focus is the agent runtime, not creator monetization. Will reopen if a paying creator asks for promo codes." |
| #9 | Timed Content Unlock | Misaligned + stale — same monetization lane as #11; the implementation plan references files that don't exist in the codebase (`internal/access/`, `internal/jobs/`), so it would need a full re-spec anyway | "Closing: no demand since March; the spec is stale against the current codebase. Will reopen with a fresh design if drip/scheduled unlock is requested." |
| #12 | Typography Processor for CIS Languages | Wontfix-for-now — self-labeled priority:low polish; render-time typograph is a nice-to-have that can return as a goldmark post-processor or template processor when RU publishing demand justifies it | "Closing as low-priority polish. If RU typography becomes a real ask, this comes back as a goldmark post-processor — the spec here stays as reference." |

Note: #9 and #11 are the two remaining March-batch AI-generated mega-specs (5-question "Questions for creator" format, invented file paths). Closing them also cleans that generation artifact out of the tracker.

---

## Backlog (2 issues)

- **#5 — Telegram scheduled posts (calendar).** Keep open, don't touch now. The June status comment already trimmed it honestly: one-time `telegram_publish_at` scheduling **shipped** — that was the 80%. What's left is recurring schedules (modest backend) and a drag-and-drop calendar UI (a large frontend build; the kanban board showed how expensive that class of UI is). If demand appears, split: do recurring `post_schedule` backend alone first; the calendar stays parked. It's still the strongest backlog item — priority:high label is fair for the recurring half.
- **#26 — OKF export (wikilinks → markdown links).** Self-filed a week ago as an explicit "deferred TODO so it isn't lost" — keep it exactly that. It aligns with the agent-memory/OKF/fleet-KB direction, but the issue itself says the placement is undecided (memcli vs server-side vs лк), and the fresh `2026-07-02_wikilink_multilang_resolution.md` design should settle where a link-rewrite pass lives before any code. Revisit after that design lands.

---

## Focus verdict

Say yes to one thing: finish #6, because it's a day or two of work that compounds with the SEO/launch push already in flight and improves every Telegram link trip2g ever emits. Consciously drop the March monetization pair (#9, #11) and the typography polish (#12) — they belong to the creator-economy framing trip2g had before the fleet/Markdown-OS pivot, and keeping them open invites scope creep back into a lane with no current customers. #5's remaining scope (calendar UI) and #26 (OKF export) are real but should wait for a demand signal and the wikilink design respectively; the tracker stays honest with 3 open issues instead of 6.
