package mdloader

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"trip2g/internal/image"
	"trip2g/internal/model"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/wikilink"
)

// based on https://raw.githubusercontent.com/abhinav/goldmark-wikilink/refs/heads/main/renderer.go

// linkRenderer renders wikilinks as HTML.
//
// Install it on your goldmark Markdown object with Extender, or directly on a
// goldmark linkRenderer by using the WithNodeRenderers option.
//
//	wikilinkRenderer := util.Prioritized(&wikilink.linkRenderer{...}, 199)
//	goldmarkRenderer.AddOptions(renderer.WithNodeRenderers(wikilinkRenderer))
type linkRenderer struct {
	// linkRenderer determines destinations for wikilink pages.
	//
	// If a resolver returns an empty destination, the Renderer will skip
	// the link and render just its contents. That is, instead of,
	//
	//   <a href="foo">bar</a>
	//
	// The renderer will render just the following.
	//
	//   bar
	//
	// Defaults to DefaultResolver if unspecified.
	resolver wikilink.Resolver

	once sync.Once // guards init

	// hasDest records whether a node had a destination when we resolved
	// it. This is needed to decide whether a closing </a> must be added
	// when exiting a Node render.
	hasDest sync.Map // *Node => struct{}

	nvs *model.NoteViews
}

func (r *linkRenderer) init() {
	r.once.Do(func() {
		if r.resolver == nil {
			r.resolver = wikilink.DefaultResolver
		}
	})
}

// RegisterFuncs registers wikilink rendering functions with the provided
// goldmark registerer. This teaches goldmark to call us when it encounters a
// wikilink in the AST.
func (r *linkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(wikilink.Kind, r.Render)
}

// Render renders the provided Node. It must be a Wikilink [Node].
//
// goldmark will call this method if this renderer was registered with it
// using the WithNodeRenderers option.
//
// All nodes will be rendered as links (with <a> tags),
// except for embed links (![[..]]) that refer to images.
// Those will be rendered as images (with <img> tags).
func (r *linkRenderer) Render(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	r.init()

	n, ok := node.(*wikilink.Node)
	if !ok {
		return ast.WalkStop, fmt.Errorf("unexpected node %T, expected *wikilink.Node", node)
	}

	if entering {
		return r.enter(w, n, src)
	}

	r.exit(w, n)
	return ast.WalkContinue, nil
}

func (r *linkRenderer) enter(w util.BufWriter, n *wikilink.Node, src []byte) (ast.WalkStatus, error) {
	dest, err := r.resolver.ResolveWikilink(n)
	if err != nil {
		return ast.WalkStop, fmt.Errorf("resolve %q: %w", n.Target, err)
	}
	if len(dest) == 0 {
		return ast.WalkContinue, nil
	}

	img := resolveAsImage(n)
	if !img {
		if n.Embed {
			return r.renderEmbed(w, n, dest)
		}

		r.hasDest.Store(n, struct{}{})
		_, _ = w.WriteString(`<a`)

		anchorFound := true
		lr, ok := r.resolver.(*myLinkResolver)
		if ok {
			_, anchorFound = resolveHeadingAnchor(lr.nvs, lr.currentPage, n)
		}

		note := r.resolveNoteForAttributes(string(dest))
		switch {
		case !anchorFound || (note == nil && len(n.Target) > 0):
			_, _ = w.WriteString(` class="wip"`)
		case note != nil:
			if !note.Free {
				subgraphClasses := ""

				if len(note.Subgraphs) == 0 {
					subgraphClasses = "paywall:core"
				} else {
					for _, subgraph := range note.SubgraphNames {
						subgraphClasses += " paywall-" + subgraph
					}
				}

				_, _ = w.WriteString(` class="paywall ` + subgraphClasses + `"`)
			}

			_, _ = fmt.Fprintf(w, ` data-pid="%d"`, note.PathID)
		}

		_, _ = w.WriteString(` href="`)
		_, _ = w.Write(safeURL(dest))
		_, _ = w.WriteString(`">`)
		return ast.WalkContinue, nil
	}

	// Check if it's a video file
	isVideo := image.IsVideoExtension(string(n.Target))

	if isVideo {
		// Render as <video> tag
		_, _ = w.WriteString(`<video controls src="`)
		_, _ = w.Write(safeURL(dest))
		_, _ = w.WriteString(`">`)
		_, _ = w.WriteString(`Your browser does not support the video tag.`)
		_, _ = w.WriteString(`</video>`)
		return ast.WalkSkipChildren, nil
	}

	// Check if it's an audio file
	if image.IsAudioExtension(string(n.Target)) {
		_, _ = w.WriteString(`<audio controls src="`)
		_, _ = w.Write(safeURL(dest))
		_, _ = w.WriteString(`">`)
		_, _ = w.WriteString(`</audio>`)
		return ast.WalkSkipChildren, nil
	}

	// Check if it's a document file — render as download link
	if image.IsDocExtension(string(n.Target)) {
		_, _ = w.WriteString(`<a href="`)
		_, _ = w.Write(safeURL(dest))
		_, _ = w.WriteString(`" class="file-link">`)
		_, _ = w.Write(util.EscapeHTML(n.Target))
		_, _ = w.WriteString(`</a>`)
		return ast.WalkSkipChildren, nil
	}

	// Render as <img> tag for images
	_, _ = w.WriteString(`<img src="`)
	_, _ = w.Write(safeURL(dest))

	// The label portion of the link becomes the alt text
	// only if it isn't the same as the target.
	// This way, [[foo.jpg]] does not become alt="foo.jpg",
	// but [[foo.jpg|bar]] does become alt="bar".
	// Also parse size from label: ![[foo.jpg|20x20]] or ![[foo.jpg|alt|20x20]]
	var size *imageSize
	if n.ChildCount() == 1 {
		label := nodeText(src, n.FirstChild())
		if !bytes.Equal(label, n.Target) {
			cleanAlt, parsedSize := parseImageSize(string(label))
			size = parsedSize
			if cleanAlt != "" {
				_, _ = w.WriteString(`" alt="`)
				_, _ = w.Write(util.EscapeHTML([]byte(cleanAlt)))
			}
		}
	}

	// Add size attributes if present
	if size != nil {
		_, _ = w.WriteString(`" width="`)
		_, _ = w.WriteString(size.Width)
		if size.Height != "" {
			_, _ = w.WriteString(`" height="`)
			_, _ = w.WriteString(size.Height)
		}
	}

	_, _ = w.WriteString(`">`)
	return ast.WalkSkipChildren, nil
}

// safeURL escapes dest for an attribute and blanks it when it carries an
// executable scheme (javascript:, vbscript:, file:, data: except images),
// matching goldmark's policy for ordinary markdown links. The check runs on
// the escaped form, after entity references like &#106; are resolved.
func safeURL(dest []byte) []byte {
	escaped := util.URLEscape(dest, true /* resolve references */)
	if html.IsDangerousURL(escaped) {
		return nil
	}
	return escaped
}

// removeVersion strips ?version=... from URL without full URL parsing.
func removeVersion(originalURL string) string {
	idx := strings.Index(originalURL, "?version=")
	if idx == -1 {
		return originalURL
	}
	return originalURL[:idx]
}

func (r *linkRenderer) renderEmbed(w util.BufWriter, n *wikilink.Node, dest []byte) (ast.WalkStatus, error) {
	url := removeVersion(string(dest))

	note := r.nvs.GetByPath(url)
	if note == nil {
		lr, ok := r.resolver.(*myLinkResolver)
		if ok && lr.domainRenderNotes != nil {
			note = lr.domainRenderNotes[url]
		}
	}
	if note == nil {
		lr, ok := r.resolver.(*myLinkResolver)
		if ok {
			lr.currentPage.AddWarning(model.NoteWarningInfo, "embedded note not found: %s", url)
		}

		return ast.WalkSkipChildren, nil
	}

	lr, isLoadResolver := r.resolver.(*myLinkResolver)
	if !isLoadResolver || !embedVisibleToParentReaders(lr.currentPage, note, lr.renderingPreview) {
		_, _ = w.WriteString(`<a href="`)
		_, _ = w.Write(safeURL(dest))
		_, _ = w.WriteString(`">`)
		_, _ = w.Write(util.EscapeHTML(n.Target))
		_, _ = w.WriteString(`</a>`)
		return ast.WalkSkipChildren, nil
	}

	if lr.embedRendering[note] {
		// The free-preview pass renders the same embed again; warn once.
		msg := "embed cycle: " + url
		if !hasWarningMessage(lr.currentPage, msg) {
			lr.currentPage.AddWarning(model.NoteWarningWarning, "%s", msg)
		}
		return ast.WalkSkipChildren, nil
	}

	class := "embedded-note"
	if note.EmbededClass != "" {
		class += " embedded-note__" + note.EmbededClass
	}

	_, err := w.WriteString(`<div class="` + class + `">` + string(note.HTML) + `</div>`)
	if err != nil {
		return ast.WalkStop, fmt.Errorf("failed to write embedded note HTML: %w", err)
	}

	return ast.WalkSkipChildren, nil
}

// embedVisibleToParentReaders reports whether everyone who can read parent can
// also read target, so inlining target's HTML into parent's discloses nothing.
// The free preview of a note is shown to anonymous readers, so there only a
// target that is itself anonymously readable may be inlined.
func embedVisibleToParentReaders(parent, target *model.NoteView, preview bool) bool {
	if target.IsAnonymouslyReadable() || isUngatedPartial(target) {
		return true
	}

	if parent == nil || preview || parent.IsAnonymouslyReadable() {
		return false
	}

	if strings.HasSuffix(target.Path, ".html") {
		return false
	}

	if target.RequiresSignin() {
		return true
	}

	if parent.RequiresSignin() {
		return false
	}

	if len(target.SubgraphNames) == 0 {
		return true
	}

	if len(parent.SubgraphNames) == 0 {
		return false
	}

	for _, name := range parent.SubgraphNames {
		if !slices.Contains(target.SubgraphNames, name) {
			return false
		}
	}

	return true
}

// isUngatedPartial reports whether n is a system note (_name.md) without a
// subgraph gate: such notes are never served as pages, and exist to be
// embedded into others.
func isUngatedPartial(n *model.NoteView) bool {
	return n.IsSystem() && len(n.SubgraphNames) == 0 && !n.RequiresSignin() && !strings.HasSuffix(n.Path, ".html")
}

func (r *linkRenderer) exit(w util.BufWriter, n *wikilink.Node) {
	if _, ok := r.hasDest.LoadAndDelete(n); ok {
		_, _ = w.WriteString("</a>")
	}
}

// resolveNoteForAttributes finds the target NoteView for writing class/data-pid attributes.
// During domain re-render, dest may be a domain-specific path (e.g., "/page-b") or a
// full cross-domain URL (e.g., "https://bar.com/path") that doesn't exist in nv.Map.
// Falls back to domainRenderNotes on the resolver when GetByPath returns nil.
func (r *linkRenderer) resolveNoteForAttributes(dest string) *model.NoteView {
	path := removeVersion(dest)
	// Strip anchor fragment — the map is keyed by path only, not path#anchor.
	if idx := strings.Index(path, "#"); idx != -1 {
		path = path[:idx]
	}
	note := r.nvs.GetByPath(path)
	if note == nil {
		lr, ok := r.resolver.(*myLinkResolver)
		if ok && lr.domainRenderNotes != nil {
			note = lr.domainRenderNotes[path]
		}
	}
	return note
}

// returns true if the wikilink should be resolved to an image node.
func resolveAsImage(n *wikilink.Node) bool {
	if !n.Embed {
		return false
	}

	return image.IsMediaExtension(string(n.Target))
}

func nodeText(src []byte, n ast.Node) []byte {
	var buf bytes.Buffer
	writeNodeText(src, &buf, n)
	return buf.Bytes()
}

func writeNodeText(src []byte, dst io.Writer, n ast.Node) {
	switch n := n.(type) {
	case *ast.Text:
		_, _ = dst.Write(n.Segment.Value(src))
	case *ast.String:
		_, _ = dst.Write(n.Value)
	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeNodeText(src, dst, c)
		}
	}
}

func hasWarningMessage(note *model.NoteView, msg string) bool {
	for _, w := range note.Warnings {
		if w.Message == msg {
			return true
		}
	}
	return false
}
