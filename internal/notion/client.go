package notion

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/valyala/fasthttp"

	"trip2g/internal/notiontypes"
)

type Config struct {
	Token          string
	RequestTimeout time.Duration
}

type clientImpl struct {
	config Config
	http   *fasthttp.Client
}

func DefaultConfig() Config {
	return Config{
		Token:          "",
		RequestTimeout: 10 * time.Second,
	}
}

func New(config Config) (notiontypes.Client, error) {
	client := clientImpl{
		config: config,
		http: &fasthttp.Client{
			ReadTimeout:  config.RequestTimeout,
			WriteTimeout: config.RequestTimeout,
		},
	}

	return &client, nil
}

func (c *clientImpl) AllPages() ([]*notiontypes.Page, error) {
	var allPages []*notiontypes.Page
	var cursor *string

	for {
		searchReq := notiontypes.SearchRequest{
			Filter: &notiontypes.SearchFilter{
				Property: "object",
				Value:    "page",
			},
			PageSize:    100,
			StartCursor: cursor,
		}

		response, err := c.search(searchReq)
		if err != nil {
			return nil, fmt.Errorf("failed to search pages: %w", err)
		}

		allPages = append(allPages, response.Results...)

		if !response.HasMore {
			break
		}

		cursor = response.NextCursor
	}

	return allPages, nil
}

func (c *clientImpl) GetPage(pageID string) (*notiontypes.Page, error) {
	httpReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(httpReq)

	httpResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(httpResp)

	httpReq.SetRequestURI(fmt.Sprintf("https://api.notion.com/v1/pages/%s", pageID))
	httpReq.Header.SetMethod(fasthttp.MethodGet)
	httpReq.Header.Set("Authorization", "Bearer "+c.config.Token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Notion-Version", "2022-06-28")

	err := c.http.DoTimeout(httpReq, httpResp, c.config.RequestTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}

	if httpResp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", httpResp.StatusCode(), string(httpResp.Body()))
	}

	var page notiontypes.Page
	err = json.Unmarshal(httpResp.Body(), &page)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Store raw JSON
	page.Raw = make([]byte, len(httpResp.Body()))
	copy(page.Raw, httpResp.Body())

	return &page, nil
}

func (c *clientImpl) GetPageContent(pageID string) (*notiontypes.PageContent, error) {
	var allBlocks []*notiontypes.Block
	var cursor *string

	for {
		uri := fmt.Sprintf("https://api.notion.com/v1/blocks/%s/children", pageID)
		if cursor != nil {
			uri += "?start_cursor=" + *cursor
		}

		httpReq := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(httpReq)

		httpResp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(httpResp)

		httpReq.SetRequestURI(uri)
		httpReq.Header.SetMethod(fasthttp.MethodGet)
		httpReq.Header.Set("Authorization", "Bearer "+c.config.Token)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Notion-Version", "2022-06-28")

		err := c.http.DoTimeout(httpReq, httpResp, c.config.RequestTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to make HTTP request: %w", err)
		}

		if httpResp.StatusCode() != fasthttp.StatusOK {
			return nil, fmt.Errorf("API request failed with status %d: %s", httpResp.StatusCode(), string(httpResp.Body()))
		}

		// First unmarshal into raw map to get individual block data
		var rawResponse map[string]interface{}
		err = json.Unmarshal(httpResp.Body(), &rawResponse)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		var pageResponse notiontypes.PageContent
		err = json.Unmarshal(httpResp.Body(), &pageResponse)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// Store raw JSON for each block in this page
		if results, ok := rawResponse["results"].([]interface{}); ok {
			for i, result := range results {
				if i < len(pageResponse.Results) && result != nil {
					blockJSON, marshalErr := json.Marshal(result)
					if marshalErr == nil {
						pageResponse.Results[i].RawContent = blockJSON
					}
				}
			}
		}

		allBlocks = append(allBlocks, pageResponse.Results...)

		if !pageResponse.HasMore {
			break
		}

		cursor = pageResponse.NextCursor
	}

	return &notiontypes.PageContent{
		Object:     "list",
		Results:    allBlocks,
		NextCursor: nil,
		HasMore:    false,
	}, nil
}

func (c *clientImpl) search(req notiontypes.SearchRequest) (*notiontypes.SearchResponse, error) {
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(httpReq)

	httpResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(httpResp)

	httpReq.SetRequestURI("https://api.notion.com/v1/search")
	httpReq.Header.SetMethod(fasthttp.MethodPost)
	httpReq.Header.Set("Authorization", "Bearer "+c.config.Token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Notion-Version", "2022-06-28")
	httpReq.SetBody(requestBody)

	err = c.http.DoTimeout(httpReq, httpResp, c.config.RequestTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}

	if httpResp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", httpResp.StatusCode(), string(httpResp.Body()))
	}

	// First unmarshal into raw map to preserve individual page data
	var rawResponse map[string]interface{}
	err = json.Unmarshal(httpResp.Body(), &rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var response notiontypes.SearchResponse
	err = json.Unmarshal(httpResp.Body(), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Store raw JSON for each page
	if results, ok := rawResponse["results"].([]interface{}); ok {
		for i, result := range results {
			if i < len(response.Results) && result != nil {
				pageJSON, marshalErr := json.Marshal(result)
				if marshalErr == nil {
					response.Results[i].Raw = pageJSON
				}
			}
		}
	}

	return &response, nil
}
