---
title: "A trip2g network on preemptible nodes"
free: true
lang_redirect: "[[ru/thoughts/surviving-node-loss]]"
---

Short version. A trip2g knowledge network ([[en/thoughts/markdown-vault|markdown vaults]] backed by SQLite) can run on preemptible nodes. They cost far less and can vanish at any moment. In a synthetic test the network survives it: a node dies, comes up on another machine, and restores its data in about 35 seconds, losing no more than 1 second of writes. What holds the data is not the local disk but a copy in object storage (Litestream). The rest is detail: what I tested, how it works, and where the limits are.

I tested on a throwaway two-node cluster, a real power-off via the Hetzner API. The test is synthetic, not production: it shows the mechanism holds, not that you will never lose data.

## What preemptible nodes are

Preemptible (spot) nodes can be taken back by the provider within seconds, for a much lower price. It takes them back hard: the hypervisor reclaims the machine, you get a SIGTERM and about 30 seconds. There is no polite "let me finish." For a knowledge network that chaos is acceptable under one condition: every node must come up with its data, on any machine.

What it saves (provider numbers): AWS Spot is usually 70-90% cheaper than on-demand, Google Cloud Spot 60-91%, Yandex Cloud preemptible a flat 50% (and they live no longer than 24 hours). So 2x to about 10x depending on provider and machine type. One caveat: what follows is a simulation (power-off via API on throwaway VMs), repeatable even on a home machine. The real saving comes from actual cloud spot; the test only shows the network survives how spot behaves.

## The warm cache migrates, the data doesn't

Nomad has `ephemeral_disk { sticky = true, migrate = true }`. The docs are accurate: when an allocation moves, the warm cache really does migrate to the new node. The trap is the word "moves": it means the graceful case, and there is no graceful case on spot.

- Graceful drain (`nomad node drain`): data migrates. A marker from node 2 showed up intact on node 1 in 27-34 seconds.
- Hard power-off (which is what preemption is): the allocation reschedules onto a fresh node and comes up empty. `migrate` copies from the old node, and the old node is dead. Nothing to copy.

Why `migrate` structurally cannot help here: for it to fire, the scheduler has to gracefully drain the allocation, which takes minutes. Spot gives you a SIGTERM and 30 seconds. It does not add up.

## What actually survives: a copy off the node

Only an off-node copy survives. Litestream streams the SQLite WAL to object storage (MinIO here) once a second, and on every start the allocation restores from object storage before the app boots. Same hard power-off:

| | `ephemeral_disk{migrate}` | Litestream in MinIO |
|---|---|---|
| Graceful drain | survives, ~26s | survives, 27-34s |
| Hard power-off | empty disk, data lost | fresh node, 33-35s |
| Loss window | the whole DB if there is no off-node copy (with a periodic backup, the backup interval) | no more than the sync interval (1s); ~0 in the run |

The database came back on a node that had never held it. The trick that makes it robust: restore on every start, not only on a fresh node. Then the object store becomes the single source of truth at every placement.

## The config

```hcl
# prestart: object store is the source of truth on every start
task "restore" {
  lifecycle { hook = "prestart" }
  # rm -f /data/db.sqlite3* && litestream restore -if-replica-exists -o /data/db.sqlite3 s3://bucket/db
}
# poststart sidecar: stream the WAL once a second
task "replicate" {
  lifecycle { hook = "poststart", sidecar = true }
  # litestream replicate /data/db.sqlite3 s3://bucket/db   (sync-interval: 1s)
}
# job: let Nomad recreate forever and declare the node lost fast
reschedule { unlimited = true }
disconnect { lost_after = "30s", replace = true }
```

Plus docker `volumes { enabled = true }` for the host bind, and keep the control-plane and object store on a stable node, not on spot.

## Where it loses

One bound: Litestream replicates once a second. A write landing under a second before the power-off is lost; if Litestream dies in the same window as the node, those WAL pages go with it. I measured about zero in this run, but that is luck of timing, not a zero-loss guarantee. Need it tighter: lower the interval and pay for more PUTs.

Between the two extremes (migrate with no copy = total loss, Litestream = no more than 1 second) there is a middle: a periodic off-node backup every N. Loss is then bounded by N (a minute, if you back up every minute), not "everything since placement." The downside of naive rotation: you keep only the last few points and can roll back ~5 minutes at most. The fix is exponential thinning (a point a minute ago, an hour, a day, 7 days) with continuous rotation: tight RPO and deep history at once. That is a separate piece of work.

## What's next: zero-downtime reads via a read replica

The restore above costs ~35 seconds where the network is rebuilding. Reads can stay up with no gap. Litestream can keep a continuously-updated read replica on a second node; if the writer's node is preempted, reads keep serving from the replica while a fresh writer comes back from object storage. That is a separate scenario with its own test and write-up (planned; this section will link to it once measured).

## Bottom line

- Durability on spot lives off the node: WAL to object storage, restore on every start.
- `ephemeral_disk migrate` is a warm cache for graceful moves, not durability. Spot has no graceful move.
- A node's death becomes a 35-second reassembly, not a loss.
- The loss window equals the sync interval (no more than 1 second here). This is a sandbox and a 2x-10x saving, not a zero-loss production guarantee.
