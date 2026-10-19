# Large file uploads

**Goal: an attachment of up to 64 MB uploads and downloads reliably.** Larger files are out of scope for now.

Today the default `--max-request-body-size` is 10 MB. Raising it to 64 MB alone is not enough: the problems below are what stand between the flag and the goal. They are recorded, not fixed.

## What limits a file today

There is no per-file limit. A file is bounded by the request that carries it:

- `--max-request-body-size` (MB, default 10) is set on both the fasthttp server and gqlgen's `MultipartForm` transport (`cmd/server/server.go`, `internal/graph/handler.go`).
- `--storage-assets-limit` and `--storage-db-limit` (`CheckStorageLimits`, `cmd/server/assets.go`) cap the total stored, not one file.

## Problems

### 1. The whole upload sits in memory, about twice

The path of `uploadNoteAsset`:

1. fasthttp reads the full body into memory, up to `MaxRequestBodySize`.
2. fasthttp pre-parses the multipart form (`DisablePreParseMultipartForm` is false); a file over 16 MB goes to a temp file.
3. `fasthttpadaptor` calls `PostBody()`, which re-encodes that parsed form into a new in-memory buffer.
4. gqlgen parses the rebuilt body again (`MaxMemory` 10 MB, the rest to temp files).

So one upload holds roughly two copies of the file in RAM plus temp files, and every concurrent upload adds its own. This is read from the code, not measured.

### 2. Field order is lost (the "first part must be operations" error)

Step 3 rebuilds the form from Go maps, so the parts come out in random order. gqlgen requires `operations` first and fails otherwise. About one upload in eight fails; reordering fields in the client does not help, because the server reorders them. A repro test is on branch `fix/graphql-multipart-order` (`internal/fastgql/multipart_order_test.go`).

### 3. 60 seconds for the whole request

`ReadTimeout`, `WriteTimeout` and the `TimeoutHandler` are all 60 s outside dev mode (`cmd/server/server.go`). Reading a 64 MB upload in 60 s needs about 8.5 Mbit/s upstream from the client; a slower link fails. Downloads from `/_system/assets` stream from storage with Range support, but a single 64 MB download is bound by the same `WriteTimeout` (not verified on a live server).

### 4. Every byte passes through the trip2g process

Uploads and downloads both go through the server; presigned S3 URLs are not used. This is fine at 64 MB and becomes the limit for video-sized files.

## Options, not decided

- `DisablePreParseMultipartForm: true` — removes step 2 and the reordering in step 3, which fixes problem 2 and one copy from problem 1. Everything else on the server that reads a multipart form must then parse it itself.
- A narrow wrapper that restores the field order before gqlgen (`fix/graphql-multipart-order`) — fixes problem 2 only.
- A separate upload endpoint that streams the file to storage outside the GraphQL adaptor — fixes problems 1–3 for uploads.
- Presigned upload and download URLs to S3 — fixes problems 1, 3 and 4 when storage is S3; local storage still needs one of the above.
- A longer timeout only for upload and asset routes — fixes problem 3 alone.
