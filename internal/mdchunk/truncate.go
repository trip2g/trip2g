package mdchunk

// TruncateToTokens returns the longest rune-aligned prefix of s whose estimated
// token count (see estimateTokens) does not exceed maxTokens. If s is already
// within budget it is returned unchanged. Used to keep whole-note embedding
// input under the model's hard input-token limit, since notes can be arbitrarily
// large while embedding APIs reject inputs past their window.
func TruncateToTokens(s string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	if estimateTokens(s) <= maxTokens {
		return s
	}
	// estimateTokens is cyr/2 + other/4 + 1; walk runes accumulating the same
	// weights and stop before the budget is exceeded. Cutting on a rune boundary
	// keeps the result valid UTF-8.
	cyr, other := 0, 0
	for i, r := range s {
		if r >= 0x0400 && r <= 0x04FF {
			cyr++
		} else {
			other++
		}
		if cyr/2+other/4+1 > maxTokens {
			return s[:i]
		}
	}
	return s
}
