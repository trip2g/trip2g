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
	AssetOwnership(sha256Hash string) (assetindex.Ownership, bool)
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

	env := req.Env.(Env)
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

	ownership, known := env.AssetOwnership(hash)
	public := known && ownership.Public

	if !public {
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

	offset, length := int64(0), asset.Size
	status := http.StatusOK

	if rangeHdr := string(ctx.Request.Header.Peek(fasthttpRangeHeader)); rangeHdr != "" {
		start, end, satisfiable, applied := parseRange(rangeHdr, asset.Size)
		if !satisfiable {
			h.Set("Content-Range", fmt.Sprintf("bytes */%d", asset.Size))
			ctx.SetStatusCode(http.StatusRequestedRangeNotSatisfiable)
			return true
		}
		if applied {
			offset, length = start, end-start+1
			status = http.StatusPartialContent
			h.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, asset.Size))
		}
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
func parsePath(path string) (hash, fileName string, ok bool) {
	rest := path[len(model.AssetRoutePrefix):]
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return "", "", false
	}
	hash, fileName = rest[:slash], rest[slash+1:]
	if len(hash) != 64 || !isHex(hash) {
		return "", "", false
	}
	if fileName == "" || strings.ContainsAny(fileName, "/\\") {
		return "", "", false
	}
	return hash, fileName, true
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
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
func parseRange(header string, size int64) (start, end int64, satisfiable, applied bool) {
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
