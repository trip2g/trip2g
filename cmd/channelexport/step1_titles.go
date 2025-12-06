package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Custom emoji in markdown: ![emoji](tg://emoji?id=123) or ![emoji](https://ce.trip2g.com/123.webp)
	customEmojiRegex = regexp.MustCompile(`!\[[^\]]*\]\((tg://emoji\?id=\d+|https://ce\.trip2g\.com/\d+\.webp)\)`)
	// Malformed custom emoji with extra chars: ![<u](tg://emoji?id=123)>➡️</u>.
	malformedEmojiRegex = regexp.MustCompile(`!\[[^\]]*\]\(tg://emoji\?id=\d+\)>[^<]*</u>`)
	// Numbered list prefix with emoji: ![1️⃣](url). or ![1️⃣](url) followed by dot/space.
	numberedEmojiPrefixRegex = regexp.MustCompile(`^!\[[^\]]*\]\([^)]+\)[\.\s]*`)
	// Markdown links: [text](url) -> text.
	markdownLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\([^)]+\)`)
	// HTML tags like <u>, </u>, <b>, </b>.
	htmlTagRegex = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	// Timecodes: 00:00, 1:23, 01:23:45 (anywhere in text).
	timecodeRegex = regexp.MustCompile(`\d{1,2}:\d{2}(?::\d{2})?\s*`)
	// Leading emoji (including skin tones) and special chars.
	leadingJunkRegex = regexp.MustCompile(`^[\x{1F300}-\x{1F9FF}\x{1F3FB}-\x{1F3FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{25A0}-\x{25FF}\x{2B00}-\x{2BFF}\x{FE00}-\x{FE0F}\x{200D}\s\-–—•·°№#@!?\.,;:\*"'«»„"'']+`)
)

func runStep1(inputDir, outputDir string) error {
	// Create output directory
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Read all .md files from input directory
	files, err := filepath.Glob(filepath.Join(inputDir, "*.md"))
	if err != nil {
		return fmt.Errorf("failed to glob files: %w", err)
	}

	log.Printf("Found %d files in %s", len(files), inputDir)

	usedFilenames := make(map[string]bool)
	processedCount := 0
	failedCount := 0

	for i, file := range files {
		log.Printf("=== Processing %d/%d: %s ===", i+1, len(files), filepath.Base(file))

		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("❌ Failed to read %s: %v", file, err)
			failedCount++
			continue
		}

		// Extract message ID from filename (e.g., "123.md" -> 123)
		baseName := filepath.Base(file)
		messageID := strings.TrimSuffix(baseName, ".md")

		// Extract title from content
		title := extractTitle(string(content))
		if title == "" {
			title = fmt.Sprintf("message-%s", messageID)
		}

		// Generate unique filename
		filename := generateFilename(title, messageID, usedFilenames)
		usedFilenames[filename] = true

		// Write to output
		outputPath := filepath.Join(outputDir, filename)
		err = os.WriteFile(outputPath, content, 0644)
		if err != nil {
			log.Printf("❌ Failed to write %s: %v", outputPath, err)
			failedCount++
			continue
		}

		log.Printf("✓ %s -> %s", baseName, filename)
		processedCount++
	}

	log.Printf("✓ Processed: %d, ❌ Failed: %d, Total: %d", processedCount, failedCount, len(files))
	return nil
}

func extractTitle(content string) string {
	// Skip frontmatter if present
	text := content
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			text = strings.TrimSpace(parts[2])
		}
	}

	// Remove malformed custom emoji first (more specific pattern)
	text = malformedEmojiRegex.ReplaceAllString(text, "")

	// Remove custom emoji markdown: ![emoji](tg://emoji?id=123)
	text = customEmojiRegex.ReplaceAllString(text, "")

	// Remove HTML tags: <u>, </u>, etc
	text = htmlTagRegex.ReplaceAllString(text, "")

	// Convert markdown links to just text: [text](url) -> text
	text = markdownLinkRegex.ReplaceAllString(text, "$1")

	// Remove markdown formatting early
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "`", "")

	// Remove timecodes globally: 00:00, 1:23:45
	text = timecodeRegex.ReplaceAllString(text, "")

	// Get first non-empty line (for title extraction)
	var firstLine string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			firstLine = line
			break
		}
	}
	firstParagraph := firstLine

	// Remove numbered emoji prefix: ![1️⃣](url). at the start
	firstParagraph = numberedEmojiPrefixRegex.ReplaceAllString(firstParagraph, "")

	// Strip leading junk repeatedly until stable
	for {
		cleaned := leadingJunkRegex.ReplaceAllString(firstParagraph, "")
		cleaned = strings.TrimSpace(cleaned)
		if cleaned == firstParagraph {
			break
		}
		firstParagraph = cleaned
	}

	// Take first 7 words
	words := strings.Fields(firstParagraph)
	if len(words) > 7 {
		words = words[:7]
	}

	title := strings.Join(words, " ")

	// Remove invalid filename characters
	invalidChars := []string{"/", "\\", ":", "?", "\"", "<", ">", "|", "[", "]", "(", ")", "#"}
	for _, char := range invalidChars {
		title = strings.ReplaceAll(title, char, "")
	}

	// Strip trailing punctuation
	title = strings.TrimRight(title, ".,;:!?…-–—")

	return strings.TrimSpace(title)
}

func generateFilename(title string, messageID string, usedFilenames map[string]bool) string {
	baseFilename := title + ".md"

	if !usedFilenames[baseFilename] {
		return baseFilename
	}

	// Add message ID to make it unique
	return fmt.Sprintf("%s (%s).md", title, messageID)
}
