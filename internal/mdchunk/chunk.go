package mdchunk

import "strings"

const (
	// Sizing is in ESTIMATED TOKENS (see estimateTokens), not characters: at
	// ~2 chars/token for Cyrillic, the old 2000-char target overflowed the
	// 512-token window of bge-m3/E5, so the tail of Russian chunks was silently
	// truncated server-side and never embedded.
	chunkTargetTokens = 450 // keep comfortably under a 512-token encoder window
	chunkMinTokens    = 60
	chunkOverlap      = 200 // chars — tail of prev chunk prepended to next
)

// Chunk is a fragment of a note prepared for vector embedding.
type Chunk struct {
	Index   int
	Content string // "{title} > {h1} > {h2}...\n\n{body}"
}

// estimateTokens approximates the subword-token count of s without a tokenizer.
// Cyrillic tokenizes to roughly one token per ~2 characters; Latin/other to
// roughly one per ~4. Good enough to keep chunks under an encoder window.
func estimateTokens(s string) int {
	cyr, other := 0, 0
	for _, r := range s {
		if r >= 0x0400 && r <= 0x04FF {
			cyr++
		} else {
			other++
		}
	}
	return cyr/2 + other/4 + 1
}

// Split splits a note into chunks suitable for vector embedding.
// It strips frontmatter, splits content into Markdown blocks at paragraph
// boundaries, respects heading boundaries as hard split points, avoids tiny
// chunks by accumulating below chunkMinTokens, sizes by estimated tokens, and
// prepends a heading breadcrumb ("{title} > {h1} > {h2}...") to every chunk so
// deep chunks carry their document context (a cheap form of contextual retrieval).
func Split(title string, rawContent []byte) []Chunk { //nolint:gocognit // inherently complex markdown chunking logic
	body := StripFrontmatter(string(rawContent))
	blocks := splitIntoBlocks(body)
	if len(blocks) == 0 {
		return []Chunk{{Index: 0, Content: title}}
	}

	var chunks []Chunk
	var current []string
	currentSize := 0
	var headingStack []string // headingStack[level-1] = heading text at that level

	prefix := func() string {
		parts := []string{title}
		for _, h := range headingStack {
			if h != "" {
				parts = append(parts, h)
			}
		}
		return strings.Join(parts, " > ")
	}

	flush := func() {
		if len(current) == 0 {
			return
		}
		chunks = append(chunks, Chunk{
			Index:   len(chunks),
			Content: prefix() + "\n\n" + strings.Join(current, "\n\n"),
		})
	}

	overlapBlock := func() string {
		if len(current) == 0 {
			return ""
		}
		last := current[len(current)-1]
		if len(last) > chunkOverlap {
			return last[len(last)-chunkOverlap:]
		}
		return last
	}

	startNewChunk := func() {
		overlap := overlapBlock()
		flush()
		current = nil
		currentSize = 0
		if overlap != "" {
			current = append(current, overlap)
			currentSize = estimateTokens(overlap)
		}
	}

	setHeading := func(level int, text string) {
		if level > len(headingStack) {
			for len(headingStack) < level {
				headingStack = append(headingStack, "")
			}
		} else {
			headingStack = headingStack[:level] // drop deeper levels
		}
		headingStack[level-1] = text
	}

	for i, block := range blocks {
		// Heading always causes a split if there is accumulated content before it,
		// then updates the breadcrumb stack for the chunk it begins.
		if level, text, ok := parseHeading(block); ok {
			if len(current) > 0 {
				startNewChunk()
			}
			setHeading(level, text)
		}

		current = append(current, block)
		currentSize += estimateTokens(block)

		// Flush when we reach the target size (if above min size),
		// but not on the last block — the remaining flush handles that.
		if currentSize >= chunkTargetTokens && currentSize >= chunkMinTokens && i < len(blocks)-1 {
			startNewChunk()
		}
	}

	if len(current) > 0 {
		flush()
	}

	if len(chunks) == 0 {
		return []Chunk{{Index: 0, Content: title}}
	}

	return chunks
}

// splitIntoBlocks splits markdown content into paragraph-level blocks at
// double-newline boundaries. Code fences (``` ... ```) are treated as atomic
// units — blank lines inside a code fence do not end the current block.
func splitIntoBlocks(content string) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	var blocks []string
	var current strings.Builder
	inCodeFence := false

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeFence = !inCodeFence
		}

		if !inCodeFence && line == "" {
			if block := strings.TrimSpace(current.String()); block != "" {
				blocks = append(blocks, block)
			}
			current.Reset()
		} else {
			if current.Len() > 0 {
				current.WriteByte('\n')
			}
			current.WriteString(line)
		}
	}

	if block := strings.TrimSpace(current.String()); block != "" {
		blocks = append(blocks, block)
	}

	return blocks
}

// parseHeading returns the level (1–6) and trimmed text of a Markdown heading
// block, or ok=false if the block is not a heading. Only the first line is
// considered, so a heading followed by body text in the same block still parses.
func parseHeading(block string) (int, string, bool) {
	line := block
	if nl := strings.IndexByte(block, '\n'); nl >= 0 {
		line = block[:nl]
	}
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 || i > 6 || i >= len(line) || line[i] != ' ' {
		return 0, "", false
	}
	return i, strings.TrimSpace(line[i+1:]), true
}
