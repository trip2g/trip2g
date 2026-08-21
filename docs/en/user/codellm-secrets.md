---
title: "Secrets for code roles"
free: true
lang_redirect: "[[ru/user/codellm-secrets]]"
---

A code role that pulls from an external API needs a token. There are two ways to give it one, and neither is a workaround for the other.

The usual way is the run environment: the operator of the codellm service holds the value, and the code reads it like any other environment variable. If you already run Docker secrets, Kubernetes secrets, or a Vault agent, this is where your credentials already live, and nothing here asks you to move them.

The other way is what Rails does with `credentials.yml.enc`: commit the encrypted value, keep the key out of the repo. The role note carries the ciphertext, codellm holds the key, and the value is decrypted for that one role at the moment it runs. Reach for it when a secret belongs to a single role, or when adding one should not mean editing a compose file and restarting.

In this article:

- [From the run environment](#from-the-run-environment)
- [Sealing a value](#sealing-a-value)
- [Declaring it in the role](#declaring-it-in-the-role)
- [Reading it in code](#reading-it-in-code)
- [What this protects](#what-this-protects)
- [Rotating keys](#rotating-keys)

## From the run environment

The codellm operator decides what code may see, with an allowlist of variable names:

```bash
CODELLM_EXPOSE_ENV=KRISP_TOKEN,KRISP_BASE_URL
CODELLM_EXPOSE_ENV_PREFIX=KRISP_
```

Nothing is exposed by default. A variable that appears on neither list never reaches the code, whatever a role asks for.

The role then reads it:

```python
import os
token = os.environ["KRISP_TOKEN"]
```

By default every code role on that codellm sees every allowlisted variable. A role can narrow its own share:

```yaml
env_passthrough: [KRISP_TOKEN]
env_prefix: [KRISP_]
```

This only narrows. The operator's allowlist stays the boundary, so declaring a variable the operator did not allow gets you nothing, and a role that declares neither field keeps seeing the whole allowlist. Narrowing is worth doing once several roles share one codellm and each has its own credential.

The cost of this path is that adding a secret is an operator action: a new variable means editing the deployment and restarting. That is the trade the second path removes.

## Sealing a value

The key is a 32-byte string in codellm's environment, named `SEAL_KEY` by default. Generate one once:

```bash
openssl rand -hex 16     # 32 characters
```

Put it in codellm's environment the way you keep any other server secret, then seal a value with it:

```bash
printf %s "$KRISP_TOKEN" | codellm seal
sealed:v1:9nR2v0QkX8tWc1pE...
```

The secret goes in on stdin, not as an argument: a flag would show up in the process table and in your shell history.

## Declaring it in the role

Paste the output into the role note's frontmatter and list which fields to open:

```yaml
---
fleet_id: codellm
mode: cron
cron_schedule: "*/30 * * * *"
write_patterns: ["transcripts/**"]

unseal: [krisp_token]
krisp_token: sealed:v1:9nR2v0QkX8tWc1pE...
krisp_base_url: https://api.krisp.ai
---
```

`unseal` is a list of field names, not a switch. Naming the fields means a bad blob fails at unseal time with a clear message, instead of travelling on as an ordinary string and coming back as a 401 from the API twenty minutes later.

Everything else in the frontmatter stays what it is. `krisp_base_url` is not a secret and needs no ceremony.

## Reading it in code

Two accessors, deliberately separate:

```python
import fleetkit

cfg = fleetkit.frontmatter()   # the role's own configuration
sec = fleetkit.secrets()       # what was opened for this run

base_url = cfg.krisp_base_url.rstrip("/")
token = sec.krisp_token
```

Secrets are not merged into the frontmatter, and the reason is practical: the delivery bag is the thing you print while debugging a role. A `print(cfg)` that included the token would publish it into a note, which is the one place this whole mechanism exists to keep it out of. Keeping the two apart means the object you are most likely to dump whole does not carry the secret.

The ciphertext stays visible in `cfg.krisp_token`, so a role can still tell that a field was sealed.

Node roles read the same two:

```javascript
const cfg = fleetkit.frontmatter();
const sec = fleetkit.secrets();
```

## What this protects

The boundary is narrower than the word "encrypted" suggests, so it is worth stating exactly.

The secret is not in cleartext anywhere the vault goes: not in Obsidian sync, not in the vault's git history, not in a note backup, not in a database dump. Someone who reads your notes learns nothing from them. Each blob is opened only for the role that carries it, so one careless role cannot spill another's credential into a note.

Whoever runs codellm can read every secret it opens. The code it executes needs the plaintext, so the plaintext exists in that process. This is not a way to keep a credential from your own server operator, and no storage scheme could be: the machine that uses a secret can see it.

A role that decides to write its token into a note will also succeed. Scope enforcement checks which paths a role may write, never what it writes there. What stops that is not encryption but the fact that you wrote the role.

The key itself never enters the sandbox. Code cannot decrypt anything on its own; it receives only the values its own note declared. codellm refuses to start if the key is configured to be passed into executed code, because that one line would undo the arrangement without anything appearing broken.

## Rotating keys

A role can name its own key:

```yaml
unseal: [krisp_token]
unseal_env_key: SEAL_KEY_V2
```

That makes rotation gradual instead of a flag day. Add `SEAL_KEY_V2` alongside the old key, re-seal and move roles one at a time, and drop the old key when nothing points at it.

Re-sealing means editing the note, so replacing a compromised upstream token is a note edit plus a sync, not a database update. And the old ciphertext stays in the vault's git history — it is unreadable without the key, but rotating the key does not erase it, so treat a leaked `SEAL_KEY` as a leak of everything sealed with it.
