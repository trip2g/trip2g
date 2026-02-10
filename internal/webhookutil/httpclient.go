package webhookutil

import (
	"fmt"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	connectTimeout  = 5 * time.Second
	maxResponseBody = 1 << 20 // 1MB.
	userAgent       = "trip2g-webhooks/1.0"
)

// DeliveryResult holds the result of an HTTP webhook delivery.
type DeliveryResult struct {
	StatusCode int
	Body       []byte
	DurationMs int64
	Err        error
}

// Deliver sends an HTTP POST to the given URL with payload and headers.
func Deliver(client *fasthttp.Client, url string, payload []byte, headers map[string]string, timeout time.Duration) DeliveryResult {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.Header.Set("User-Agent", userAgent)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	req.SetBody(payload)

	start := time.Now()

	err := client.DoTimeout(req, resp, timeout)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		return DeliveryResult{
			DurationMs: durationMs,
			Err:        fmt.Errorf("HTTP request failed: %w", err),
		}
	}

	body := resp.Body()
	if len(body) > maxResponseBody {
		body = body[:maxResponseBody]
	}

	bodyClone := make([]byte, len(body))
	copy(bodyClone, body)

	return DeliveryResult{
		StatusCode: resp.StatusCode(),
		Body:       bodyClone,
		DurationMs: durationMs,
	}
}

// NewClient creates a fasthttp.Client with appropriate defaults.
func NewClient() *fasthttp.Client {
	return &fasthttp.Client{
		MaxConnWaitTimeout:  connectTimeout,
		MaxResponseBodySize: maxResponseBody,
	}
}
