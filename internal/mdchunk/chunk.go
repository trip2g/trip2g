package mdchunk

import "strings"

const (
	chunkTargetSize = 2000 // chars (~500 tokens)
	chunkMinSize    = 300  // chars
	chunkOverlap    = 200  // chars — last block of prev chunk prepended to next
)

// Chunk is a fragment of a note prepared for vector embedding.
type Chunk struct {
	Index   int
	Content string // "{title}\n\n{body}"
}

// Split splits a note into chunks suitable for vector embedding.
// It strips frontmatter, splits content into Markdown blocks at paragraph
// boundaries, respects heading boundaries as hard split points, avoids tiny
// chunks by accumulating below chunkMinSize, and adds overlap between chunks.
func Split(title string, rawContent []byte) []Chunk {
	body := StripFrontmatter(string(rawContent))
	blocks := splitIntoBlocks(body)
	if len(blocks) == 0 {
		return []Chunk{{Index: 0, Content: title}}
	}

	var chunks []Chunk
	var current []string
	currentSize := 0

	flush := func() {
		if len(current) == 0 {
			return
		}
		chunks = append(chunks, Chunk{
			Index:   len(chunks),
			Content: title + "\n\n" + strings.Join(current, "\n\n"),
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
			currentSize = len(overlap)
		}
	}

	for i, block := range blocks {
		// Heading always causes a split if there is accumulated content before it.
		if isHeadingBlock(block) && len(current) > 0 {
			startNewChunk()
		}

		current = append(current, block)
		currentSize += len(block)

		// Flush when we reach the target size (if above min size),
		// but not on the last block — the remaining flush handles that.
		if currentSize >= chunkTargetSize && currentSize >= chunkMinSize && i < len(blocks)-1 {
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

// isHeadingBlock reports whether a block starts with a Markdown heading (# to ######).
func isHeadingBlock(block string) bool {
	i := 0
	for i < len(block) && block[i] == '#' {
		i++
	}
	return i > 0 && i <= 6 && i < len(block) && block[i] == ' '
}
