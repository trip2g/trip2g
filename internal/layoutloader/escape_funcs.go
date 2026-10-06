package layoutloader

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"
	"trip2g/internal/model"

	"github.com/CloudyKit/jet/v6"
)

// addEscapeFuncs replaces Jet's html, url and json builtins with versions that
// return model.SafeHTML, so their already-escaped output is not escaped twice.
func addEscapeFuncs(views *jet.Set) {
	views.AddGlobal("html", func(s string) model.SafeHTML {
		return model.SafeHTML(html.EscapeString(s))
	})

	views.AddGlobal("url", func(s string) model.SafeHTML {
		return model.SafeHTML(url.QueryEscape(s))
	})

	views.AddGlobal("json", func(v any) model.SafeHTML {
		raw, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		return model.SafeHTML(raw)
	})
}

// attrSafeURL percent-encodes the characters that would let a URL leave a
// quoted attribute or a CSS url(""). A URL that has none comes back unchanged.
func attrSafeURL(u string) model.SafeHTML {
	var sb strings.Builder
	for i := range len(u) {
		c := u[i]
		if c <= ' ' || c == 0x7f || strings.IndexByte(`"'<>\`+"`", c) >= 0 {
			fmt.Fprintf(&sb, "%%%02X", c)
			continue
		}
		sb.WriteByte(c)
	}
	return model.SafeHTML(sb.String())
}
