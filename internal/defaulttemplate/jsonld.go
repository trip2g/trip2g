package defaulttemplate

import (
	"strings"
	"time"

	"trip2g/internal/templateviews"
)

// JSON-LD data layer. These methods make the structured-data DECISIONS in Go;
// the actual JSON is assembled in jsonld.html using quicktemplate's {%q= %}
// (JSON-quoted, XSS-safe against </script>) — far cheaper than json.Marshal on
// a per-page hot path.

// JSONLDCrumb is one entry of the BreadcrumbList.
type JSONLDCrumb struct {
	Name string
	Item string // absolute URL
}

// ShouldEmitJSONLD reports whether the current page should carry JSON-LD.
// Non-article/non-indexable pages (paywall, sign-in, onboarding, 404,
// unsupported file, or any noindex page) are excluded — there is no article to
// describe and it would mislead crawlers.
func (ctx *Ctx) ShouldEmitJSONLD() bool {
	if ctx.Note == nil {
		return false
	}
	if ctx.PaywallError != nil || ctx.SigninWallError != nil {
		return false
	}
	if ctx.OnboardingMode || ctx.NotFoundMode || ctx.UnsupportedFileExt != "" {
		return false
	}
	if strings.Contains(ctx.MetaRobots, "noindex") {
		return false
	}
	return true
}

// JSONLDType returns the schema.org @type for the page node.
// Priority: explicit schema_type override, profile/person, home page, else BlogPosting.
func (ctx *Ctx) JSONLDType() string {
	if ctx.Note == nil {
		return "WebPage"
	}
	m := ctx.Note.M()
	if st := strings.TrimSpace(m.GetString("schema_type", "")); st != "" {
		return st
	}
	switch strings.ToLower(strings.TrimSpace(m.GetString("type", ""))) {
	case "profile", "person":
		return "ProfilePage"
	}
	if ctx.Note.IsHomePage() {
		return "WebPage"
	}
	return "BlogPosting"
}

// JSONLDPageURL returns the canonical page URL (same value as og:url and the
// canonical link), or "" if unavailable.
func (ctx *Ctx) JSONLDPageURL() string {
	if ctx.OGTags == nil {
		return ""
	}
	return ctx.OGTags["og:url"]
}

// JSONLDImage returns the explicit og_image, else the first body image, else "".
func (ctx *Ctx) JSONLDImage() string {
	if ctx.Note == nil {
		return ""
	}
	if img := ctx.Note.OGImageURL(); img != "" {
		return img
	}
	return ctx.Note.FirstImageURL()
}

// JSONLDPublished returns datePublished in RFC3339, or "" if unset.
func (ctx *Ctx) JSONLDPublished() string {
	if ctx.Note == nil {
		return ""
	}
	t := ctx.Note.CreatedAt()
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// JSONLDModified returns dateModified in RFC3339, or "" if unset.
func (ctx *Ctx) JSONLDModified() string {
	if ctx.Note == nil {
		return ""
	}
	t := ctx.Note.UpdatedAt()
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// JSONLDSiteURL returns the site base URL with a trailing slash, or "".
func (ctx *Ctx) JSONLDSiteURL() string {
	if ctx.PublicURL == "" {
		return ""
	}
	return strings.TrimRight(ctx.PublicURL, "/") + "/"
}

// JSONLDLogo returns the site logo URL — the first image of the header note
// (_header.md or a glob-matched header layout section) — or "" if none.
func (ctx *Ctx) JSONLDLogo() string {
	if ctx.Notes == nil {
		return ""
	}
	ref := ctx.HeaderRef()
	var note *templateviews.Note
	switch ref.Kind {
	case ContentRefFile:
		note = ctx.Notes.ByPath(ref.Value)
	case ContentRefWikiLink:
		note = ctx.Notes.ByWikilink(ref.Value)
	case ContentRefSelfContent, ContentRefMagazine, ContentRefNone, ContentRefSimilar, ContentRefInLinks, ContentRefOutLinks, ContentRefTOC:
		// no header note to resolve — note stays nil
	}
	if note == nil {
		return ""
	}
	return note.FirstImageURL()
}

// JSONLDBreadcrumb builds the breadcrumb trail (Home → …segments) from the
// note's permalink. Returns nil for the home page / single-level pages, where a
// breadcrumb adds nothing.
func (ctx *Ctx) JSONLDBreadcrumb() []JSONLDCrumb {
	if ctx.Note == nil || ctx.PublicURL == "" {
		return nil
	}
	base := strings.TrimRight(ctx.PublicURL, "/")

	crumbs := []JSONLDCrumb{{Name: "Home", Item: base + "/"}}

	cum := ""
	for _, seg := range strings.Split(strings.Trim(ctx.Note.Permalink(), "/"), "/") {
		if seg == "" {
			continue
		}
		cum += "/" + seg
		name := humanizeSegment(seg)
		if ctx.Notes != nil {
			if n := ctx.Notes.ByPermalink(cum); n != nil && n.Title() != "" {
				name = n.Title()
			}
		}
		crumbs = append(crumbs, JSONLDCrumb{Name: name, Item: base + cum})
	}

	if len(crumbs) <= 1 {
		return nil
	}
	return crumbs
}

// humanizeSegment turns a URL slug segment into a readable label.
func humanizeSegment(seg string) string {
	s := strings.ReplaceAll(seg, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	return strings.TrimSpace(s)
}

// DeriveSiteName extracts a site name from the site_title_template (e.g.
// "My Blog" from "%s | My Blog"); falls back to the public URL host.
func DeriveSiteName(titleTemplate, publicURL string) string {
	name := strings.ReplaceAll(titleTemplate, "%s", "")
	name = strings.Trim(name, " |-—–·•:")
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return hostFromURL(publicURL)
}

func hostFromURL(u string) string {
	s := u
	if i := strings.Index(s, "://"); i != -1 {
		s = s[i+3:]
	}
	if i := strings.IndexByte(s, '/'); i != -1 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
