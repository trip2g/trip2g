---
title: "A knowledge base in one container"
free: true
lang_redirect: "[[ru/user/docs-image]]"
---

`Dockerfile.docs` in the repository builds an image with a vault baked in. Run it and the site is up: no volume, no sign-in, no sync client on the side. The image is the content; to change a page, rebuild.

```bash
docker build -f Dockerfile.docs -t trip2g-docs .
docker run --rm -p 8080:8080 trip2g-docs
```

Then open `http://localhost:8080/en/user`, or point an MCP client at `http://localhost:8080/_system/mcp`. Every note is marked free by the patches under `patches/`, so both work without a key.

## What the image holds

By default, this documentation: `docs/en/user`, `docs/ru/user` and `docs/patches`. To ship your own vault, replace the three `COPY` lines with your folder. Frontmatter patches travel with the notes, so `free: true`, `lang` and sidebars are set once for the whole folder, not per page.

## How it boots

The entrypoint starts trip2g, waits for it to answer, fetches the sync client from the instance's own onboarding archive, pushes `/vault` once and then keeps serving. The database and the secrets live inside the container and are created at start, which is why nothing has to be mounted or configured.

`PUBLIC_URL` is the one setting worth overriding: it is what the instance stamps into links and sign-in URLs. Pass it with `-e PUBLIC_URL=https://docs.example.com`.

## Instructions for agents

An MCP client reads the note marked `mcp_method: instructions` as the base's instructions. When several sections keep their own, each under its own method name, a client picks one with `?method=<name>` in the MCP address: `http://localhost:8080/_system/mcp?method=agent_instructions`. See [[en/user/mcp]] for the rest of the surface.
