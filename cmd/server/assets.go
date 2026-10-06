package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"trip2g/assets"
	"trip2g/internal/case/uploadnoteasset"
	"trip2g/internal/db"
	graphmodel "trip2g/internal/graph/model"

	"github.com/valyala/fasthttp"
)

func (a *app) StorageDBLimit() int64 {
	return int64(a.config.StorageDBLimit)
}

func (a *app) StorageAssetsLimit() int64 {
	return int64(a.config.StorageAssetsLimit)
}

func (a *app) CheckStorageLimits(ctx context.Context, additionalAssetBytes int64) (string, error) {
	if limit := int64(a.config.StorageDBLimit); limit > 0 {
		info, err := os.Stat(a.config.DatabaseFile)
		if err != nil {
			return "", fmt.Errorf("failed to stat database file: %w", err)
		}

		if info.Size() >= limit {
			return "database storage limit exceeded", nil
		}
	}

	if limit := int64(a.config.StorageAssetsLimit); limit > 0 {
		currentSize, err := a.SumNoteAssetsSizes(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get current assets size: %w", err)
		}

		if currentSize+additionalAssetBytes > limit {
			return "assets storage limit exceeded", nil
		}
	}

	return "", nil
}

// ReadAssetObject streams an asset's bytes from object storage.
func (a *app) ReadAssetObject(ctx context.Context, asset db.NoteAsset) (io.ReadCloser, error) {
	return a.GetAssetObject(ctx, asset)
}

func (a *app) UploadNoteAsset(ctx context.Context, input graphmodel.UploadNoteAssetInput) (graphmodel.UploadNoteAssetOrErrorPayload, error) {
	return uploadnoteasset.Resolve(ctx, a, input)
}

func (a *app) setupAssets() {
	a.assetsFS = &fasthttp.FS{
		FS:                 assets.FS,
		IndexNames:         []string{},
		GenerateIndexPages: false,
		Compress:           !a.config.DevMode,
		SkipCache:          a.config.DevMode,
		AcceptByteRange:    true,

		PathRewrite: func(ctx *fasthttp.RequestCtx) []byte {
			// remove /assets prefix
			return ctx.Path()[7:]
		},
	}

	// initialize asset hashes map
	a.assetHashes = make(map[string]string)
}

// assetsHandler serves /assets/ with explicit caching. fasthttp.FS reports the
// embed.FS zero ModTime as Last-Modified, so it would answer any
// If-Modified-Since with 304 even after a deploy; revalidation uses a
// content-hash ETag instead.
func (a *app) assetsHandler() fasthttp.RequestHandler {
	fsHandler := a.assetsFS.NewRequestHandler()

	return func(ctx *fasthttp.RequestCtx) {
		ctx.Request.Header.Del("If-Modified-Since")

		hash, ok := a.assetHash(string(ctx.Path()[len("/assets/"):]))
		if !ok {
			fsHandler(ctx)
			return
		}

		// Weak: the same tag covers the gzip/br/identity encodings.
		etag := `W/"` + hash + `"`
		cacheControl := "public, max-age=3600"
		switch {
		case a.config.DevMode:
			cacheControl = "no-cache"
		case string(ctx.QueryArgs().Peek("h")) == hash[:8]:
			cacheControl = "public, max-age=31536000, immutable"
		}

		if strings.Contains(string(ctx.Request.Header.Peek("If-None-Match")), hash) {
			ctx.NotModified()
		} else {
			fsHandler(ctx)
			ctx.Response.Header.Del("Last-Modified")
		}

		ctx.Response.Header.Set("ETag", etag)
		ctx.Response.Header.Set("Cache-Control", cacheControl)
	}
}

// assetHash returns the sha256 of an embedded asset, cached outside dev mode.
func (a *app) assetHash(assetPath string) (string, bool) {
	a.assetsMu.Lock()
	defer a.assetsMu.Unlock()

	if hash, exists := a.assetHashes[assetPath]; exists && !a.config.DevMode {
		return hash, true
	}

	content, err := fs.ReadFile(assets.FS, assetPath)
	if err != nil {
		return "", false
	}

	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])

	if !a.config.DevMode {
		a.assetHashes[assetPath] = hash
	}

	return hash, true
}

// TODO: read all asset urls from flags.
func (a *app) assetURL(path string) string {
	// Remove leading / if it exists
	assetPath := path
	assetPath = strings.TrimPrefix(assetPath, "/")

	// Remove /assets/ prefix if it exists
	assetPath = strings.TrimPrefix(assetPath, "assets/")

	hash, ok := a.assetHash(assetPath)
	if !ok {
		a.log.Debug("asset file not found", "path", assetPath, "original", path)
		return path
	}

	return path + "?h=" + hash[:8]
}

func (a *app) AdminJSURL() string {
	return a.assetURL(a.config.AdminJSURL)
}

func (a *app) UserJSURLs() []string {
	// Core bootstrap, loaded on every page. Per-language widget glue
	// (chart.js, mermaid.js) is appended conditionally per note — see
	// rendernotepage.buildDefaultTemplateCtx.
	return []string{
		a.assetURL("/assets/defaulttemplate.js"),
		a.assetURL("/assets/ui/user/-/web.js"),
	}
}

// UserLocaleHashes returns a map of language -> 8-char content hash for the user
// web.locale=<lang>.json files. The locale JSON is a separate artifact from web.js,
// so it must be cache-busted by its own content hash; otherwise locale-only changes
// keep the same URL and the browser/CDN serves a stale locale file. Reuses assetURL
// so hashing/caching stays single-source.
func (a *app) UserLocaleHashes() map[string]string {
	return a.localeHashes("ui/user/-")
}

// EditorJSURL is the page editor's bundle. Only the editor frame loads it, so
// visitors who never open the editor don't download it.
func (a *app) EditorJSURL() string {
	return a.assetURL("/assets/ui/editor/pane/-/web.js")
}

func (a *app) EditorLocaleHashes() map[string]string {
	return a.localeHashes("ui/editor/pane/-")
}

func (a *app) localeHashes(bundleDir string) map[string]string {
	out := map[string]string{}
	matches, err := fs.Glob(assets.FS, bundleDir+"/web.locale=*.json")
	if err != nil {
		a.log.Error("failed to glob locale files", "dir", bundleDir, "error", err)
		return out
	}
	for _, m := range matches {
		base := path.Base(m) // e.g. web.locale=en.json
		lang := strings.TrimSuffix(strings.TrimPrefix(base, "web.locale="), ".json")
		url := a.assetURL("/assets/" + m)
		if i := strings.Index(url, "?h="); i >= 0 {
			out[lang] = url[i+3:]
		}
	}
	return out
}

// AssetURL returns the cache-busting URL for an embedded asset path. Exposed so
// render cases can build conditional script tags (widget glue) with the same
// hashing as the core scripts.
func (a *app) AssetURL(path string) string {
	return a.assetURL(path)
}

func (a *app) UserCSSURLs() []string {
	return []string{a.assetURL("/assets/defaulttemplate.css")}
}

func (a *app) UserInlineCSS() string {
	b, err := fs.ReadFile(assets.FS, "defaulttemplate.css")
	if err != nil {
		a.log.Error("failed to read defaulttemplate.css", "error", err)
		return "/* failed to load CSS: " + err.Error() + " */"
	}
	return string(b)
}

func (a *app) AssetVersion() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}
