// Package serveasset serves note and layout assets over the stable,
// content-addressed route GET /_system/assets/{sha256}/{fileName}.
//
// Access model (security boundary — fail closed):
//   - Asset reachable via a layout or a publicly readable note → served
//     anonymously with immutable caching.
//   - Otherwise a session is required and the requester must be able to read
//     at least one owning note (canreadnote semantics via env.CanReadNote).
//   - Hash known in the DB but not referenced by any live note or layout →
//     admin-only.
package serveasset

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/assetindex"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg serveasset_test . Env

type Env interface {
	Logger() logger.Logger
	NoteAssetsBySha256Hash(ctx context.Context, sha256Hash string) ([]db.NoteAsset, error)
	// AssetOwnership reports ownership for the exact (hash, fileName) identity
	// — never for the hash alone. Two note_assets rows can share a hash while
	// having different file names and different owning notes (e.g. a public
	// asset and an unrelated private one that happen to be byte-identical);
	// keying by hash alone would let the private row's fileName inherit the
	// public row's access decision.
	AssetOwnership(sha256Hash, fileName string) (assetindex.Ownership, bool)
	CanReadNote(ctx context.Context, note *model.NoteView) (bool, error)
	// StreamAssetObject streams length bytes of the asset starting at offset,
	// without buffering the whole object in memory.
	StreamAssetObject(ctx context.Context, asset db.NoteAsset, offset, length int64) (io.ReadCloser, error)
}

const privateMaxAge = 300 // seconds; per-user access is re-checked after this

// Handle serves GET /_system/assets/{sha256}/{fileName}. It reports whether
// the request was handled (i.e. the path matched the asset route).
func Handle(req *appreq.Request) bool {
	if !strings.HasPrefix(req.Path, model.AssetRoutePrefix) {
		return false
	}

	env, _ := req.Env.(Env)
	ctx := req.Req

	if !ctx.IsGet() && !ctx.IsHead() {
		ctx.SetStatusCode(http.StatusMethodNotAllowed)
		return true
	}

	hash, fileName, ok := parsePath(req.Path)
	if !ok {
		notFound(ctx)
		return true
	}

	rows, err := env.NoteAssetsBySha256Hash(ctx, hash)
	if err != nil {
		env.Logger().Error("serveasset: asset lookup failed", "hash", hash, "error", err)
		ctx.SetStatusCode(http.StatusInternalServerError)
		return true
	}

	// The fileName must match a stored row exactly: it selects the storage key
	// and prevents serving known bytes under an attacker-chosen name (and thus
	// an attacker-chosen Content-Type).
	var asset *db.NoteAsset
	for i := range rows {
		if rows[i].FileName == fileName {
			asset = &rows[i]
			break
		}
	}
	if asset == nil {
		notFound(ctx)
		return true
	}

	ownership, known := env.AssetOwnership(hash, fileName)
	public := known && ownership.Public

	if !public && enforceAccess(req, env, ownership, hash) {
		return true
	}

	h := &ctx.Response.Header
	h.Set("ETag", `"`+hash+`"`)
	h.Set("Accept-Ranges", "bytes")
	h.Set("X-Content-Type-Options", "nosniff")
	if public {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		h.Set("Cache-Control", "private, max-age="+strconv.Itoa(privateMaxAge))
	}

	if inm := string(ctx.Request.Header.Peek("If-None-Match")); strings.Contains(inm, hash) {
		ctx.SetStatusCode(http.StatusNotModified)
		return true
	}

	ctx.SetContentType(contentType(fileName))

	offset, length, status, handled := resolveRange(ctx, h, asset)
	if handled {
		return true
	}

	ctx.SetStatusCode(status)

	if ctx.IsHead() {
		h.SetContentLength(int(length))
		return true
	}

	reader, err := env.StreamAssetObject(ctx, *asset, offset, length)
	if err != nil {
		env.Logger().Error("serveasset: failed to open asset stream", "hash", hash, "error", err)
		ctx.SetStatusCode(http.StatusInternalServerError)
		return true
	}

	ctx.SetBodyStream(reader, int(length))
	return true
}

// enforceAccess applies the non-public access rules. It returns true when it
// has written a 401/403/500 response and the caller must stop; false when the
// requester is allowed to read the asset.
func enforceAccess(req *appreq.Request, env Env, ownership assetindex.Ownership, hash string) bool {
	ctx := req.Req

	token, tokenErr := req.UserToken()
	if tokenErr != nil || token == nil {
		ctx.SetStatusCode(http.StatusUnauthorized)
		ctx.SetBodyString("401 Unauthorized")
		return true
	}

	allowed := token.IsAdmin()
	for _, note := range ownership.Notes {
		if allowed {
			break
		}
		canRead, readErr := env.CanReadNote(ctx, note)
		if readErr != nil {
			env.Logger().Error("serveasset: access check failed", "hash", hash, "error", readErr)
			ctx.SetStatusCode(http.StatusInternalServerError)
			return true
		}
		allowed = canRead
	}

	if !allowed {
		ctx.SetStatusCode(http.StatusForbidden)
		ctx.SetBodyString("403 Forbidden")
		return true
	}

	return false
}

// resolveRange applies a Range header (if any) to asset, setting Content-Range
// headers as needed. It returns the byte offset/length to stream, the response
// status code, and handled=true when a 416 response was already written and the
// caller must stop.
func resolveRange(ctx *fasthttp.RequestCtx, h *fasthttp.ResponseHeader, asset *db.NoteAsset) (int64, int64, int, bool) {
	offset, length := int64(0), asset.Size
	status := http.StatusOK

	rangeHdr := string(ctx.Request.Header.Peek(fasthttpRangeHeader))
	if rangeHdr == "" {
		return offset, length, status, false
	}

	start, end, satisfiable, applied := parseRange(rangeHdr, asset.Size)
	if !satisfiable {
		h.Set("Content-Range", fmt.Sprintf("bytes */%d", asset.Size))
		ctx.SetStatusCode(http.StatusRequestedRangeNotSatisfiable)
		return 0, 0, 0, true
	}
	if applied {
		offset, length = start, end-start+1
		status = http.StatusPartialContent
		h.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, asset.Size))
	}

	return offset, length, status, false
}

const fasthttpRangeHeader = "Range"

func notFound(ctx interface {
	SetStatusCode(int)
	SetBodyString(string)
}) {
	ctx.SetStatusCode(http.StatusNotFound)
	ctx.SetBodyString("404 Not Found")
}

// parsePath splits "/_system/assets/{sha256}/{fileName}" and validates the
// hash (64 lowercase-insensitive hex chars) and fileName (single segment).
func parsePath(path string) (string, string, bool) {
	rest := path[len(model.AssetRoutePrefix):]
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return "", "", false
	}
	hash, fileName := rest[:slash], rest[slash+1:]
	if len(hash) != 64 || !isHex(hash) {
		return "", "", false
	}
	if fileName == "" || strings.ContainsAny(fileName, "/\\") {
		return "", "", false
	}
	return hash, fileName, true
}

func isHex(s string) bool {
	for i := range len(s) {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func contentType(fileName string) string {
	if t := mime.TypeByExtension(filepath.Ext(fileName)); t != "" {
		return t
	}
	return "application/octet-stream"
}

// parseRange parses a single-range "bytes=" header against size.
// applied=false with satisfiable=true means the header should be ignored
// (multi-range or malformed) and the full body served with 200.
// satisfiable=false means respond 416.
func parseRange(header string, size int64) (int64, int64, bool, bool) {
	spec, found := strings.CutPrefix(header, "bytes=")
	if !found || strings.Contains(spec, ",") {
		return 0, 0, true, false // ignore: not a byte range / multi-range
	}

	first, last, found := strings.Cut(spec, "-")
	if !found {
		return 0, 0, true, false
	}

	switch {
	case first == "" && last == "": // "bytes=-"
		return 0, 0, true, false
	case first == "": // suffix range "-n"
		n, err := strconv.ParseInt(last, 10, 64)
		if err != nil {
			return 0, 0, true, false
		}
		if n <= 0 || size == 0 {
			return 0, 0, false, false
		}
		if n > size {
			n = size
		}
		return size - n, size - 1, true, true
	default:
		s, err := strconv.ParseInt(first, 10, 64)
		if err != nil || s < 0 {
			return 0, 0, true, false
		}
		if s >= size {
			return 0, 0, false, false
		}
		e := size - 1
		if last != "" {
			e, err = strconv.ParseInt(last, 10, 64)
			if err != nil {
				return 0, 0, true, false
			}
			if e < s {
				return 0, 0, false, false
			}
			if e > size-1 {
				e = size - 1
			}
		}
		return s, e, true, true
	}
}
