package defaulttemplate

import (
	"net/http"

	"github.com/valyala/fasthttp"
)

// fallbackSystemMessage stands in for a message key the bundle does not know,
// so a typo surfaces as a generic apology rather than as the key itself.
const fallbackSystemMessage = "system_error"

// systemMessage is the copy SystemPage renders.
type systemMessage struct {
	Lang  string
	Title string
	Text  string
	Home  string
}

// WriteSystemMessage answers with a plain page a person can read: what
// happened, what to do about it, and a way back. msg names the situation
// ("hat_expired"); its copy comes from the translation bundle as <msg>_title
// and <msg>_text, in the visitor's own language.
//
// The page shares nothing with the site template on purpose — no notes, no
// bundles, no injections — because the requests that need it are the ones that
// never got far enough to have any of that.
func WriteSystemMessage(ctx *fasthttp.RequestCtx, code int, msg string) {
	lang := uiLangFromCtx(ctx)

	if !hasSystemMessage(lang, msg) {
		msg = fallbackSystemMessage
	}

	if code == 0 {
		code = http.StatusInternalServerError
	}

	ctx.ResetBody()
	ctx.SetStatusCode(code)
	ctx.SetContentType("text/html; charset=utf-8")

	// T falls back to English on an unknown or empty language; <html lang>
	// needs an actual tag, so it follows the same fallback.
	htmlLang := lang
	if htmlLang == "" {
		htmlLang = "en"
	}

	WriteSystemPage(ctx, systemMessage{
		Lang:  htmlLang,
		Title: T(lang, msg+"_title"),
		Text:  T(lang, msg+"_text"),
		Home:  T(lang, "system_home"),
	})
}

// hasSystemMessage reports whether the bundle knows msg. T echoes the message
// ID back when it does not, which is what makes the check possible.
func hasSystemMessage(lang, msg string) bool {
	titleKey := msg + "_title"

	return msg != "" && T(lang, titleKey) != titleKey
}
