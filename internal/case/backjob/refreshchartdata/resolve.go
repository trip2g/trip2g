package refreshchartdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"trip2g/internal/logger"
)

// maxChartDataBytes caps the response we read into memory.
const maxChartDataBytes = 5 << 20 // 5 MiB

const fetchTimeout = 30 * time.Second

// Params identifies a chart's cache row and how to (re)fetch its data.
type Params struct {
	VersionID int64  `json:"version_id"`
	Hash      string `json:"hash"`
	URL       string `json:"url"`
	Body      string `json:"body,omitempty"` // POSTed when non-empty (e.g. {"sql":...})
}

type Env interface {
	Logger() logger.Logger
	// SaveChartData persists the fetched rows JSON for (versionID, hash). The
	// implementation stamps fetched_at.
	SaveChartData(ctx context.Context, versionID int64, hash, dataJSON string) error
}

// Resolve fetches the chart's data from its HTTP-JSON endpoint and caches it.
func Resolve(ctx context.Context, env Env, p Params) error {
	data, err := fetch(ctx, p.URL, p.Body)
	if err != nil {
		return fmt.Errorf("refreshchartdata: %w", err)
	}
	if !json.Valid(data) {
		return fmt.Errorf("refreshchartdata: %q returned non-JSON", p.URL)
	}
	return env.SaveChartData(ctx, p.VersionID, p.Hash, string(data))
}

func fetch(ctx context.Context, url, body string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	method := http.MethodGet
	var reader io.Reader
	if body != "" {
		method = http.MethodPost
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(reqCtx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxChartDataBytes))
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%q status %d", url, resp.StatusCode)
	}
	return data, nil
}
