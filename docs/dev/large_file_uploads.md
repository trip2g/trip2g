# Large file uploads

**Goal: an attachment of up to 64 MB uploads and downloads reliably.** Larger files are out of scope for now.

Status: problem 2 is fixed, and so are uploads through a read replica, which reached the leader with an empty body. The leader's internal listener takes the same `--max-request-body-size` as the public one; it used to fall back to fasthttp's default 4 MB. The public and internal servers no longer pre-parse multipart forms (`DisablePreParseMultipartForm: true`, `cmd/server/server.go`), so `fasthttpadaptor` passes the raw body through and drops one in-memory copy. The limits stay at 10 MB and 60 s. The up-front body reservation below is fixed; raising the limits still waits for per-caller limits.

## The request-body problem behind the limits

fasthttp used to reserve a buffer of the announced `Content-Length` as soon as it had read the headers (`appendBodyFixedSize`), then wait for the body until `ReadTimeout`. This happened before routing and before any credential was checked. So on every route of every instance an anonymous connection that announced 10 MB and sent a byte at a time held 10 MB for 60 s; a thousand of them held 10 GB.

**Fixed: the up-front reservation.** Both the public and the internal server run with `StreamRequestBody: true`. fasthttp then pre-reads at most 8 KB and hands the rest over as a stream, so memory grows only with the bytes that actually arrive. In this mode `MaxRequestBodySize` stops being a limit (a larger body, or any chunked one, becomes a stream that `PostBody()` would read whole), so one wrapper, `limitRequestBody` in `cmd/server/server.go`, sits in front of every handler on both servers. It answers 413 without reading when the declared `Content-Length` is over the limit; otherwise it reads the stream through a `limit+1` reader into the request, answering 413 when a chunked body runs over. Every existing reader (`PostBody()`, `Request.Body()`, `PostArgs()`, `fasthttpadaptor` for gqlgen, the replica forwarder's `CopyTo`, git, MCP, webhooks) still gets the whole body. The wrapper runs outside `TimeoutHandler`, so the read happens on the connection goroutine under its read deadline, and a timed-out handler goroutine never reads the connection the next keep-alive request arrives on. `Expect: 100-continue` works as before: fasthttp sends `100 Continue` and then the wrapper reads the body. The limits are unchanged: `--max-request-body-size` (10 MB) per request on both servers, 60 s timeouts on the public one. Tests: `cmd/server/request_body_test.go`.

**What remains:**

- **A read still holds memory up to the limit for up to `ReadTimeout`** when the attacker actually sends the bytes. The attack now costs bandwidth, not just headers.
- **The internal server has no `ReadTimeout`**, so a slow body there is bounded by nothing but the connection. It listens on the private network only.
- **No connection limits** (`MaxConnsPerIP`, `Concurrency`, a shorter header timeout). A thousand idle connections still cost a goroutine and about 8 KB each.
- **The limit is the same for every caller.** Who needs more than a small body: `pushNotes` sends batches of 100 notes (`assets/ui/sync/browser-sync.js`), `uploadNoteAsset` carries attachments, a git push lands whole in memory (`internal/gitapi/api.go`), and a replica forwards all of these to the leader. All of them come with an API key, a Bearer token or an admin session. Guests, users, webhooks and federation peers send small bodies; 1 MB would do for them. The wrapper is where a per-caller limit would go, once the caller is resolved before the read. Not decided.
- **With `--acme-domain` the HTTPS listener is `fasthttp.Serve(lnTLS, s.Handler)`**, a default server that uses none of the configured settings (no streaming, a 4 MB body limit, no timeouts). The wrapper still runs there, but the reservation is not removed on that path.
- **Other options, not decided:** a separate upload endpoint that checks credentials from the headers and streams the file to storage (changes the API for `uploadNoteAsset` clients); leaving fasthttp for `net/http`, where a body is a stream from the start, `http.MaxBytesReader` limits it per handler, `ReadHeaderTimeout` is separate from the body, and gqlgen needs no adaptor. fasthttp is imported by 43 non-test files with 95 uses of `RequestCtx`, plus `appreq` and SSE on `SetBodyStreamWriter`.

## What limits a file

There is no per-file limit. A file is bounded by the request that carries it:

- `--max-request-body-size` (MB, default 10) is set on both the fasthttp server and gqlgen's `MultipartForm` transport (`cmd/server/server.go`, `internal/graph/handler.go`).
- `--storage-assets-limit` and `--storage-db-limit` (`CheckStorageLimits`, `cmd/server/assets.go`) cap the total stored, not one file.

## Problems as first found

### 1. The whole upload sits in memory, about twice

The path of `uploadNoteAsset`:

1. fasthttp parses the multipart form straight from the connection (`DisablePreParseMultipartForm` was false); a file over 16 MB goes to a temp file.
2. The raw body is not kept, which is also why a read replica forwarded an empty body to the leader.
3. `fasthttpadaptor` calls `PostBody()`, which re-encodes that parsed form into a new in-memory buffer.
4. gqlgen parses the rebuilt body again (`MaxMemory` 10 MB, the rest to temp files).

So one upload holds roughly two copies of the file in RAM plus temp files, and every concurrent upload adds its own. This is read from the code, not measured.

### 2. Field order is lost (the "first part must be operations" error)

Step 3 rebuilds the form from Go maps, so the parts come out in random order. gqlgen requires `operations` first and fails otherwise. About one upload in eight fails; reordering fields in the client does not help, because the server reorders them. Fixed; `cmd/server/multipart_order_test.go` sends 400 uploads to the public and the internal server and fails on any of them.

### 3. 60 seconds for the whole request

`ReadTimeout`, `WriteTimeout` and the `TimeoutHandler` are all 60 s outside dev mode (`cmd/server/server.go`). Reading a 64 MB upload in 60 s needs about 8.5 Mbit/s upstream from the client; a slower link fails. Downloads from `/_system/assets` stream from storage with Range support, but a single 64 MB download is bound by the same `WriteTimeout` (not verified on a live server).

### 4. Every byte passes through the trip2g process

Uploads and downloads both go through the server; presigned S3 URLs are not used. This is fine at 64 MB and becomes the limit for video-sized files.

## Options for problems 1, 3 and 4, not decided

- A separate upload endpoint that streams the file to storage outside the GraphQL adaptor — fixes problems 1–3 for uploads.
- Presigned upload and download URLs to S3 — fixes problems 1, 3 and 4 when storage is S3; local storage still needs one of the above.
- A longer timeout only for upload and asset routes — fixes problem 3 alone, and stretches the request-body problem above on those routes.
