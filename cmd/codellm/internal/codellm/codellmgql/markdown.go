package codellmgql

import (
	"bytes"
	"strings"

	"trip2g/cmd/codellm/internal/codellm/codellmgql/model"
	"trip2g/cmd/codellm/internal/coderun"
	"trip2g/internal/yamlutil"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
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
func parseMarkdownBlocks(md string) []model.MarkdownBlock {
	fenced := coderun.ExtractFencedBlocks(md)
	blocks := make([]model.MarkdownBlock, 0, len(fenced)*2+1)

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
		blocks = append(blocks, model.MarkdownCodeBlock{
			Index:    idx,
			Language: lang,
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

func parseMarkdownDocument(md string) (map[string]interface{}, []model.MarkdownBlock) {
	frontmatter := parseFrontmatter(md)
	body := stripFrontmatter(md)
	return frontmatter, parseMarkdownBlocks(body)
}

func parseFrontmatter(md string) map[string]interface{} {
	context := parser.NewContext()
	goldmark.New(goldmark.WithExtensions(meta.Meta)).Parser().Parse(
		text.NewReader([]byte(md)), parser.WithContext(context),
	)
	value := meta.Get(context)
	if value == nil {
		return map[string]interface{}{}
	}
	normalized, ok := yamlutil.Normalize(value).(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return normalized
}

func stripFrontmatter(md string) string {
	if !strings.HasPrefix(md, "---\n") {
		return md
	}
	end := strings.Index(md[4:], "\n---")
	if end < 0 {
		return md
	}
	end += 4
	lineEnd := strings.IndexByte(md[end+4:], '\n')
	if lineEnd < 0 {
		return ""
	}
	return md[end+4+lineEnd+1:]
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
func proseBlock(index int, content string) model.MarkdownBlock {
	return model.MarkdownProseBlock{
		Index:   index,
		Content: content,
		HTML:    renderProse(content),
	}
}

func renderProse(content string) string {
	var output bytes.Buffer
	_ = goldmark.New(goldmark.WithExtensions(extension.GFM)).Convert([]byte(content), &output)
	return output.String()
}
