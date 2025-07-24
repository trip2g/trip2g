package patreon

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
)

const baseURL = "https://www.patreon.com/api/oauth2/v2"

type ClientConfig struct {
	CreatorAccessToken string
}

type Client struct {
	accessToken string
	http        *fasthttp.Client
	reqTimeout  time.Duration
}

type UnexpectedStatusCodeError struct {
	StatusCode int
	Body       string
}

func (e *UnexpectedStatusCodeError) Error() string {
	return fmt.Sprintf("unexpected status code: %d, body: %s", e.StatusCode, e.Body)
}

func NewClient(config ClientConfig) (*Client, error) {
	c := Client{
		accessToken: config.CreatorAccessToken,
		reqTimeout:  10 * time.Second,
		http: &fasthttp.Client{
			NoDefaultUserAgentHeader:      true,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
		},
	}

	return &c, nil
}

// CampaignID retrieves the first campaign ID associated with the creator's account.
func (c *Client) CampaignID() (string, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(baseURL + "/campaigns")
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := c.http.DoTimeout(req, resp, c.reqTimeout)
	if err != nil {
		return "", fmt.Errorf("failed to get campaigns: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return "", &UnexpectedStatusCodeError{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
		}
	}

	var respData CampaignsResponse

	err = easyjson.Unmarshal(resp.Body(), &respData)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(respData.Data) == 0 {
		return "", errors.New("no campaigns found")
	}

	return respData.Data[0].ID, nil
}

// CreateWebhook creates a new webhook for the campaign.
func (c *Client) CreateWebhook(campaignID string, webhookURL string, triggers []string) error {
	webhookData := WebhookRequest{
		Data: WebhookData{
			Type: "webhook",
			Attributes: WebhookAttributes{
				Triggers: triggers,
				URI:      webhookURL,
			},
			Relationships: WebhookRelationships{
				Campaign: RelationshipData{
					Data: map[string]string{
						"type": "campaign",
						"id":   campaignID,
					},
				},
			},
		},
	}

	body, err := easyjson.Marshal(webhookData)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook data: %w", err)
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(baseURL + "/webhooks")
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.SetBody(body)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = c.http.DoTimeout(req, resp, c.reqTimeout)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusCreated && resp.StatusCode() != fasthttp.StatusOK {
		var errResp ErrorResponse
		if unmarshalErr := easyjson.Unmarshal(resp.Body(), &errResp); unmarshalErr == nil && len(errResp.Errors) > 0 {
			return fmt.Errorf("patreon API error: %s - %s", errResp.Errors[0].Title, errResp.Errors[0].Detail)
		}
		return &UnexpectedStatusCodeError{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
		}
	}

	return nil
}

// ListWebhooks retrieves all webhooks for the authenticated user.
func (c *Client) ListWebhooks() ([]Webhook, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(baseURL + "/webhooks")
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := c.http.DoTimeout(req, resp, c.reqTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, &UnexpectedStatusCodeError{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
		}
	}

	var respData WebhooksResponse

	err = easyjson.Unmarshal(resp.Body(), &respData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return respData.Data, nil
}

// ListPatrons retrieves patrons for a campaign with specified includes and fields.
func (c *Client) ListPatrons(campaignID string, nextPageURL ...string) (*PatronsResponse, error) {
	var reqURL string

	if len(nextPageURL) > 0 && nextPageURL[0] != "" {
		reqURL = nextPageURL[0]
	} else {
		params := url.Values{}
		params.Set("include", "currently_entitled_tiers,user")
		params.Set("fields[member]", "patron_status,email")
		params.Set("fields[user]", "email,full_name")

		reqURL = fmt.Sprintf("%s/campaigns/%s/members?%s", baseURL, campaignID, params.Encode())
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(reqURL)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := c.http.DoTimeout(req, resp, c.reqTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to list patrons: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		var errResp ErrorResponse
		if unmarshalErr := easyjson.Unmarshal(resp.Body(), &errResp); unmarshalErr == nil && len(errResp.Errors) > 0 {
			return nil, fmt.Errorf("patreon API error: %s - %s", errResp.Errors[0].Title, errResp.Errors[0].Detail)
		}
		return nil, &UnexpectedStatusCodeError{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
		}
	}

	var respData PatronsResponse

	err = easyjson.Unmarshal(resp.Body(), &respData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &respData, nil
}

// DeleteWebhook deletes a webhook by ID.
func (c *Client) DeleteWebhook(webhookID string) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(fmt.Sprintf("%s/webhooks/%s", baseURL, webhookID))
	req.Header.SetMethod(fasthttp.MethodDelete)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := c.http.DoTimeout(req, resp, c.reqTimeout)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusNoContent && resp.StatusCode() != fasthttp.StatusOK {
		var errResp ErrorResponse
		if unmarshalErr := easyjson.Unmarshal(resp.Body(), &errResp); unmarshalErr == nil && len(errResp.Errors) > 0 {
			return fmt.Errorf("patreon API error: %s - %s", errResp.Errors[0].Title, errResp.Errors[0].Detail)
		}
		return &UnexpectedStatusCodeError{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
		}
	}

	return nil
}
