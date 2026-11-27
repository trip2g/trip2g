package defaulttemplate

import (
	"html"
	"net/url"
	"strings"

	"trip2g/internal/usertoken"
)

const leaveGuestPath = "/_system/federation/leave"

// GuestBar is the line a federation guest sees on every page: where they came
// from, the way back and the way out. It returns an empty string for anyone
// else, so a layout calls it without a condition.
func GuestBar(token *usertoken.Data, lang string) string {
	if !token.IsFederationGuest() {
		return ""
	}

	parsed, err := url.Parse(token.Origin)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return guestBarWithoutHub(lang)
	}

	host := html.EscapeString(parsed.Host)
	back := html.EscapeString(parsed.Scheme + "://" + parsed.Host + "/")

	var sb strings.Builder
	sb.WriteString(`<div class="guest-bar" role="note"><span>`)
	sb.WriteString(html.EscapeString(T(lang, "guest_bar_from")))
	sb.WriteString(` <strong>`)
	sb.WriteString(host)
	sb.WriteString(`</strong></span> <a href="`)
	sb.WriteString(back)
	sb.WriteString(`" rel="noopener">`)
	sb.WriteString(html.EscapeString(T(lang, "guest_bar_back")))
	sb.WriteString(` `)
	sb.WriteString(host)
	sb.WriteString(`</a>`)
	writeLeave(&sb, lang)
	sb.WriteString(`</div>`)
	return sb.String()
}

func guestBarWithoutHub(lang string) string {
	var sb strings.Builder
	sb.WriteString(`<div class="guest-bar" role="note"><span>`)
	sb.WriteString(html.EscapeString(T(lang, "guest_bar_guest")))
	sb.WriteString(`</span>`)
	writeLeave(&sb, lang)
	sb.WriteString(`</div>`)
	return sb.String()
}

func writeLeave(sb *strings.Builder, lang string) {
	sb.WriteString(` <form method="post" action="` + leaveGuestPath + `"><button type="submit">`)
	sb.WriteString(html.EscapeString(T(lang, "guest_bar_leave")))
	sb.WriteString(`</button></form>`)
}
