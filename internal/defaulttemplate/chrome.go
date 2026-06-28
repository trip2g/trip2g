package defaulttemplate

// Header resolves the page header note (per the same precedence as NotePage:
// per-note `header:` frontmatter, glob-matched layout section, `_header`
// fallback) and returns its rendered site-header HTML, or "" when the page has
// no header. Exposed so custom Jet layouts can embed the standard chrome via
// {{ defaultTemplate.Header() }} without re-implementing resolution.
func Header(ctx *Ctx) string {
	ref := ctx.HeaderRef()
	if ref.Kind == ContentRefNone {
		return ""
	}
	note := ctx.resolveNoteRef(ref)
	if note == nil {
		return ""
	}
	return SiteHeader(ctx, note, false, false)
}

// Footer resolves the page footer note and returns its rendered site-footer
// HTML, or "" when the page has no footer.
func Footer(ctx *Ctx) string {
	ref := ctx.FooterRef()
	if ref.Kind == ContentRefNone {
		return ""
	}
	note := ctx.resolveNoteRef(ref)
	if note == nil {
		return ""
	}
	return SiteFooter(ctx, note)
}
