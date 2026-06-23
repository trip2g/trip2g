---
home_position: 60
title: "Hosting"
lang_redirect: "[[ru/user/hosting]]"
free: true
---

Three ways to get a running trip2g instance.

### Demo

Try it at [simplecloud.2pub.me](https://simplecloud.2pub.me). It's a free shared instance — no registration, no payment. Just open it and log in with your email.

Good for: seeing how the service works before committing to anything.

Not for: publishing real content. It's a demo environment, not a permanent home for your site.

### Managed hosting

Send an email to alexes.dev@gmail.com. I'll install trip2g on your server and get it running. You'll need a server (any VPS works) and a domain.

Good for: people who want their own instance without dealing with setup.

### Self-hosting

Use the minimal self-hosting guide with `docker-compose.yml`, `.env`, `ghcr.io/trip2g/trip2g:0.2`, MinIO, Resend, and optional vector search:

- [[en/user/selfhosted|Deploy on your own server]]
- [[en/user/fly|Deploy on fly.io]] — one machine, one volume, no extra services
- [github.com/trip2g/trip2g](https://github.com/trip2g/trip2g)

trip2g is a single Go binary with SQLite. For a simple production deployment, Docker, MinIO, and one `.env` file are enough.

Good for: developers who want full control.
