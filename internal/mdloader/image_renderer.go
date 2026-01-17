package mdloader

import (
	"fmt"
	"regexp"
	"strings"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/quailyquaily/goldmark-enclave/object"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// imageSizeRegex matches size specifications like "20x20" or "100".
var imageSizeRegex = regexp.MustCompile(`^(\d+)(?:x(\d+))?$`)

// imageSize represents parsed image dimensions.
type imageSize struct {
	Width  string
	Height string
}

// parseImageSize extracts size specification from the end of alt text.
// Supports formats: "alt|20x20", "alt|100", "|20x20", "20x20".
// Returns the clean alt text and size if found.
func parseImageSize(alt string) (string, *imageSize) {
	if alt == "" {
		return "", nil
	}

	// Check if alt contains a pipe - size should be after the last pipe
	lastPipe := strings.LastIndex(alt, "|")
	if lastPipe == -1 {
		// No pipe, check if the entire alt is a size spec
		if match := imageSizeRegex.FindStringSubmatch(alt); match != nil {
			return "", &imageSize{Width: match[1], Height: match[2]}
		}
		return alt, nil
	}

	// Extract potential size part after the last pipe
	beforePipe := alt[:lastPipe]
	afterPipe := strings.TrimSpace(alt[lastPipe+1:])

	// Check if the part after pipe is a valid size
	match := imageSizeRegex.FindStringSubmatch(afterPipe)
	if match == nil {
		// Not a size specification, return original alt
		return alt, nil
	}

	// Valid size found
	return strings.TrimSpace(beforePipe), &imageSize{Width: match[1], Height: match[2]}
}

// imageRenderer renders Enclave image nodes with AssetReplaces URL substitution.
type imageRenderer struct {
	resolver *myLinkResolver
}

func newImageRenderer(resolver *myLinkResolver) *imageRenderer {
	return &imageRenderer{
		resolver: resolver,
	}
}

func (r *imageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(enclavecore.KindEnclave, r.renderEnclave)
}

func (r *imageRenderer) renderEnclave(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Remove text children (cleanup from enclave transformer)
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Kind() == ast.KindText {
				node.RemoveChildren(node)
			}
		}
		return ast.WalkContinue, nil
	}

	enc, ok := node.(*enclavecore.Enclave)
	if !ok {
		return ast.WalkContinue, nil
	}

	// Handle different providers
	switch enc.Provider {
	case enclavecore.EnclaveRegularImage, enclavecore.EnclaveProviderQuailImage:
		r.renderImage(w, enc)

	case enclavecore.EnclaveProviderYouTube:
		html, err := object.GetYoutubeEmbedHtml(enc)
		if err != nil || html == "" {
			html = wrapEnclaveErrorHtml("youtube", enc.ObjectID)
		} else {
			html = wrapEnclaveHtml("youtube", html, false, false)
		}
		_, _ = w.Write([]byte(html))

	case enclavecore.EnclaveProviderBilibili:
		html, err := object.GetBilibiliEmbedHtml(enc)
		if err != nil || html == "" {
			html = wrapEnclaveErrorHtml("bilibili", enc.ObjectID)
		} else {
			html = wrapEnclaveHtml("bilibili", html, false, false)
		}
		_, _ = w.Write([]byte(html))

	case enclavecore.EnclaveProviderTwitter:
		html, err := object.GetTweetOembedHtml(enc.ObjectID, enc.Theme)
		if err != nil || html == "" {
			html = wrapEnclaveErrorHtml("twitter", enc.ObjectID)
		} else {
			html = wrapEnclaveHtml("twitter", html, true, false)
		}
		_, _ = w.Write([]byte(html))

	case enclavecore.EnclaveProviderTradingView:
		html, err := object.GetTradingViewWidgetHtml(enc)
		if err != nil || html == "" {
			html = wrapEnclaveErrorHtml("tradingview", enc.ObjectID)
		} else {
			html = wrapEnclaveHtml("tradingview", html, false, false)
		}
		_, _ = w.Write([]byte(html))

	case enclavecore.EnclaveProviderDifyWidget:
		html, err := object.GetDifyWidgetHtml(enc)
		if err != nil || html == "" {
			html = wrapEnclaveErrorHtml("dify", enc.ObjectID)
		} else {
			html = wrapEnclaveHtml("dify", html, true, false)
		}
		_, _ = w.Write([]byte(html))

	case enclavecore.EnclaveProviderQuailWidget:
		html, err := object.GetQuailWidgetHtml(enc)
		if err != nil || html == "" {
			html = wrapEnclaveErrorHtml("quail", enc.ObjectID)
		} else {
			html = wrapEnclaveHtml("quail", html, true, false)
		}
		_, _ = w.Write([]byte(html))

	case enclavecore.EnclaveProviderQuailAd:
		html, err := object.GetQuailAdHtml(enc)
		if err != nil || html == "" {
			html = wrapEnclaveErrorHtml("quail-ad", enc.ObjectID)
		}
		_, _ = w.Write([]byte(html))

	case enclavecore.EnclaveProviderSpotify:
		html, err := object.GetSpotifyWidgetHtml(enc)
		if err != nil || html == "" {
			html = wrapEnclaveErrorHtml("spotify", enc.ObjectID)
		} else {
			html = wrapEnclaveHtml("spotify", html, true, false)
		}
		_, _ = w.Write([]byte(html))

	case enclavecore.EnclaveProviderPodbean:
		html, err := object.GetPodbeanHtml(enc)
		if err != nil || html == "" {
			html = wrapEnclaveErrorHtml("podbean", enc.ObjectID)
		} else {
			html = wrapEnclaveHtml("podbean", html, true, false)
		}
		_, _ = w.Write([]byte(html))

	case enclavecore.EnclaveHtml5Audio:
		html, err := object.GetAudioHtml(enc)
		if err != nil || html == "" {
			html = wrapEnclaveErrorHtml("audio", enc.ObjectID)
		} else {
			html = wrapEnclaveHtml("audio", html, true, false)
		}
		_, _ = w.Write([]byte(html))
	}

	return ast.WalkContinue, nil
}

// renderImage renders regular images and quail images with asset replacement
func (r *imageRenderer) renderImage(w util.BufWriter, enc *enclavecore.Enclave) {
	// Get the original URL - ObjectID contains the clean URL for QuailImage
	originalURL := enc.URL.String()
	if enc.Provider == enclavecore.EnclaveProviderQuailImage && enc.ObjectID != "" {
		originalURL = enc.ObjectID
	}

	// Try to resolve from AssetReplaces
	resolvedURL := originalURL
	if r.resolver != nil && r.resolver.currentPage != nil {
		assetReplace, found := r.resolver.currentPage.AssetReplaces[originalURL]
		if found && assetReplace != nil {
			resolvedURL = assetReplace.URL
		}
	}

	// Get size - first check Params (set by enclave transformer), then parse from alt
	var size *imageSize
	if enc.Provider == enclavecore.EnclaveProviderQuailImage {
		// Size is in Params for QuailImage
		width := enc.Params["width"]
		height := enc.Params["height"]
		if width != "" {
			size = &imageSize{Width: width, Height: height}
		}
	}

	// Parse size from alt text (for RegularImage or fallback)
	alt := enc.Alt
	cleanAlt, altSize := parseImageSize(alt)
	if size == nil && altSize != nil {
		size = altSize
	}
	if cleanAlt == "" {
		cleanAlt = alt
	}

	// Build alt text fallback
	if cleanAlt == "" && len(enc.Title) != 0 {
		cleanAlt = fmt.Sprintf("An image to describe %s", enc.Title)
	}
	if cleanAlt == "" {
		cleanAlt = "An image to describe post"
	}

	// Build HTML with optional size attributes
	var html string
	if size != nil {
		if size.Height != "" {
			html = fmt.Sprintf(`<img src="%s" alt="%s" width="%s" height="%s" />`, resolvedURL, cleanAlt, size.Width, size.Height)
		} else {
			html = fmt.Sprintf(`<img src="%s" alt="%s" width="%s" />`, resolvedURL, cleanAlt, size.Width)
		}
	} else {
		html = fmt.Sprintf(`<img src="%s" alt="%s" />`, resolvedURL, cleanAlt)
	}
	_, _ = w.Write([]byte(html))
}

// wrapEnclaveErrorHtml wraps error message in enclave error HTML
func wrapEnclaveErrorHtml(enclaveName, objectID string) string {
	return fmt.Sprintf(
		`<div class="enclave-object-wrapper normal-wrapper"><div class="enclave-object %s-enclave-object error">Failed to load %s from %s</div></div>`,
		enclaveName, enclaveName, objectID,
	)
}

// wrapEnclaveHtml wraps content in enclave HTML wrapper
func wrapEnclaveHtml(enclaveName, html string, isNormal, hasBorder bool) string {
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

	return fmt.Sprintf(
		`<div class="enclave-object-wrapper %s"><div class="enclave-object %s-enclave-object %s %s">%s</div></div>`,
		autoResizeCls, enclaveName, normalCls, borderCls, html,
	)
}
