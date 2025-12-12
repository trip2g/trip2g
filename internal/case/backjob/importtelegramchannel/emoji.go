package importtelegramchannel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"trip2g/internal/tgtd"
)

const ceEmojiBaseURL = "https://ce.trip2g.com"

// EmojiCache caches downloaded custom emojis to avoid re-downloading.
// Each emoji is downloaded once, but uploadAsset is called per-note (server deduplicates).
type EmojiCache struct {
	mu    sync.RWMutex
	cache map[string]*tgtd.DownloadedMedia // emojiID -> media
}

// NewEmojiCache creates a new emoji cache.
func NewEmojiCache() *EmojiCache {
	return &EmojiCache{
		cache: make(map[string]*tgtd.DownloadedMedia),
	}
}

// Get returns cached emoji or nil if not cached.
func (c *EmojiCache) Get(emojiID string) *tgtd.DownloadedMedia {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache[emojiID]
}

// Set caches the emoji.
func (c *EmojiCache) Set(emojiID string, media *tgtd.DownloadedMedia) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[emojiID] = media
}

// Cleanup removes all temp files.
func (c *EmojiCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, media := range c.cache {
		media.Cleanup()
	}
	c.cache = make(map[string]*tgtd.DownloadedMedia)
}

// downloadCustomEmoji downloads a custom emoji webp from ce.trip2g.com.
// Returns DownloadedMedia with temp file path. Caller must call Cleanup() when done.
func downloadCustomEmoji(ctx context.Context, emojiID string) (*tgtd.DownloadedMedia, error) {
	url := fmt.Sprintf("%s/%s.webp", ceEmojiBaseURL, emojiID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch emoji: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d for emoji %s", resp.StatusCode, emojiID)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "tg-emoji-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Use TeeReader to calculate hash while downloading
	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	size, err := io.Copy(writer, resp.Body)
	_ = tmpFile.Close()

	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("failed to download emoji: %w", err)
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	filename := fmt.Sprintf("tg_ce_%s.webp", emojiID)

	return &tgtd.DownloadedMedia{
		Filename:   filename,
		MimeType:   "image/webp",
		Sha256Hash: hash,
		Size:       size,
		TempPath:   tmpPath,
		IsImage:    true,
	}, nil
}

// downloadAllCustomEmojis downloads all unique custom emojis from content.
// Uses cache to avoid re-downloading the same emoji.
// Returns map of emojiID -> local filename and slice of media for upload.
// Note: media is NOT owned by caller - cache owns it and will cleanup at end of import.
func downloadAllCustomEmojis(ctx context.Context, content string, cache *EmojiCache) (map[string]string, []*tgtd.DownloadedMedia) {
	emojiIDs := extractCustomEmojiIDs(content)
	if len(emojiIDs) == 0 {
		return nil, nil
	}

	emojiMap := make(map[string]string)
	var mediaList []*tgtd.DownloadedMedia

	for _, emojiID := range emojiIDs {
		// Check cache first
		media := cache.Get(emojiID)
		if media == nil {
			// Download and cache
			var err error
			media, err = downloadCustomEmoji(ctx, emojiID)
			if err != nil {
				// Skip failed downloads - emoji will keep original URL
				continue
			}
			cache.Set(emojiID, media)
		}

		emojiMap[emojiID] = media.Filename
		mediaList = append(mediaList, media)
	}

	return emojiMap, mediaList
}
