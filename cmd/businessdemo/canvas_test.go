package main

import (
	"testing"
)

// extractFirstImage is the unit we're testing.
// It returns the first image reference for the note, plus the body with
// frontmatter and the consumed image embed stripped out. The first image is:
//   - the `image:` value from YAML frontmatter, if present
//   - else the first ![[…]] wikilink embed whose target has a media extension
//   - else "" with the body just frontmatter-stripped
//
// This drives whether the bot calls sendPhoto vs sendMessage.

func TestExtractFirstImage_FromFrontmatter(t *testing.T) {
	in := "---\ntitle: Pricing\nimage: cover.png\n---\n\n# Pricing\n\nFree and Pro plans."
	img, body := extractFirstImage(in)
	if img != "cover.png" {
		t.Errorf("image: want %q, got %q", "cover.png", img)
	}
	wantBody := "# Pricing\n\nFree and Pro plans."
	if body != wantBody {
		t.Errorf("body:\nwant %q\ngot  %q", wantBody, body)
	}
}

func TestExtractFirstImage_FromEmbed(t *testing.T) {
	in := "---\ntitle: Services\n---\n\nFlagship offering.\n\n![[diagram.jpg]]\n\nDetails below."
	img, body := extractFirstImage(in)
	if img != "diagram.jpg" {
		t.Errorf("image: want %q, got %q", "diagram.jpg", img)
	}
	if containsString(body, "diagram.jpg") {
		t.Errorf("body should not contain the consumed embed, got %q", body)
	}
	if !containsString(body, "Flagship offering.") || !containsString(body, "Details below.") {
		t.Errorf("body lost regular text: %q", body)
	}
}

func TestExtractFirstImage_FrontmatterWinsOverEmbed(t *testing.T) {
	in := "---\nimage: hero.webp\n---\nText with ![[other.png]] embed."
	img, _ := extractFirstImage(in)
	if img != "hero.webp" {
		t.Errorf("frontmatter must win: want %q, got %q", "hero.webp", img)
	}
}

func TestExtractFirstImage_NoImage(t *testing.T) {
	in := "---\ntitle: Contact\n---\n\nReach out: @owner"
	img, body := extractFirstImage(in)
	if img != "" {
		t.Errorf("no image expected, got %q", img)
	}
	if body != "Reach out: @owner" {
		t.Errorf("body: want %q, got %q", "Reach out: @owner", body)
	}
}

func TestExtractFirstImage_EmbedNonImageIgnored(t *testing.T) {
	in := "Note refs ![[other.md]] which is not an image."
	img, body := extractFirstImage(in)
	if img != "" {
		t.Errorf("non-image embed should not match, got %q", img)
	}
	if body == "" {
		t.Errorf("body should be preserved when no image extracted")
	}
}

func TestExtractFirstImage_NoFrontmatter(t *testing.T) {
	in := "Plain text with ![[a.png]] inline."
	img, body := extractFirstImage(in)
	if img != "a.png" {
		t.Errorf("image: want %q, got %q", "a.png", img)
	}
	if containsString(body, "a.png") {
		t.Errorf("body should drop the embed, got %q", body)
	}
}

func containsString(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
