package mdloader

import (
	"fmt"

	enclavecore "github.com/quailyquaily/goldmark-enclave/core"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

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
	if !entering {
		return ast.WalkContinue, nil
	}

	enc, ok := node.(*enclavecore.Enclave)
	if !ok {
		return ast.WalkContinue, nil
	}

	// Only handle regular images
	if enc.Provider != enclavecore.EnclaveRegularImage {
		// Let the default enclave renderer handle other providers
		return ast.WalkContinue, nil
	}

	// Get the original URL
	originalURL := enc.URL.String()

	// Try to resolve from AssetReplaces
	resolvedURL := originalURL
	if r.resolver != nil && r.resolver.currentPage != nil {
		assetReplace, found := r.resolver.currentPage.AssetReplaces[originalURL]
		if found && assetReplace != nil {
			resolvedURL = assetReplace.URL
		}
	}

	// Build alt text
	var alt string
	if enc.Alt == "" && len(enc.Title) != 0 {
		alt = fmt.Sprintf("An image to describe %s", enc.Title)
	}
	if alt == "" {
		alt = "An image to describe post"
	}

	html := fmt.Sprintf(`<img src="%s" alt="%s" />`, resolvedURL, alt)
	_, _ = w.Write([]byte(html))

	return ast.WalkSkipChildren, nil
}
