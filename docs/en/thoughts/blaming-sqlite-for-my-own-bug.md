---
title: "I spent half a year blaming SQLite for my own bug"
free: true
lang_redirect: "[[ru/thoughts/blaming-sqlite-for-my-own-bug]]"
---

*What this is: the story of a `SQLITE_BUSY` error I chased for half a year, blamed on SQLite, and finally found in my own connection pool. Part war story, part the case for running a database inside one process. Read it if you have ever seen "database is locked" and wondered whether SQLite was the wrong call.*

The error was mine. For about half a year trip2g kept throwing `SQLITE_BUSY` and `database is locked` under load, and for about that same half a year I kept deciding the problem was SQLite. It was not. I had the connection pool configured to lie to me about how many writers I had. The day I finally understood that is the day I stopped being afraid of running a database inside my own process, and started to actually like it.

I came to this from years on Postgres. On Postgres you do not think about how many writers there are. You open a pool of thirty connections, they all write whenever they want, and the server sorts it out. So when I moved trip2g onto a single embedded SQLite file, the model felt wrong in my hands. One writer at a time, even in WAL mode? It read like a limitation I would have to route around, not a contract I should design for. I spent months treating it as the former, and I was quietly furious at the engine the whole time.

---

## The fixes that each made the error rarer

My first move, back in August, was the bluntest one possible. I set `SetMaxOpenConns(1)` on the shared pool. BUSY vanished, because now there was physically one connection and nothing could race it. It also broke sign in. A slow write would park the only connection, and a request that just wanted to read a session cookie sat in line behind it. I left a `// TODO: it breaks sign in` in the code, reverted the pool to 10, then to 25, and BUSY came straight back.

The same day I added a `WithRetry` helper that caught the busy error and tried again with a little jitter. It worked the way a band-aid works. Retries only hid the contention. I poured in pragmas too: `busy_timeout`, `_txlock=immediate`, WAL, a bigger cache. Every change made the error rarer. None of them made it stop. I read that as SQLite being temperamental. It was the opposite. SQLite was being perfectly consistent, and I was the variable.

Lesson: a bug that gets rarer with every fix but never dies is usually one bug, not many.

---

## The bug was that configured is not applied

The thing that finally cracked it was small and a little humiliating. I had `busy_timeout = 20000` in the config. Twenty seconds. And the database was returning `database is locked` instantly, with no wait at all. A twenty second timeout firing in zero seconds should have stopped me cold months earlier. It did not, because I never asked why.

An Anthropic model I had been pairing with, Fable 5, asked. It read the extended result code instead of the headline number, noticed the value in the parentheses was 517 and not a plain 5, and then went and read the driver source. trip2g runs on `modernc.org/sqlite`, the pure-Go driver. I had written the connection string in the mattn style I half-remembered from somewhere: `_busy_timeout`, `_journal`, `_timeout`. modernc does not understand any of those. It supports `_pragma`, `_txlock`, and `_time_format`, and it silently drops everything else. No error. No warning. Every pragma I thought I was setting was thrown on the floor.

There was a fallback that ran a one-off `PRAGMA busy_timeout = 20000` at startup, which is how I had fooled myself into thinking it took. But `database/sql` is a pool, not a connection. A one-off Exec configures exactly the one connection it happens to grab. The read pool holds up to 25. So one connection out of twenty-five carried my timeout, and the other twenty-four came up with raw SQLite defaults: `busy_timeout=0`, `foreign_keys=OFF`. The startup log said pragmas configured, and it was telling the truth about one connection in twenty-five.

The 517 was the other half of it. A plain deferred `BEGIN` starts life as a reader and takes a read snapshot. When that transaction later tries to write, after some other connection has committed in between, it is upgrading on a snapshot that is already stale, and SQLite returns `SQLITE_BUSY_SNAPSHOT` (517) at once. busy_timeout can never help there, because no amount of waiting un-stales a snapshot. The cure is `_txlock=immediate`, so a write transaction takes the write lock at `BEGIN` and queues politely instead of failing. I had that pragma in the string the entire time. The driver had been ignoring it along with all the others.

Lesson: configured and applied are different words. A green startup log tells you about one connection. The rest will only tell you the truth if you pin them and ask.

---

## And the background jobs were quietly stealing the writer

Even with the pragmas finally landing on every connection, there was a second bug, and this one was pure architecture. Months earlier I had wrapped every GraphQL mutation in a write transaction. Convenient. Every mutation is atomic by default. It also means that for the whole duration of a mutation, that one request owns the single write connection. Anything the mutation kicks off that also wants to write is now asking for a connection the same request is already holding.

Plenty of things wanted to write inside a mutation. Saving a note enqueued a Telegram publish job, and the enqueue used the global app context instead of the request's, so it lost the transaction and went looking for its own write connection. Background jobs created inside a transaction were being created outside its scope, which produced more 517s and left duplicate jobs behind on rollback. Cron runs and nested admin mutations opened their own transactions and deadlocked on the write connection the middleware was already holding. Those needed a `@skipTx` directive to opt out.

The worst offender was the queue itself. goqite polls SQLite on a timer, firing DELETE and UPDATE against its table forever. I had it running on the write pool. So the queue poller and the application were in a permanent standoff over the single write slot, and no busy_timeout in the world fixes a contender that never stops knocking. On the first of May I gave goqite its own dedicated connection, separate from the write pool. That, with the `@skipTx` work the day before, is where the saga actually ends. Nine months from the first BUSY-specific commit to the last one, and six of them, the bad ones, spent watching my own code fight itself for one writer.

---

## What changed when I stopped babysitting the database

Here is the part I did not see coming. Once the writer was honest, I stopped babysitting the database entirely.

For years on Postgres I had performed a small ritual for every new tenant. Do I run `CREATE DATABASE` inside the existing cluster, or stand up a new server? Who else lives in this instance? If I get a `WHERE tenant_id = ?` wrong, whose data leaks into whose page? In trip2g each knowledge base is its own SQLite file, owned by its own process. There is no `tenant_id` column anywhere, because there is nothing to discriminate. The whole database is the tenant. A multitenant bug or a wrong parameter physically cannot reach a neighbor's data, because the neighbor's rows are in a different file, behind a different process, usually on a different host. I isolate the mesh of bases at the network level, not with a careful clause buried in a query. The strongest boundary I can draw turned out to be the one I no longer have to remember to draw.

I also made peace with N+1. On Postgres, two hundred small queries to render one page is a sin, because every one of them is a network round trip. SQLite is not a server. It runs in the same address space as the application, so a query is a function call rather than a message on a wire. The official page says it plainly: [many small queries are efficient in SQLite](https://www.sqlite.org/np1queryprob.html). David Crawshaw makes the bigger version of the argument in his [one-process programming notes](https://crawshaw.io/blog/one-process-programming-notes): do not use N computers when one will do. I had read both before any of this. I only believed them after I stopped fighting the writer.

---

## One process, all the way down

What I defend now is not only the database. It is the shape. trip2g is a single Go binary. The job queue is goqite, which is just another table in the same SQLite file, so there is no Redis to run and no broker to keep alive. Full-text search, the page cache, the queue, and the data all live in the one process. Go cross-compiles that process to a static binary for most platforms, so it runs on a small VM, an old laptop, or whatever hardware is lying around, with nothing to install beside it. The reason I can put a tenant on its own machine is the same reason the whole thing fits on weak hardware: there is only one moving part.

---

## Thank you

None of this would exist without the people who made single-process SQLite a serious way to ship software. D. Richard Hipp and the SQLite team built the engine, and the precise extended codes I had been ignoring, 5 and 517, are the only reason the bug was findable at all. David Crawshaw wrote the one-process notes and an early Go binding, and gave me the mental model that one writer is a design and not a wound. Jan Mercl maintains the pure-Go modernc driver trip2g actually runs on, including the `_pragma` mechanism that turned out to be the entire fix. Markus Wüstenberg wrote goqite, and pointing it at its own connection was the commit that finally closed the case. And Ben Johnson built [Litestream](https://litestream.io/) and LiteFS, which is what lets a single file behave like durable, replicated, restorable production storage. Without Litestream and friends there would be no trip2g. I mean that literally.

---

## What I would tell past me

Decide where your single writer lives on day one. One write connection, taken with `BEGIN IMMEDIATE`, a generous `busy_timeout` for the rare honest wait, and a separate read pool that fans out as wide as you like. Anything that writes on a schedule, queues and crons most of all, gets its own connection so it never competes with a request. Put the pragmas in the DSN, not in a one-off Exec, and then pin a handful of connections and check that they actually took.

BUSY was never SQLite being difficult. It was SQLite telling me, patiently, for half a year, that my pool was lying about how many writers I had. I just had to stop arguing long enough to listen.
