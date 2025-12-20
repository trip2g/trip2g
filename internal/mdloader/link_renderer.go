package mdloader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync"
	"trip2g/internal/image"
	"trip2g/internal/model"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
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
			return r.renderEmbed(w, dest)
		}

		r.hasDest.Store(n, struct{}{})
		_, _ = w.WriteString(`<a`)

		note := r.nvs.GetByPath(removeVersion(string(dest)))
		if note != nil {
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
		} else {
			_, _ = w.WriteString(` class="wip"`)
		}

		_, _ = w.WriteString(` href="`)
		_, _ = w.Write(util.URLEscape(dest, true /* resolve references */))
		_, _ = w.WriteString(`">`)
		return ast.WalkContinue, nil
	}

	// Check if it's a video file
	isVideo := image.IsVideoExtension(string(n.Target))

	if isVideo {
		// Render as <video> tag
		_, _ = w.WriteString(`<video controls src="`)
		_, _ = w.Write(util.URLEscape(dest, true /* resolve references */))
		_, _ = w.WriteString(`">`)
		_, _ = w.WriteString(`Your browser does not support the video tag.`)
		_, _ = w.WriteString(`</video>`)
		return ast.WalkSkipChildren, nil
	}

	// Render as <img> tag for images
	_, _ = w.WriteString(`<img src="`)
	_, _ = w.Write(util.URLEscape(dest, true /* resolve references */))

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

// TODO: find a better way to handle this.
func removeVersion(originalURL string) string {
	u, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}

	q := u.Query()
	q.Del("version")
	u.RawQuery = q.Encode()

	return u.String()
}

var errNoHTML = errors.New("note has no HTML content")

func (r *linkRenderer) renderEmbed(w util.BufWriter, dest []byte) (ast.WalkStatus, error) {
	url := removeVersion(string(dest))

	note := r.nvs.GetByPath(url)
	if note == nil {
		lr, ok := r.resolver.(*myLinkResolver)
		if ok {
			lr.currentPage.AddWarning(model.NoteWarningInfo, "embedded note not found: %s", url)
		}

		return ast.WalkSkipChildren, nil
	}

	if len(note.HTML) == 0 {
		// mdloader will try to render it again later
		return ast.WalkSkipChildren, errNoHTML
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

func (r *linkRenderer) exit(w util.BufWriter, n *wikilink.Node) {
	if _, ok := r.hasDest.LoadAndDelete(n); ok {
		_, _ = w.WriteString("</a>")
	}
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
