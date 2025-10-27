package nowpayments

import (
	"encoding/json"
	"fmt"
	"time"
	"trip2g/internal/logger"

	"github.com/valyala/fasthttp"
)

const baseURL = "https://api.nowpayments.io/v1"

type Client struct {
	apiKey string
	http   *fasthttp.Client

	reqTimeout time.Duration

	logger logger.Logger
}

type UnexpectedStatusCodeError struct {
	StatusCode int
}

func (e *UnexpectedStatusCodeError) Error() string {
	return fmt.Sprintf("unexpected status code: %d", e.StatusCode)
}

func NewClient(apiKey string, l logger.Logger) (*Client, error) {
	client := Client{
		apiKey:     apiKey,
		reqTimeout: 5 * time.Second,
		logger:     logger.WithPrefix(l, "nowpayments"),
		http: &fasthttp.Client{
			NoDefaultUserAgentHeader:      true,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
		},
	}

	err := client.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	// TODO: maybe need to check if the API key is working

	return &client, nil
}

func (client *Client) Status() error {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(baseURL + "/status")
	req.Header.SetMethod(fasthttp.MethodGet)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := client.http.Do(req, resp)
	fasthttp.ReleaseRequest(req)

	if err != nil {
		return fmt.Errorf("failed to get /status: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return &UnexpectedStatusCodeError{StatusCode: resp.StatusCode()}
	}

	var respData struct {
		Message string
	}

	err = json.Unmarshal(resp.Body(), &respData) //nolint:musttag // internal API response
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if respData.Message != "OK" {
		return fmt.Errorf("status not OK: %s", string(resp.Body()))
	}

	return nil
}

type CreateInvoiceParams struct {
	PriceAmount      float64 `json:"price_amount"`
	PriceCurrency    string  `json:"price_currency"`
	OrderID          string  `json:"order_id"`
	OrderDescription string  `json:"order_description"`
	IPNCallbackURL   string  `json:"ipn_callback_url"`
	SuccessURL       string  `json:"success_url"`
	CancelURL        string  `json:"cancel_url"`
}

type CreateInvoiceResponse struct {
	ID               string    `json:"id"`
	OrderID          string    `json:"order_id"`
	OrderDescription string    `json:"order_description"`
	PriceAmount      string    `json:"price_amount"`
	PriceCurrency    string    `json:"price_currency"`
	PayCurrency      *string   `json:"pay_currency"`
	IPNCallbackURL   string    `json:"ipn_callback_url"`
	InvoiceURL       string    `json:"invoice_url"`
	SuccessURL       string    `json:"success_url"`
	CancelURL        string    `json:"cancel_url"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (client *Client) CreateInvoice(params CreateInvoiceParams) (*CreateInvoiceResponse, error) {
	reqBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(baseURL + "/invoice")
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", client.apiKey)
	req.SetBodyRaw(reqBody)
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err = client.http.DoTimeout(req, resp, client.reqTimeout)
	fasthttp.ReleaseRequest(req)

	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		client.logger.Error("unexpected status code", "status_code", resp.StatusCode(), "body", string(resp.Body()))

		return nil, &UnexpectedStatusCodeError{StatusCode: resp.StatusCode()}
	}

	var respData CreateInvoiceResponse

	err = json.Unmarshal(resp.Body(), &respData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &respData, nil
}
