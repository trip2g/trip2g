package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/tg"

	"trip2g/internal/tgtd"
)

func runExport(ctx context.Context, sessionPath string, args []string) error {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	channelID := fs.Int64("channel", 0, "channel ID (required; get it from 'channels' command)")
	outDir := fs.String("out", "", "output directory for markdown files (required)")
	withMedia := fs.Bool("media", false, "download media files into assets/ subdirectory")
	noSkip := fs.Bool("no-skip-exists", false, "re-export messages that already have a file on disk")
	fs.Usage = func() {
		fmt.Println(`usage: exporttgchannel export --channel ID --out DIR [--media] [--no-skip-exists]

Exports all messages from a Telegram channel to Obsidian-compatible markdown files.

Flags:
  --channel ID         numeric channel ID (get it from 'channels' command)
  --out DIR            output directory; created if it does not exist
  --media              download photos and videos into DIR/assets/
  --no-skip-exists     overwrite files that already exist on disk

Output structure:
  DIR/
    _index.md           chronological index with wikilinks to all posts
    My post title.md    one file per message group
    assets/             media files (only with --media)
      12345_0.jpg`)
	}
	_ = fs.Parse(args)

	if *channelID == 0 {
		return fmt.Errorf("--channel is required (use 'exporttgchannel channels' to find IDs)")
	}
	if *outDir == "" {
		return fmt.Errorf("--out is required")
	}

	sess, err := loadSession(sessionPath)
	if err != nil {
		return err
	}
	sessionData, err := sess.sessionBytes()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	client := tgtd.NewClient(&cliEnv{}, 0, sess.APIID, sess.APIHash).WithClock(globalClock)

	// Phase 1: fetch all messages via a single API session.
	fmt.Println("Fetching messages...")
	var allMessages []*tg.Message

	err = client.RunWithAPI(ctx, sessionData, func(ctx context.Context, api *tg.Client) error {
		offsetID := 0
		for {
			result, fetchErr := client.GetChannelMessagesWithAPI(ctx, api, tgtd.GetChannelMessagesParams{
				ChannelID: *channelID,
				Limit:     100,
				OffsetID:  offsetID,
			})
			if fetchErr != nil {
				return fmt.Errorf("fetch messages: %w", fetchErr)
			}
			if len(result.Messages) == 0 {
				break
			}
			allMessages = append(allMessages, result.Messages...)
			fmt.Printf("  fetched %d messages...\n", len(allMessages))
			offsetID = result.Messages[len(result.Messages)-1].ID
			if !result.HasMore {
				break
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("Total: %d messages\n", len(allMessages))

	// Phase 2: group into logical posts, sort newest-first.
	groups := groupByMediaGroup(allMessages)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].primary.ID > groups[j].primary.ID
	})

	// Build post map (messageID → filename-without-ext) for wikilink resolution.
	usedFilenames := make(map[string]bool)
	posts := make([]postInfo, len(groups))
	postMap := make(map[string]string) // messageID → title-without-ext

	for i, g := range groups {
		md := tgtd.Convert(g.primary)
		title := extractTitle(md)
		if title == "" {
			title = fmt.Sprintf("message-%d", g.primary.ID)
		}
		filename := uniqueFilename(title, g.primary.ID, usedFilenames)
		usedFilenames[filename] = true
		titleNoExt := strings.TrimSuffix(filename, ".md")
		for _, m := range g.all {
			postMap[strconv.Itoa(m.ID)] = titleNoExt
		}
		posts[i] = postInfo{group: g, filename: filename, title: titleNoExt}
	}

	// Write _index.md.
	writeIndex(*outDir, posts)

	assetsDir := filepath.Join(*outDir, "assets")
	if *withMedia {
		if err := os.MkdirAll(assetsDir, 0o755); err != nil {
			return fmt.Errorf("create assets dir: %w", err)
		}
	}

	// Phase 3: process and write each post.
	imported, skipped, assetCount := 0, 0, 0
	for _, p := range posts {
		destPath := filepath.Join(*outDir, p.filename)

		if !*noSkip {
			if _, statErr := os.Stat(destPath); statErr == nil {
				skipped++
				continue
			}
		}

		msg := p.group.primary
		md := tgtd.Convert(msg)
		md = replaceTelegramLinks(md, postMap)

		// Collect asset links for all messages in the group.
		var assetLinks []string

		if *withMedia {
			var groupMedia []tgtd.DownloadedMedia
			downloadErr := client.RunWithAPI(ctx, sessionData, func(ctx context.Context, api *tg.Client) error {
				for _, m := range p.group.all {
					if m.Media == nil {
						continue
					}
					media, dlErr := tgtd.DownloadMessageMedia(ctx, api, m)
					if dlErr != nil {
						fmt.Fprintf(os.Stderr, "warn: download media for msg %d: %v\n", m.ID, dlErr)
						continue
					}
					groupMedia = append(groupMedia, media...)
				}
				return nil
			})
			if downloadErr != nil {
				fmt.Fprintf(os.Stderr, "warn: media download for msg %d: %v\n", msg.ID, downloadErr)
			}

			for idx, media := range groupMedia {
				ext := filepath.Ext(media.Filename)
				assetName := fmt.Sprintf("%d_%d%s", msg.ID, idx, ext)
				destAsset := filepath.Join(assetsDir, assetName)
				if saveErr := copyMedia(&groupMedia[idx], destAsset); saveErr != nil {
					fmt.Fprintf(os.Stderr, "warn: save asset %s: %v\n", assetName, saveErr)
				} else {
					assetLinks = append(assetLinks, fmt.Sprintf("![[assets/%s]]", assetName))
					assetCount++
				}
				media.Cleanup()
			}
		} else {
			// Placeholder links using media info (no download).
			for idx, info := range collectMediaInfo(p.group.all) {
				ext := filepath.Ext(info.Filename)
				assetName := fmt.Sprintf("%d_%d%s", msg.ID, idx, ext)
				assetLinks = append(assetLinks, fmt.Sprintf("![[assets/%s]]", assetName))
			}
		}

		if len(assetLinks) > 0 {
			md = strings.Join(assetLinks, "\n") + "\n\n" + md
		}

		content := frontmatter(*channelID, msg) + md
		if writeErr := os.WriteFile(destPath, []byte(content), 0o644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "warn: write %s: %v\n", p.filename, writeErr)
			continue
		}
		imported++
		if imported%10 == 0 || imported == len(posts)-skipped {
			fmt.Printf("  %d/%d written\n", imported, len(posts)-skipped)
		}
	}

	fmt.Printf("Done. %d exported, %d skipped, %d assets.\n", imported, skipped, assetCount)
	return nil
}

// postInfo holds pre-computed info for a single post.
type postInfo struct {
	group    *msgGroup
	filename string // e.g. "My title.md"
	title    string // e.g. "My title"
}

// msgGroup is a single logical post (may be a media group with multiple messages).
type msgGroup struct {
	primary *tg.Message
	all     []*tg.Message
}

func groupByMediaGroup(messages []*tg.Message) []*msgGroup {
	grouped := make(map[int64][]*tg.Message)
	var ungrouped []*tg.Message

	for _, m := range messages {
		gid, ok := m.GetGroupedID()
		if ok && gid != 0 {
			grouped[gid] = append(grouped[gid], m)
		} else {
			ungrouped = append(ungrouped, m)
		}
	}

	var result []*msgGroup

	for _, msgs := range grouped {
		var primary *tg.Message
		for _, m := range msgs {
			if m.Message != "" {
				primary = m
				break
			}
		}
		if primary == nil {
			primary = msgs[0]
		}
		result = append(result, &msgGroup{primary: primary, all: msgs})
	}
	for _, m := range ungrouped {
		result = append(result, &msgGroup{primary: m, all: []*tg.Message{m}})
	}
	return result
}

func collectMediaInfo(msgs []*tg.Message) []tgtd.MediaInfo {
	var out []tgtd.MediaInfo
	for _, m := range msgs {
		out = append(out, tgtd.GetMessageMediaInfo(m)...)
	}
	return out
}

func copyMedia(m *tgtd.DownloadedMedia, dest string) error {
	src, err := m.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func writeIndex(outDir string, posts []postInfo) {
	// Sort chronologically (oldest first) for the index.
	sorted := make([]postInfo, len(posts))
	copy(sorted, posts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].group.primary.Date < sorted[j].group.primary.Date
	})

	var sb strings.Builder
	for _, p := range sorted {
		t := time.Unix(int64(p.group.primary.Date), 0)
		sb.WriteString(t.Format("2006-01-02 15:00"))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("[[%s]]", p.title))
		sb.WriteString("\n\n")
	}

	_ = os.WriteFile(filepath.Join(outDir, "_index.md"), []byte(sb.String()), 0o644)
}

func frontmatter(channelID int64, msg *tg.Message) string {
	publishAt := time.Unix(int64(msg.Date), 0).Format(time.RFC3339)
	link := fmt.Sprintf("https://t.me/c/%d/%d", channelID, msg.ID)
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("telegram_publish_channel_id: \"%d\"\n", channelID))
	sb.WriteString(fmt.Sprintf("telegram_publish_message_id: %d\n", msg.ID))
	sb.WriteString(fmt.Sprintf("telegram_publish_message_link: %s\n", link))
	sb.WriteString(fmt.Sprintf("telegram_publish_at: %s\n", publishAt))
	sb.WriteString("telegram_import_allow_override: true\n")
	sb.WriteString("---\n\n")
	return sb.String()
}

// title / filename helpers (adapted from importtelegramchannel)

var (
	rCustomEmoji    = regexp.MustCompile(`!\[[^\]]*\]\((tg://emoji\?id=\d+|https://ce\.trip2g\.com/\d+\.webp)\)`)
	rMalformedEmoji = regexp.MustCompile(`!\[[^\]]*\]\(tg://emoji\?id=\d+\)>[^<]*</u>`)
	rMDLink         = regexp.MustCompile(`\[([^\]]*)\]\([^)]+\)`)
	rHTMLTag        = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	rLeadingJunk    = regexp.MustCompile(
		`^[\x{1F300}-\x{1F9FF}\x{1F3FB}-\x{1F3FF}\x{2600}-\x{26FF}` +
			`\x{2700}-\x{27BF}\x{25A0}-\x{25FF}\x{2B00}-\x{2BFF}` +
			`\x{FE00}-\x{FE0F}\x{200D}\s\-–—•·°№#@!?\.,;:\*"'«»„"'']+`,
	)
	rSafeFilename = regexp.MustCompile(`[^a-zA-Z0-9\p{Cyrillic} \-_.]`)
	rTGLink       = regexp.MustCompile(`\[([^\]]*)\]\(https?://t\.me/(?:[^/]+/)*(\d+)\)`)
)

func extractTitle(md string) string {
	text := rMalformedEmoji.ReplaceAllString(md, "")
	text = rCustomEmoji.ReplaceAllString(text, "")
	text = rHTMLTag.ReplaceAllString(text, "")
	text = rMDLink.ReplaceAllString(text, "$1")
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "`", "")
	text = strings.ReplaceAll(text, "||", "")
	text = strings.ReplaceAll(text, "~~", "")
	text = strings.ReplaceAll(text, `\[`, "[")
	text = strings.ReplaceAll(text, `\]`, "]")
	text = strings.ReplaceAll(text, `\*`, "*")
	text = strings.ReplaceAll(text, `\_`, "_")

	firstParagraph := text
	if idx := strings.Index(text, "\n\n"); idx != -1 {
		firstParagraph = text[:idx]
	}
	firstParagraph = strings.ReplaceAll(firstParagraph, "\n", " ")
	firstParagraph = rLeadingJunk.ReplaceAllString(firstParagraph, "")
	firstParagraph = strings.TrimSpace(firstParagraph)

	words := strings.Fields(firstParagraph)
	if len(words) > 7 {
		words = words[:7]
	}
	title := strings.Join(words, " ")
	title = rSafeFilename.ReplaceAllString(title, "")
	title = strings.Join(strings.Fields(title), " ")
	title = strings.TrimRight(title, ".,;:!?…-–—")
	return strings.TrimSpace(title)
}

func uniqueFilename(title string, msgID int, used map[string]bool) string {
	base := title + ".md"
	if !used[base] {
		return base
	}
	return fmt.Sprintf("%s (%d).md", title, msgID)
}

func replaceTelegramLinks(content string, postMap map[string]string) string {
	return rTGLink.ReplaceAllStringFunc(content, func(match string) string {
		sub := rTGLink.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		linkText, postID := sub[1], sub[2]
		if title, ok := postMap[postID]; ok {
			if linkText != "" && linkText != title {
				return fmt.Sprintf("[[%s|%s]]", title, linkText)
			}
			return fmt.Sprintf("[[%s]]", title)
		}
		return match
	})
}
