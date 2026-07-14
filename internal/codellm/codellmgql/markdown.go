package codellmgql

import (
	"strings"

	"trip2g/internal/codellm/codellmgql/model"
	"trip2g/internal/coderun"
)

// parseMarkdownBlocks splits md into ordered code/prose blocks. It reuses
// coderun.ExtractFencedBlocks — codellm's OWN fence parser, the same one
// execution runs with — so the block boundaries the editor sees match exactly
// what codellm executes. Prose is everything between/around the code fences,
// recovered by splitting md on each block's reconstructed fence text.
//
// Round-trip guarantee: assembleMarkdownBlocks(parseMarkdownBlocks(md)) == md
// for normally-formatted fences (```lang\n...\n```). The guarantee holds because
// prose segments are the exact slices of md between the located fences, and each
// code block is re-fenced with its own language.
func parseMarkdownBlocks(md string) []model.MdBlock {
	fenced := coderun.ExtractFencedBlocks(md)
	blocks := make([]model.MdBlock, 0, len(fenced)*2+1)

	cursor := 0
	idx := 0
	for _, fb := range fenced {
		fence := fenceText(fb.Lang, fb.Code)
		rel := strings.Index(md[cursor:], fence)
		if rel < 0 {
			// Reconstructed fence not found verbatim (e.g. an unusual info string
			// with surrounding spaces ExtractFencedBlocks trimmed). Skip it rather
			// than misattribute prose — round-trip is best-effort for such fences.
			continue
		}
		pos := cursor + rel
		if pos > cursor {
			blocks = append(blocks, proseBlock(idx, md[cursor:pos]))
			idx++
		}
		lang := fb.Lang
		blocks = append(blocks, model.MdBlock{
			Index:    idx,
			Kind:     model.BlockKindCode,
			Language: &lang,
			Content:  fb.Code,
		})
		idx++
		cursor = pos + len(fence)
	}
	if cursor < len(md) {
		blocks = append(blocks, proseBlock(idx, md[cursor:]))
	}
	return blocks
}

// assembleMarkdownBlocks is the inverse of parseMarkdownBlocks: it concatenates
// the ordered blocks back into a single markdown string. CODE blocks are
// re-fenced with their language; PROSE blocks are emitted verbatim.
func assembleMarkdownBlocks(blocks []model.MdBlockInput) string {
	var sb strings.Builder
	for _, b := range blocks {
		if b.Kind == model.BlockKindCode {
			lang := ""
			if b.Language != nil {
				lang = *b.Language
			}
			sb.WriteString(fenceText(lang, b.Content))
		} else {
			sb.WriteString(b.Content)
		}
	}
	return sb.String()
}

// fenceText reconstructs a fenced code block's exact text: ```<lang>\n<code>```.
func fenceText(lang, code string) string {
	return "```" + lang + "\n" + code + "```"
}

// proseBlock builds a PROSE MdBlock (language is null for prose).
func proseBlock(index int, content string) model.MdBlock {
	return model.MdBlock{Index: index, Kind: model.BlockKindProse, Content: content}
}
