---
title: "Email delivery (SMTP)"
free: true
home_position: 62
lang_redirect: "[[ru/user/smtp]]"
---

To sign in, trip2g emails you a one-time code. No working email = no code = you can't get in. So a self-hosted instance needs a way to send mail. trip2g speaks plain SMTP, so any transactional-email service works — pick one below, drop its SMTP credentials into your config, done.

Two ways to skip email entirely, before you commit to a provider:

- **Sign in with Google or GitHub (OAuth).** No email, no SMTP, no provider account. If OAuth login covers everyone who needs access, you don't need any of this page.
- **Don't run your own mail server.** It's the worst option — details below.

See [[selfhosted]] for the full self-hosting guide; this page is only the email piece.

### Why not run your own mail server?

Theoretically possible, in practice the worst option. Don't waste a day on it. Field test on a fresh Hetzner VPS:

- **Outbound port 25 is blocked by default.** `nc gmail-smtp-in.l.google.com 25` just times out, while `nc <relay-host> 587` connects instantly. Most hosts (Hetzner, and the big clouds) block `25` outbound to fight spam.
- **A local MTA doesn't help.** Install Postfix and your mail sits `deferred` forever — it can never reach port `25` outbound, so it is never delivered. Not to the inbox, not even to spam. It just queues.
- **Getting `25` unblocked only starts the real fight.** Even with an open port you then have to win rDNS/PTR, SPF, DKIM, DMARC, and build IP reputation from zero on a cold IP. Weeks of work, and one misstep lands you on a blocklist.

Conclusion: send through a hosted relay on port `587`. Don't run your own outbound MTA. The rest of this page is how to pick that relay.

### Which provider?

All of these give you an SMTP username and password you paste straight into trip2g. Sorted by usefulness for a small self-host: generous forever-free tiers with plain SMTP first, trial-only and pay-as-you-go last.

The biggest free tier isn't automatically the best sender — deliverability and sender reputation vary. The **Reputation** column flags how each is *commonly regarded* for transactional mail; it's a rough signal, not a ranking, and your own domain setup (verified sender, SPF, DKIM) matters just as much.

Figures are current as of mid-2026. Free tiers change often — **confirm on the provider's own pricing page before you rely on a number.**

| Service | Free tier | SMTP? | API? | Reputation* | Catch |
|---|---|---|---|---|---|
| **Brevo** (ex-Sendinblue) | 300/day, forever (~9,000/mo) | Yes | Yes | Good | Most generous forever-free SMTP. 300/day hard cap. |
| **Mailjet** | 6,000/mo, forever | Yes | Yes | Good | 200/day cap; over-cap mail queues up to 3 days then is dropped. |
| **Resend** | 3,000/mo, forever | Yes | Yes | Strong | 100/day cap. Modern, developer-focused. |
| **Elastic Email** | 3,000/mo (100/day), forever | Yes | Yes | Mixed | Cheap, but shared-pool reputation is uneven; free mail carries an Elastic Email footer. |
| **SMTP2GO** | 1,000/mo (200/day), forever | Yes | Yes | Good | Low monthly cap; fine for a single-owner instance. |
| **Mailgun** | 100/day, forever | Yes | Yes | Strong | 1 sending domain, 1-day log retention; a card may be required at signup — verify. |
| **Scaleway TEM** | 300/mo, forever | Yes | Yes | Good | Then €0.25/1,000. EU provider; needs a Scaleway account + billing set up. |
| **Amazon SES** | 3,000 msg/mo — first 12 months only | Yes | Yes | Strong | Not forever-free (old 62k EC2 tier is gone). Then ~$0.10/1,000 — cheapest at volume. New accounts start in a sandbox; request production access. |
| **Zoho ZeptoMail** | 10,000-email credit, ~1 month trial | Yes | Yes | Good | Trial credit, not recurring. Then ~$2.50/10,000 pay-as-you-go — very cheap. |
| **Postmark** | 100/**month** — testing only | Yes (paid) | Yes | Excellent | Widely regarded as the deliverability benchmark for transactional mail. Real sending needs a paid plan (~$15/mo). |
| **SendGrid** (Twilio) | 60-day trial, 100/day | Yes | Yes | Strong | No longer has a permanent free tier — it became a 60-day trial in 2025. Paid from ~$19.95/mo. |
| **MailerSend** | 500/mo, forever | Yes | Yes | Good | Allowance was cut from 3,000 to 500/mo in Dec 2025 — check it hasn't moved again. |

\* As commonly recommended for transactional mail, not an invented ranking. Deliverability also depends on your own domain (verified sender, SPF, DKIM).

### Plug it into trip2g

trip2g sends mail over plain SMTP. The relevant settings (env var / CLI flag):

| Env var | Flag | Default | Meaning |
|---|---|---|---|
| `SMTP_HOST` | `-smtp-host` | *(empty = log only)* | Provider's SMTP server host. Empty means no mail is sent. |
| `SMTP_PORT` | `-smtp-port` | `587` | SMTP port. `587` for STARTTLS is the usual choice. |
| `SMTP_USER` | `-smtp-user` | *(empty)* | SMTP username from the provider. |
| `SMTP_PASS` | `-smtp-pass` | *(empty)* | SMTP password / API key from the provider. |
| `SMTP_STARTTLS` | `-smtp-starttls` | `true` | Use STARTTLS. Leave `true` for port `587`. |
| `MAIL_FROM` | `-mail-from` | `no-reply@example.com` | Sender address. **Must** be on a domain you verified with the provider, or mail is rejected. |

Example `/etc/trip2g.env` block, using Brevo's SMTP relay:

```dotenv
SMTP_HOST=smtp-relay.brevo.com
SMTP_PORT=587
SMTP_USER=your-login@example.com
SMTP_PASS=<brevo SMTP key>
SMTP_STARTTLS=true
MAIL_FROM=no-reply@your-verified-domain.com
```

Every provider gives you the equivalent three values — host, user, pass. For Resend it's `smtp.resend.com`, user `resend`, pass = your API key. For Mailjet, `in-v3.mailjet.com` with an API key/secret pair. Check the provider's SMTP setup page for the exact host.

`MAIL_FROM` is the common gotcha: it must be an address on a domain you've added and verified (its DNS records) with the provider. An unverified sender domain gets rejected — so mail effectively only works once verification is green.

**No email yet?** trip2g runs fine without SMTP — it just skips sign-in emails. To bootstrap the very first login, set `LOG_SIGN_IN_CODES=true`, restart, trigger a sign-in, and read the code from the server log (`journalctl -u trip2g`). Unset it once you're in — it prints codes in plaintext.

### Which should I pick?

- **Easiest free start:** **Brevo** (300/day forever, plain SMTP), **Resend**, or **Mailjet**. All free and enough for a personal or small-team instance.
- **Best deliverability reputation:** **Postmark** (the transactional benchmark, but paid) or **Amazon SES** — worth it if a code failing to arrive is a real problem for you.
- **Highest volume / cheapest:** **Amazon SES** (~$0.10/1,000) or **Zoho ZeptoMail** (~$2.50/10,000). More setup, tiny bills.
- **Zero email at all:** use **Google/GitHub OAuth** login and skip this page entirely.

Remember the biggest free tier isn't automatically the best sender. Start with a free option, verify sign-in email actually arrives before anything else, and only move to a paid tier when you outgrow the daily cap or need stronger deliverability.
