package enclavefix

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"trip2g/internal/image"

	"github.com/quailyquaily/goldmark-enclave/core"
	"github.com/quailyquaily/goldmark-enclave/helper"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type astTransformer struct {
	cfg *core.Config
}

func (a *astTransformer) InsertFailedHint(n ast.Node, msg string) {
	msgNode := ast.NewString([]byte(fmt.Sprintf("\n<!-- goldmark-enclave: %s -->\n", msg)))
	msgNode.SetCode(true)
	n.Parent().InsertAfter(n.Parent(), n, msgNode)
}

// imageReplacement holds data for deferred image-to-enclave replacement.
type imageReplacement struct {
	img     *ast.Image
	enclave *core.Enclave
}

//nolint:gocognit,gocyclo,cyclop,funlen // provider detection requires many branches.
func (a *astTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	// Pass 1: Walk and collect replacements without modifying the AST.
	// ReplaceChild inside ast.Walk breaks sibling traversal,
	// causing only the first image per paragraph to be transformed.
	var replacements []imageReplacement

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if n.Kind() != ast.KindImage {
			return ast.WalkContinue, nil
		}

		img := n.(*ast.Image)
		u, err := url.Parse(string(img.Destination))
		if err != nil {
			a.InsertFailedHint(n, fmt.Sprintf("failed to parse url: %s, %s", img.Destination, err))
			return ast.WalkContinue, nil
		}

		// [alt](url "title")
		// read the title and alt from markdown
		title := string(img.Title)
		altText := helper.ExtractTextRecursivelyByReader(n, reader)

		oid := ""
		theme := "light"
		provider := ""
		params := map[string]string{}
		if u.Host == "www.youtube.com" && u.Path == "/watch" {
			// this is a youtube video: https://www.youtube.com/watch?v={vid}
			provider = core.EnclaveProviderYouTube
			oid = u.Query().Get("v")
		} else if u.Host == "youtu.be" {
			// this is also a youtube video: https://youtu.be/{vid}
			provider = core.EnclaveProviderYouTube
			oid = u.Path[1:]
			oid = strings.Trim(oid, "/")

		} else if u.Host == "www.bilibili.com" && strings.HasPrefix(u.Path, "/video/") {
			// this is a bilibili video: https://www.bilibili.com/video/{vid}
			provider = core.EnclaveProviderBilibili
			oid = u.Path[7:]
			oid = strings.Trim(oid, "/")

		} else if u.Host == "twitter.com" || u.Host == "m.twitter.com" || u.Host == "x.com" {
			// https://twitter.com/{username}/status/{id number}?theme=dark
			provider = core.EnclaveProviderTwitter
			oid = string(img.Destination)
			if u.Host == "x.com" {
				// replace x.com with twitter.com, because x.com doesn't support using x.com as the source host, what a shame
				oid = strings.Replace(oid, "x.com", "twitter.com", 1)
			}
			theme = u.Query().Get("theme")

		} else if u.Host == "tradingview.com" || u.Host == "www.tradingview.com" {
			// https://www.tradingview.com/chart/UC0wWW9o/?symbol=BITFINEX%3ABTCUSD
			provider = core.EnclaveProviderTradingView
			oid = u.Query().Get("symbol")
			theme = u.Query().Get("theme")

		} else if u.Host == "open.spotify.com" {
			// https://open.spotify.com/track/5vdp5UmvTsnMEMESIF2Ym7?si=d4ee09bfd0e941c5
			const re = `^track/([a-zA-Z0-9_-]+)$`
			provider = core.EnclaveProviderSpotify
			if len(u.Path) > 1 {
				p := strings.Trim(u.Path[1:], "/")
				// get the track id after /track/
				ok, _ := regexp.MatchString(re, p)
				if ok {
					oid = strings.Split(p, "/")[1]
				}
			}

		} else if u.Host == "www.podbean.com" || u.Host == "podbean.com" {
			// Examples:
			//  - https://www.podbean.com/ew/pb-s9x5a-196f966
			//  - https://www.podbean.com/eas/pb-s9x5a-196f966
			// Convert path segment `pb-<a>-<b>` to embed id `i=<a>-<b>-pb`
			provider = core.EnclaveProviderPodbean
			if strings.HasPrefix(u.Path, "/ew/") || strings.HasPrefix(u.Path, "/eas/") {
				re := regexp.MustCompile(`^/(?:ew|eas)/pb-([a-zA-Z0-9]+)-([a-zA-Z0-9]+)$`)
				if re.MatchString(u.Path) {
					m := re.FindStringSubmatch(u.Path)
					if len(m) == 3 {
						oid = fmt.Sprintf("%s-%s-pb", m[1], m[2])
					}
				}
			}
			theme = u.Query().Get("theme")

		} else if image.IsAudioExtension(u.Path) {
			provider = core.EnclaveHtml5Audio
			oid = string(img.Destination)

		} else {
			// check the resize params
			// form 1: ![](https://example.com/image.jpg?w=200&h=100&center=1)
			// form 2: ![](https://example.com/image.jpg|200x100) or ![](https://example.com/image.jpg|200)
			// form 3: ![alt|200x100](https://example.com/image.jpg) or ![alt|200](https://example.com/image.jpg)
			// if the width and height are numbers only, we assume it's unit is px. If not, we need to parse the unit from the string to check it.
			// supported units: %, px, rem
			w := u.Query().Get("w")
			if w == "" {
				w = u.Query().Get("width")
			}
			h := u.Query().Get("h")
			if h == "" {
				h = u.Query().Get("height")
			}
			align := u.Query().Get("align")

			destination := string(img.Destination)
			reForm := regexp.MustCompile(`\|(\d+%?|rem?|px?)(?:x(\d+%?|rem?|px?))?`)
			// check the form 2, the tail of img.Destination is like |200x100 or |200
			if strings.Contains(destination, "|") {
				matches := reForm.FindStringSubmatch(destination)
				if len(matches) > 1 {
					w = matches[1]
					if len(matches) > 2 {
						h = matches[2]
					}
				}
				destination = strings.Split(destination, "|")[0]
			}

			if strings.Contains(altText, "|") {
				matches := reForm.FindStringSubmatch(altText)
				if len(matches) > 1 {
					w = matches[1]
					if len(matches) > 2 {
						h = matches[2]
					}
				}
			}

			if len(title) != 0 || w != "" || h != "" || align != "" {
				// this is a normal image, but it has a title, so we add a caption
				provider = core.EnclaveProviderQuailImage
				oid = destination
				if title != "" {
					params["title"] = string(img.Title)
				}
				if altText != "" {
					params["alt"] = altText
				}
				if w != "" {
					params["width"] = w
				}
				if h != "" {
					params["height"] = h
				}
				if align != "" {
					params["align"] = align
				}
			} else {
				provider = core.EnclaveRegularImage
				oid = destination
			}
			u, err = url.Parse(destination)
			if err != nil {
				return ast.WalkContinue, nil
			}
		}

		if oid != "" {
			ev := NewEnclave(
				&core.Enclave{
					Image:          *img,
					Alt:            altText,
					Title:          title,
					URL:            u,
					Provider:       provider,
					ObjectID:       oid,
					Theme:          theme,
					Params:         params,
					IframeDisabled: a.cfg.IframeDisabled,
				})

			replacements = append(replacements, imageReplacement{img: img, enclave: ev})
		}

		return ast.WalkContinue, nil
	})

	// Pass 2: Apply all replacements after the walk is complete.
	for _, r := range replacements {
		parent := r.img.Parent()
		if parent != nil {
			parent.ReplaceChild(parent, r.img, r.enclave)
		}
	}
}
