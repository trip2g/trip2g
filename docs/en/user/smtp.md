---
title: "Email delivery (SMTP)"
free: true
home_position: 62
lang_redirect: "[[ru/user/smtp]]"
---

To sign in, trip2g emails you a one-time code. No working email = no code = you can't get in. So a self-hosted instance needs a way to send mail. The pragmatic answer is a hosted transactional-email service: pick one below, drop its SMTP credentials into your config, done.

Two ways to skip email entirely, before you commit to a provider:

- **Sign in with Google or GitHub (OAuth).** No email, no SMTP, no provider account. If OAuth login covers everyone who needs access, you don't need any of this page.
- **Don't run your own mail server.** On a fresh VPS, outbound port `25` is almost always blocked by the host, and mail from a brand-new IP gets spam-filtered or rejected outright. Building deliverability takes weeks. A hosted service already has warm IPs and the right DNS reputation — that's why it's the practical path.

See [[selfhosted]] for the full self-hosting guide; this page is only the email piece.

### Which provider?

All of these give you an SMTP username and password you paste straight into trip2g. Sorted by usefulness for a small self-host: generous forever-free tiers with plain SMTP first, trial-only and pay-as-you-go last.

Figures are current as of mid-2026. Free tiers change often — **confirm on the provider's own pricing page before you rely on a number.**

| Service | Free tier | SMTP? | API? | Catch |
|---|---|---|---|---|
| **Brevo** (ex-Sendinblue) | 300/day, forever (~9,000/mo) | Yes | Yes | Most generous forever-free SMTP. 300/day hard cap. |
| **Mailjet** | 6,000/mo, forever | Yes | Yes | 200/day cap; over-cap mail queues up to 3 days then is dropped. |
| **Resend** | 3,000/mo, forever | Yes | Yes | 100/day cap. trip2g's default/native provider (see below). |
| **Elastic Email** | 3,000/mo (100/day), forever | Yes | Yes | Free-tier mail carries an Elastic Email footer/branding. |
| **SMTP2GO** | 1,000/mo (200/day), forever | Yes | Yes | Low monthly cap; fine for a single-owner instance. |
| **Mailgun** | 100/day, forever | Yes | Yes | 1 sending domain, 1-day log retention; a card may be required at signup — verify. |
| **Scaleway TEM** | 300/mo, forever | Yes | Yes | Then €0.25/1,000. EU provider; needs a Scaleway account + billing set up. |
| **Amazon SES** | 3,000 msg/mo — first 12 months only | Yes | Yes | Not forever-free (old 62k EC2 tier is gone). Then ~$0.10/1,000 — cheapest at volume. New accounts start in a sandbox; request production access. |
| **Zoho ZeptoMail** | 10,000-email credit, ~1 month trial | Yes | Yes | Trial credit, not recurring. Then ~$2.50/10,000 pay-as-you-go — very cheap. |
| **Postmark** | 100/**month** — testing only | Yes (paid) | Yes | Real sending needs a paid plan (~$15/mo). Strong deliverability reputation. |
| **SendGrid** (Twilio) | 60-day trial, 100/day | Yes | Yes | No longer has a permanent free tier — it became a 60-day trial in 2025. Paid from ~$19.95/mo. |
| **MailerSend** | 500/mo, forever | Yes | Yes | Allowance was cut from 3,000 to 500/mo in Dec 2025 — check it hasn't moved again. |

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

**Legacy Resend shortcut.** Older instances configured mail with a single `RESEND_API_KEY`. That still works: if `SMTP_PASS` is unset but `RESEND_API_KEY` is present, trip2g maps it onto Resend's SMTP gateway automatically (and logs a deprecation warning). Prefer the `SMTP_*` variables for anything new.

**No email yet?** trip2g runs fine without SMTP — it just skips sign-in emails. To bootstrap the very first login, set `LOG_SIGN_IN_CODES=true`, restart, trigger a sign-in, and read the code from the server log (`journalctl -u trip2g`). Unset it once you're in — it prints codes in plaintext.

### Which should I pick?

- **Easiest start:** **Brevo** (300/day forever, plain SMTP) or **Resend** (native to trip2g, clean setup). Both are free and enough for a personal or small-team instance.
- **Highest volume / cheapest:** **Amazon SES** (~$0.10/1,000) or **Zoho ZeptoMail** (~$2.50/10,000). More setup, tiny bills.
- **Zero email at all:** use **Google/GitHub OAuth** login and skip this page entirely.

Start with the free option, verify sign-in email works before anything else, and only move to a paid tier when you actually outgrow the daily cap.
