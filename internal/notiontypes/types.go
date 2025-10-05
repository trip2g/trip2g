package notiontypes

import "time"

type Client interface {
	AllPages() ([]*Page, error)
	GetPage(pageID string) (*Page, error)
	GetPageContent(pageID string) (*PageContent, error)
}

type Page struct {
	Object         string                 `json:"object"`
	ID             string                 `json:"id"`
	CreatedTime    time.Time              `json:"created_time"`
	LastEditedTime time.Time              `json:"last_edited_time"`
	CreatedBy      *User                  `json:"created_by"`
	LastEditedBy   *User                  `json:"last_edited_by"`
	Cover          *File                  `json:"cover"`
	Icon           *Icon                  `json:"icon"`
	Parent         *Parent                `json:"parent"`
	Archived       bool                   `json:"archived"`
	Properties     map[string]interface{} `json:"properties"`
	URL            string                 `json:"url"`
	Raw            []byte                 `json:"-"`
}

type User struct {
	Object string `json:"object"`
	ID     string `json:"id"`
}

type File struct {
	Type string `json:"type"`
	File *struct {
		URL        string     `json:"url"`
		ExpiryTime *time.Time `json:"expiry_time"`
	} `json:"file"`
	External *struct {
		URL string `json:"url"`
	} `json:"external"`
}

type Icon struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
	File  *File  `json:"file"`
}

type Parent struct {
	Type       string `json:"type"`
	DatabaseID string `json:"database_id"`
	PageID     string `json:"page_id"`
	Workspace  bool   `json:"workspace"`
}

type SearchResponse struct {
	Object     string  `json:"object"`
	Results    []*Page `json:"results"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
	Type       string  `json:"type"`
	RequestID  string  `json:"request_id"`
}

type SearchRequest struct {
	Filter      *SearchFilter `json:"filter,omitempty"`
	PageSize    int           `json:"page_size,omitempty"`
	StartCursor *string       `json:"start_cursor,omitempty"`
}

type SearchFilter struct {
	Property string `json:"property"`
	Value    string `json:"value"`
}

type PageContent struct {
	Object     string   `json:"object"`
	Results    []*Block `json:"results"`
	NextCursor *string  `json:"next_cursor"`
	HasMore    bool     `json:"has_more"`
}

type Block struct {
	Object         string    `json:"object"`
	ID             string    `json:"id"`
	Parent         *Parent   `json:"parent"`
	CreatedTime    time.Time `json:"created_time"`
	LastEditedTime time.Time `json:"last_edited_time"`
	CreatedBy      *User     `json:"created_by"`
	LastEditedBy   *User     `json:"last_edited_by"`
	HasChildren    bool      `json:"has_children"`
	Archived       bool      `json:"archived"`
	Type           string    `json:"type"`
	RawContent     []byte    `json:"-"`
}
