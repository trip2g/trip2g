# Sync hazard: stale sync-state → silent "server deleted" divergence

## Summary

`trip2g-sync.mjs` keeps a client-side manifest `.sync-state.json`
(`{path: lastSyncedHash}`) in the vault root. When the **server loses notes the
client still believes it synced** (server DB reset, restore from an older backup,
or pointing the same vault at a *different/blank* server instance), every such note
is classified `server_deleted`. The current handler (`onServerDeleted`) then **keeps
the local copy and does NOT re-upload it** — silently, with only a `⚠️ N files
deleted on server, keeping local copies` line. The notes stay local-only, invisible
to the server (search, embeddings, federation), with no error and no commit failure.

Real impact observed: a vault of ~186 notes (wiki/topics, calls) existed locally but
never reached the server. `search` returned nothing for content that was plainly in
the vault. Data was **not lost** (the vault is the source of truth), but it was
**silently absent** from the server until the sync-state was cleared.

## Mechanism

Classification (`at(localHash, remoteHash, lastSyncedHash)`):

| local | server | state remembers it? | verdict | action |
|------|--------|---------------------|---------|--------|
| ✓ | ✗ | yes (`lastSyncedHash`) | `server_deleted` | keep local, **skip upload** |
| ✓ | ✗ | no | `local_only` | **push** |

So the *same* local file with the *same* "absent on server" gets the **opposite**
action based solely on whether `.sync-state.json` remembers it. The state is the
single source of this decision; there is no reconciliation against server identity.

## How it triggers

The hazard needs the server to diverge from what the client's state records:

1. **Asymmetric reset** (dev/test): the server DB/volume is wiped or recreated while
   the client vault + `.sync-state.json` persist. The most common way to hit it.
2. **Restore from an older backup** (prod DR): the server rolls back behind the
   client's synced state → client notes read as "deleted on server".
3. **Instance migration with a reused vault**: the same vault dir is pointed at a new
   or blank server. The state describes the *old* server.
4. **Server-side data loss / partial corruption.**

In steady-state operation (one stable, persistent server, never reset) it does not
trigger — client state and server stay consistent.

## Why it's dangerous

- **Silent.** No error, no failed commit — just a `⚠️` line and a benign-looking
  summary. The KB looks healthy locally while the server has none of it.
- **Indistinguishable intents.** "User deleted this note on the server on purpose"
  and "the server lost it" are identical to the client (in-state + absent). The tool
  guesses *deleted-on-purpose* and protects local-without-uploading — the opposite of
  what you want when the vault is the source of truth.
- **Worst at the worst time.** It fires precisely during restore / migration / data
  loss — high-stakes moments where silent divergence is most harmful.

## Why `--conflict-resolution` does NOT fix it

`--conflict-resolution {local|remote|skip|fail}` only governs **conflicts** (both
sides changed since last sync). `server_deleted` is a *different* branch, handled by a
hardcoded `onServerDeleted() → false` (keep local, never upload). No CLI flag changes
it. An agent choosing `--conflict-resolution` cannot influence this case.

## Recommended fixes (server + sync tool)

1. **Server identity / epoch in the protocol (best, automatic).** Have the server
   expose an instance id or vault epoch (e.g. in `FetchServerHashes`). Store it in
   `.sync-state.json`. On sync, if the server epoch ≠ the stored one, the state is for
   a different/reset server → invalidate it and treat all local files as `local_only`
   (re-push) rather than `server_deleted`. This removes the guesswork entirely.
2. **Explicit "server-missing" policy**, separate from `--conflict-resolution`, e.g.
   `--on-server-missing {push|delete|warn}` (default `warn`). `push` makes the vault
   authoritative (re-upload); `delete` is the current keep-local-skip behavior.
3. **Loud, actionable reporting.** `server_deleted` with a present local file should
   be a warning that *requires a decision* (or fails under `--conflict-resolution=fail`),
   not a silent skip buried in the summary.

## Operational workaround (until fixed)

Whenever the server is reset, restored, or swapped while reusing a vault:

```sh
rm <vault>/.sync-state.json   # forget stale state → next sync treats local as new
/opt/data/sync                # re-pushes everything (vault = source of truth)
```

Document this in any DR / migration runbook. An agent that drives sync should, on
seeing `N files deleted on server` for files it did **not** delete, clear the
sync-state and re-sync rather than accept the silent divergence — but the structural
fix (server epoch) should make that unnecessary.
