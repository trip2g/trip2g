package boosty

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
)

//go:generate go tool github.com/matryer/moq -out mocks.go . Client

const (
	baseURL       = "https://api.boosty.to/v1"
	oauthTokenURL = "https://api.boosty.to/oauth/token/"
)

type AuthData struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    string `json:"expiresAt"`
	DeviceID     string `json:"deviceId"`
	BlogName     string `json:"blogName"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type Client interface {
	Subscribers() ([]Subscriber, error)
	RefreshToken() (*RefreshTokenResponse, error)
}

type ClientImpl struct {
	authData   AuthData
	http       *fasthttp.Client
	reqTimeout time.Duration
}

type UnexpectedStatusCodeError struct {
	StatusCode int
	Body       string
}

func (e *UnexpectedStatusCodeError) Error() string {
	return fmt.Sprintf("unexpected status code: %d, body: %s", e.StatusCode, e.Body)
}

func NewClient(authData AuthData) (*ClientImpl, error) {
	c := ClientImpl{
		authData:   authData,
		reqTimeout: 10 * time.Second,
		http: &fasthttp.Client{
			NoDefaultUserAgentHeader:      true,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
		},
	}

	return &c, nil
}

type Subscriber struct {
	HasAvatar bool `json:"hasAvatar"`
	Payments  int  `json:"payments"`
	Level     struct {
		Deleted bool   `json:"deleted"`
		Name    string `json:"name"`
		Price   int    `json:"price"`
		OwnerID int    `json:"ownerId"`
		Data    []struct {
			Type        string `json:"type"`
			Content     string `json:"content"`
			Modificator string `json:"modificator"`
		} `json:"data"`
		ID             int `json:"id"`
		CurrencyPrices struct {
			RUB int     `json:"RUB"`
			USD float64 `json:"USD"`
		} `json:"currencyPrices"`
		CreatedAt  int  `json:"createdAt"`
		IsArchived bool `json:"isArchived"`
	} `json:"level"`
	Email         string `json:"email"`
	IsBlackListed bool   `json:"isBlackListed"`
	ID            int    `json:"id"`
	Name          string `json:"name"`
	OnTime        int    `json:"onTime"`
	Subscribed    bool   `json:"subscribed"`
	NextPayTime   int    `json:"nextPayTime"`
	Price         int    `json:"price"`
	AvatarURL     string `json:"avatarUrl"`
}

type SubscribersResponse struct {
	Data   []Subscriber `json:"data"`
	Offset int          `json:"offset"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
}

// Subscribers retrieves all subscribers for the configured blog across all pages.
func (c *ClientImpl) Subscribers() ([]Subscriber, error) {
	sortBy := "on_time"
	limit := 11
	order := "gt"

	var allSubscribers []Subscriber
	offset := 0

	for {
		pageResponse, err := c.fetchSubscribersPage(sortBy, limit, order, offset)
		if err != nil {
			return nil, err
		}

		allSubscribers = append(allSubscribers, pageResponse.Data...)

		// Check if we've reached the end
		if len(pageResponse.Data) < limit || offset+len(pageResponse.Data) >= pageResponse.Total {
			break
		}

		offset += len(pageResponse.Data)
	}

	return allSubscribers, nil
}

// fetchSubscribersPage retrieves a single page of subscribers.
func (c *ClientImpl) fetchSubscribersPage(sortBy string, limit int, order string, offset int) (*SubscribersResponse, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	params := url.Values{}
	params.Set("sort_by", sortBy)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("order", order)
	params.Set("offset", strconv.Itoa(offset))

	reqURL := fmt.Sprintf("%s/blog/%s/subscribers?%s", baseURL, c.authData.BlogName, params.Encode())
	req.SetRequestURI(reqURL)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Set("Authorization", "Bearer "+c.authData.AccessToken)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("X-App", "web")
	req.Header.Set("X-Currency", "RUB")
	req.Header.Set("X-Locale", "ru_RU")
	req.Header.Set("X-Referer", "boosty.to")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := c.http.DoTimeout(req, resp, c.reqTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscribers: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, &UnexpectedStatusCodeError{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
		}
	}

	var respData SubscribersResponse

	err = json.Unmarshal(resp.Body(), &respData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &respData, nil
}

// RefreshToken refreshes the access token using the refresh token.
func (c *ClientImpl) RefreshToken() (*RefreshTokenResponse, error) {
	if c.authData.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token was found to refresh auth data")
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	// Create form data exactly like the JavaScript version
	formData := url.Values{}
	formData.Set("device_id", c.authData.DeviceID)
	formData.Set("grant_type", "refresh_token")
	formData.Set("refresh_token", c.authData.RefreshToken)
	formData.Set("device_os", "web")

	req.SetRequestURI(oauthTokenURL)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", "https://boosty.to")
	req.Header.Set("Referer", fmt.Sprintf("https://boosty.to/%s/blog/statistics/subscribers", c.authData.BlogName))
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("X-App", "web")
	req.Header.Set("X-Currency", "USD")
	req.Header.Set("X-From-Id", c.authData.DeviceID)
	req.Header.Set("X-Locale", "ru_RU")
	req.SetBodyString(formData.Encode())
	req.Header.Set("Authorization", "Bearer "+c.authData.AccessToken)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := c.http.DoTimeout(req, resp, c.reqTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, &UnexpectedStatusCodeError{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
		}
	}

	var result RefreshTokenResponse
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal refresh token response: %w", err)
	}

	// Update internal auth data with new tokens
	c.authData.AccessToken = result.AccessToken
	c.authData.RefreshToken = result.RefreshToken

	return &result, nil
}
