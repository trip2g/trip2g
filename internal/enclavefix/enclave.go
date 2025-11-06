// Package enclavefix is a temporary fork of goldmark-enclave with bug fixes
// TODO: Remove this package once https://github.com/quailyquaily/goldmark-enclave/issues/XX is fixed
package enclavefix

import (
	"github.com/quailyquaily/goldmark-enclave/core"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type (
	Option           func(*enclaveExtension)
	enclaveExtension struct {
		cfg *core.Config
	}
)

func NewEnclave(c *core.Enclave) *core.Enclave {
	c.Destination = c.Image.Destination
	c.Title = string(c.Image.Title)
	return c
}

func (e *enclaveExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&astTransformer{
				cfg: e.cfg,
			}, 500),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewHTMLRenderer(e.cfg), 500),
		),
	)
}

func New(cfg *core.Config) goldmark.Extender {
	e := &enclaveExtension{
		cfg: cfg,
	}
	return e
}
