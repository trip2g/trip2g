package checkhealth

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"trip2g/internal/graph/model"
)

// 14 KB fits in the TCP initial congestion window (~10 segments × 1460 bytes),
// so the browser receives the full page in a single round trip without waiting for ACKs.
const homepageSizeLimitBytes = 14 * 1024

type HomepageSizeChecker struct{}

func (c *HomepageSizeChecker) ID() string {
	return "homepage_size"
}

func (c *HomepageSizeChecker) Check(ctx context.Context, env Env) model.HealchCheck {
	publicURL := env.GetPublicURLForRequest(ctx)
	if publicURL == "" {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusWarning,
			Description: "Unable to determine public URL for request",
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, publicURL+"/", nil)
	if err != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to create request: %v", err),
		}
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to fetch homepage: %v", err),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.HealchCheck{
			ID:          c.ID(),
			Status:      model.HealthCheckStatusCritical,
			Description: fmt.Sprintf("Failed to read response: %v", err),
		}
	}

	size := len(body)
	if size > homepageSizeLimitBytes {
		return model.HealchCheck{
			ID:     c.ID(),
			Status: model.HealthCheckStatusWarning,
			Description: fmt.Sprintf(
				"Homepage gzip size %d KB exceeds %d KB — page won't fit in TCP initial congestion window, requires extra round trips",
				size/1024,
				homepageSizeLimitBytes/1024,
			),
		}
	}

	return model.HealchCheck{
		ID:          c.ID(),
		Status:      model.HealthCheckStatusOk,
		Description: fmt.Sprintf("Homepage gzip size is %d KB — fits in TCP initial congestion window, loads in a single round trip", size/1024),
	}
}
