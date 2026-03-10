package defaulttemplate

import _ "embed"

//go:embed defaulttemplate.css
var cssContent string

// InlineCSS returns a <style> tag with the embedded CSS.
func InlineCSS() string { return "<style>" + cssContent + "</style>" }
