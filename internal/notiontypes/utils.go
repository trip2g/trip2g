package notiontypes

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"trip2g/internal/model"

	"github.com/tidwall/gjson"
)

// ExtractPageTitle extracts the title from a Notion page's properties
func ExtractPageTitle(page *Page) string {
	if page.Raw == nil {
		return "untitled"
	}

	rawJSON := string(page.Raw)

	// Look for title property (common names: "title", "Name", "Title")
	titleKeys := []string{"title", "Name", "Title"}

	for _, key := range titleKeys {
		path := fmt.Sprintf("properties.%s.title.0.plain_text", key)
		if title := gjson.Get(rawJSON, path); title.Exists() && title.String() != "" {
			return title.String()
		}
	}

	// Fallback: look for any property with type "title"
	var foundTitle string
	propertiesResult := gjson.Get(rawJSON, "properties")
	if propertiesResult.Exists() {
		propertiesResult.ForEach(func(key, value gjson.Result) bool {
			if gjson.Get(value.Raw, "type").String() == "title" {
				if title := gjson.Get(value.Raw, "title.0.plain_text"); title.Exists() && title.String() != "" {
					foundTitle = title.String()
					return false // found title, stop iteration
				}
			}
			return true // continue iteration
		})
	}

	if foundTitle != "" {
		return foundTitle
	}

	return "untitled"
}

// ExtractRawNote creates a model.RawNote from a Notion page and its content
func ExtractRawNote(page *Page, content *PageContent, basePath string) (*model.RawNote, error) {
	// Extract title and build path
	title := ExtractPageTitle(page)

	// Convert PageContent to JSON
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal page content to JSON: %w", err)
	}

	return &model.RawNote{
		Path:    filepath.Join(basePath, title+".notion.json"),
		Content: string(contentJSON),
	}, nil
}
