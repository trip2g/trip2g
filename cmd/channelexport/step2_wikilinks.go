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
	// Telegram channel link: [text](https://t.me/ryspaisensei/123)
	tgLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\(https?://t\.me/ryspaisensei/(\d+)\)`)
	// Custom emoji with capture groups: ![emoji](tg://emoji?id=123456)
	customEmojiReplaceRegex = regexp.MustCompile(`!\[([^\]]*)\]\(tg://emoji\?id=(\d+)\)`)
)

// PostInfo holds info about a post.
type PostInfo struct {
	ID       string
	Title    string // filename without .md
	Filename string
}

func runStep2(inputDir, outputDir string) error {
	// Create output directory
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build map of message ID -> post info
	postMap, err := buildPostMap(inputDir)
	if err != nil {
		return fmt.Errorf("failed to build post map: %w", err)
	}
	log.Printf("Built map with %d posts", len(postMap))

	// Read all .md files and process
	files, err := filepath.Glob(filepath.Join(inputDir, "*.md"))
	if err != nil {
		return fmt.Errorf("failed to glob files: %w", err)
	}

	log.Printf("Processing %d files", len(files))

	processedCount := 0
	linksReplaced := 0

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("❌ Failed to read %s: %v", file, err)
			continue
		}

		// Replace telegram links with wikilinks
		newContent, replaced := replaceTelegramLinks(string(content), postMap)
		linksReplaced += replaced

		// Write to output
		outputPath := filepath.Join(outputDir, filepath.Base(file))
		err = os.WriteFile(outputPath, []byte(newContent), 0644)
		if err != nil {
			log.Printf("❌ Failed to write %s: %v", outputPath, err)
			continue
		}

		processedCount++
	}

	log.Printf("✓ Processed: %d files, replaced %d links", processedCount, linksReplaced)
	return nil
}

func buildPostMap(inputDir string) (map[string]PostInfo, error) {
	postMap := make(map[string]PostInfo)

	files, err := filepath.Glob(filepath.Join(inputDir, "*.md"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Extract message ID from frontmatter
		id := extractMessageID(string(content))
		if id == "" {
			continue
		}

		filename := filepath.Base(file)
		title := strings.TrimSuffix(filename, ".md")

		postMap[id] = PostInfo{
			ID:       id,
			Title:    title,
			Filename: filename,
		}
	}

	return postMap, nil
}

func extractMessageID(content string) string {
	// Look for telegram_publish_message_id in frontmatter
	if !strings.HasPrefix(content, "---") {
		return ""
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return ""
	}

	frontmatter := parts[1]
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "telegram_publish_message_id:") {
			value := strings.TrimPrefix(line, "telegram_publish_message_id:")
			value = strings.TrimSpace(value)
			value = strings.Trim(value, "\"'")
			return value
		}
	}

	return ""
}

func replaceTelegramLinks(content string, postMap map[string]PostInfo) (string, int) {
	replaced := 0

	result := tgLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the ID from the match
		submatches := tgLinkRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		linkText := submatches[1]
		postID := submatches[2]

		// Look up in map
		if post, ok := postMap[postID]; ok {
			replaced++
			return fmt.Sprintf("[[%s]]", post.Title)
		}

		// Not found in map - keep original link but log
		log.Printf("  Link not found in map: ID=%s, text=%q", postID, linkText)
		return match
	})

	// Replace custom emoji tg://emoji?id=... with https://ce.trip2g.com/{id}.webp
	result = customEmojiReplaceRegex.ReplaceAllStringFunc(result, func(match string) string {
		submatches := customEmojiReplaceRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		altText := submatches[1]
		emojiID := submatches[2]
		replaced++
		return fmt.Sprintf("![%s](https://ce.trip2g.com/%s.webp)", altText, emojiID)
	})

	return result, replaced
}
