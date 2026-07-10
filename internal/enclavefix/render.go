package enclavefix

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/quailyquaily/goldmark-enclave/core"
	"github.com/quailyquaily/goldmark-enclave/object"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var (
	youtubeIDPattern         = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	bilibiliIDPattern        = regexp.MustCompile(`^(?:BV[A-Za-z0-9]{10}|av[0-9]+)$`)
	imageSizePattern         = regexp.MustCompile(`^[0-9]+(?:%|px|rem)?$`)
	unitlessImageSizePattern = regexp.MustCompile(`^[0-9]+$`)
	difyPathPattern          = regexp.MustCompile(`^/chatbot/[A-Za-z0-9_-]+/?$`)
)

// ValidEmbedID reports whether an embed identifier is safe for the provider's
// downstream HTML/JavaScript template. It is shared with mdloader, which has a
// second renderer for the same AST nodes.
func ValidEmbedID(provider, id string) bool {
	switch provider {
	case "youtube":
		return youtubeIDPattern.MatchString(id)
	case "bilibili":
		return bilibiliIDPattern.MatchString(id)
	case "tradingview":
		if id == "" || strings.ContainsAny(id, `"\<>`) {
			return false
		}
		for _, r := range id {
			if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// ValidDifyURL accepts only the HTTPS chatbot URLs emitted by the transformer.
// In particular, it rejects decoded quotes from dify: URLs before they reach
// the upstream text/template iframe renderer.
func ValidDifyURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "udify.app" &&
		u.User == nil && u.RawQuery == "" && u.Fragment == "" && difyPathPattern.MatchString(u.Path)
}

// SafeImageDimension returns a dimension only when it is safe to place in an
// HTML attribute or inline style. Invalid values are dropped.
func SafeImageDimension(value string) string {
	if imageSizePattern.MatchString(value) {
		return value
	}
	return ""
}

// ValidQuailLayout constrains the only query parameter that the upstream
// renderer interpolates into an iframe URL using text/template.
func ValidQuailLayout(layout string) bool {
	return layout == "" || layout == "subscribe_form" || layout == "subscribe_form_mini"
}

type HTMLRenderer struct {
	cfg *core.Config
}

func NewHTMLRenderer(cfg *core.Config) renderer.NodeRenderer {
	r := &HTMLRenderer{cfg: cfg}
	return r
}

func (r *HTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// image with alt like [alt](url "title") will generate a node seq like
	// layout:
	// - imgLeftNode: kind = paragraph, content = alt
	// - imgNode: kind = image
	// - imgRightNode: kind = text, content = alt
	// I don't know how to handle them yet.
	reg.Register(core.KindEnclave, r.renderEnclave)
}

func (r *HTMLRenderer) renderEnclave(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// check the node and print the inner html and children
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Kind() == ast.KindText {
				node.RemoveChildren(node)
			}
		}
		return ast.WalkContinue, nil
	}

	enc := node.(*core.Enclave)
	switch enc.Provider {
	case core.EnclaveProviderYouTube:
		{
			if !ValidEmbedID("youtube", enc.ObjectID) {
				w.Write([]byte(r.wrapEnclaveErrorHtml("youtube", enc.ObjectID)))
				break
			}
			html, err := object.GetYoutubeEmbedHtml(enc)
			if err != nil || html == "" {
				html = r.wrapEnclaveErrorHtml("youtube", enc.ObjectID)
			} else {
				html = r.wrapEnclaveHtml("youtube", html, false, false)
			}
			w.Write([]byte(html))
		}

	case core.EnclaveProviderBilibili:
		{
			if !ValidEmbedID("bilibili", enc.ObjectID) {
				w.Write([]byte(r.wrapEnclaveErrorHtml("bilibili", enc.ObjectID)))
				break
			}
			html, err := object.GetBilibiliEmbedHtml(enc)
			if err != nil || html == "" {
				html = r.wrapEnclaveErrorHtml("bilibili", enc.ObjectID)
			} else {
				html = r.wrapEnclaveHtml("bilibili", html, false, false)
			}
			w.Write([]byte(html))
		}

	case core.EnclaveProviderTwitter:
		html, err := object.GetTweetOembedHtml(enc.ObjectID, enc.Theme)
		if err != nil || html == "" {
			// html = fmt.Sprintf(`<div class="enclave-object-wrapper normal-wrapper"><div class="enclave-object twitter-enclave-object normal-object error">Failed to load tweet from %s</div></div>`, enc.ObjectID)
			html = r.wrapEnclaveErrorHtml("twitter", enc.ObjectID)
		} else {
			// html = fmt.Sprintf(`<div class="enclave-object-wrapper normal-wrapper"><div class="enclave-object twitter-enclave-object normal-object no-border">%s</div></div>`, html)
			html = r.wrapEnclaveHtml("twitter", html, true, false)
		}
		w.Write([]byte(html))

	case core.EnclaveProviderTradingView:
		var html string
		var err error
		if ValidEmbedID("tradingview", enc.ObjectID) {
			html, err = object.GetTradingViewWidgetHtml(enc)
		} else {
			err = fmt.Errorf("invalid TradingView symbol")
		}
		if err != nil || html == "" {
			// html = fmt.Sprintf(`<div class="enclave-object-wrapper normal-wrapper"><div class="enclave-object tradingview-enclave-object error">Failed to load tradingview chart from %s</div></div>`, enc.ObjectID)
			html = r.wrapEnclaveErrorHtml("tradingview", enc.ObjectID)
		} else {
			// html = fmt.Sprintf(`<div class="enclave-object-wrapper auto-resize"><div class="enclave-object tradingview-enclave-object no-border">%s</div></div>`, html)
			html = r.wrapEnclaveHtml("tradingview", html, false, false)
		}
		w.Write([]byte(html))

	case core.EnclaveProviderDifyWidget:
		var html string
		var err error
		if ValidDifyURL(enc.ObjectID) {
			html, err = object.GetDifyWidgetHtml(enc)
		} else {
			err = fmt.Errorf("invalid Dify chatbot URL")
		}
		if err != nil || html == "" {
			// html = fmt.Sprintf(`<div class="enclave-object-wrapper normal-wrapper"><div class="enclave-object dify-enclave-object error">Failed to load dify widget from %s</div></div>`, enc.ObjectID)
			html = r.wrapEnclaveErrorHtml("dify", enc.ObjectID)
		} else {
			// html = fmt.Sprintf(`<div class="enclave-object-wrapper normal-wrapper"><div class="enclave-object dify-enclave-object normal-object no-border">%s</div></div>`, html)
			html = r.wrapEnclaveHtml("dify", html, true, false)
		}
		w.Write([]byte(html))

	case core.EnclaveProviderQuailWidget:
		var html string
		var err error
		if ValidQuailLayout(enc.Params["layout"]) {
			html, err = object.GetQuailWidgetHtml(enc)
		} else {
			err = fmt.Errorf("invalid Quail layout")
		}
		if err != nil || html == "" {
			// html = fmt.Sprintf(`<div class="enclave-object-wrapper normal-wrapper"><div class="enclave-object quail-enclave-object error">Failed to load quail widget from %s</div></div>`, enc.ObjectID)
			html = r.wrapEnclaveErrorHtml("quail", enc.ObjectID)
		} else {
			// html = fmt.Sprintf(`<div class="enclave-object-wrapper normal-wrapper"><div class="enclave-object quail-enclave-object normal-object no-border">%s</div></div>`, html)
			html = r.wrapEnclaveHtml("quail", html, true, false)
		}
		w.Write([]byte(html))

	case core.EnclaveProviderQuailAd:
		html, err := object.GetQuailAdHtml(enc)
		if err != nil || html == "" {
			html = r.wrapEnclaveErrorHtml("quail-ad", enc.ObjectID)
		}
		w.Write([]byte(html))

	case core.EnclaveProviderSpotify:
		html, err := object.GetSpotifyWidgetHtml(enc)
		if err != nil || html == "" {
			// html = fmt.Sprintf(`<div class="enclave-object-wrapper normal-wrapper"><div class="enclave-object spotify-enclave-object error">Failed to load spotify widget from %s</div></div>`, enc.ObjectID)
			html = r.wrapEnclaveErrorHtml("spotify", enc.ObjectID)
		} else {
			// html = fmt.Sprintf(`<div class="enclave-object-wrapper normal-wrapper"><div class="enclave-object spotify-enclave-object normal-object no-border">%s</div></div>`, html)
			html = r.wrapEnclaveHtml("spotify", html, true, false)
		}
		w.Write([]byte(html))

	case core.EnclaveProviderPodbean:
		html, err := object.GetPodbeanHtml(enc)
		if err != nil || html == "" {
			html = r.wrapEnclaveErrorHtml("podbean", enc.ObjectID)
		} else {
			html = r.wrapEnclaveHtml("podbean", html, true, false)
		}
		w.Write([]byte(html))

	case core.EnclaveHtml5Audio:
		audioHTML := fmt.Sprintf(`<audio controls src="%s"></audio>`, html.EscapeString(enc.ObjectID))
		w.Write([]byte(r.wrapEnclaveHtml("audio", audioHTML, true, false)))

	case core.EnclaveProviderQuailImage:
		alt := enc.Alt
		if enc.Alt == "" && len(enc.Title) != 0 {
			alt = fmt.Sprintf("An image to describe %s", enc.Title)
		}
		if alt == "" {
			alt = "An image to describe post"
		}
		w.Write([]byte(renderQuailImageHTML(enc, alt)))

	case core.EnclaveRegularImage:
		alt := enc.Alt
		if enc.Alt == "" && len(enc.Title) != 0 {
			alt = fmt.Sprintf("An image to describe %s", enc.Title)
		}
		if alt == "" {
			alt = "An image to describe post"
		}
		html := fmt.Sprintf(`<img src="%s" alt="%s" />`, html.EscapeString(enc.URL.String()), html.EscapeString(alt))
		w.Write([]byte(html))

	}

	return ast.WalkContinue, nil
}

func renderQuailImageHTML(enc *core.Enclave, alt string) string {
	width := styleImageDimension(enc.Params["width"])
	if width == "" {
		width = "auto"
	}
	height := styleImageDimension(enc.Params["height"])
	if height == "" {
		height = "auto"
	}
	margin := "0 auto"
	if enc.Params["align"] == "left" {
		margin = "0 auto 0 0"
	} else if enc.Params["align"] == "right" {
		margin = "0 0 0 auto"
	}
	return fmt.Sprintf(
		`<figure class="quail-image-wrapper" style="width: %s; height: %s; margin: %s; display: block"><img src="%s" alt="%s" style="width: 100%%; height: auto" class="quail-image" /><figcaption class="quail-image-caption" style="display: block">%s</figcaption></figure>`,
		width,
		height,
		margin,
		html.EscapeString(enc.URL.String()),
		html.EscapeString(alt),
		html.EscapeString(enc.Title),
	)
}

func styleImageDimension(value string) string {
	value = SafeImageDimension(value)
	if unitlessImageSizePattern.MatchString(value) {
		return value + "px"
	}
	return value
}

func (r *HTMLRenderer) wrapEnclaveErrorHtml(enclaveName, objectID string) string {
	html := fmt.Sprintf(
		`<div class="enclave-object-wrapper normal-wrapper"><div class="enclave-object %s-enclave-object error">Failed to load %s from %s</div></div>`,
		html.EscapeString(enclaveName), html.EscapeString(enclaveName), html.EscapeString(objectID),
	)
	return html
}

func (r *HTMLRenderer) wrapEnclaveHtml(enclaveName, html string, isNormal, hasBorder bool) string {
	normalCls := ""
	borderCls := ""
	autoResizeCls := "normal-wrapper"
	if isNormal {
		normalCls = "normal-object"
	} else {
		autoResizeCls = "auto-resize"
	}
	if !hasBorder {
		borderCls = "no-border"
	}

	ret := fmt.Sprintf(
		`<div class="enclave-object-wrapper %s"><div class="enclave-object %s-enclave-object %s %s">%s</div></div>`,
		autoResizeCls, enclaveName, normalCls, borderCls, html,
	)
	return ret
}
