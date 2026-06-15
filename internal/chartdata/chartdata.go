// Package chartdata orchestrates server-side data for url/internal datachart
// sources: it serves cached rows to the renderer, enqueues a background refresh
// on a cache miss, and re-renders notes (debounced) when fresh data lands.
//
// It follows the gitapi pattern: it declares the dependencies it needs as Env
// and the host app implements them. The app embeds *Service, so ChartData and
// SaveChartData are promoted onto the app without proxy methods.
package chartdata

import (
	"context"
	"encoding/json"
	"time"

	"trip2g/internal/logger"
	"trip2g/internal/model"
)

// reloadDebounce coalesces a burst of cache writes (e.g. on startup render) into
// a single re-render.
const reloadDebounce = 2 * time.Second

type Env interface {
	Logger() logger.Logger
	Now() time.Time

	// CachedChartData returns the cached rows JSON for (versionID, hash);
	// found is false on a cache miss. lastError is non-empty when the last
	// fetch failed.
	CachedChartData(ctx context.Context, versionID int64, hash string) (data string, lastError string, found bool, err error)
	// StoreChartData persists fetched rows for (versionID, hash) with a timestamp.
	StoreChartData(ctx context.Context, versionID int64, hash, dataJSON string, fetchedAt int64) error
	// StoreChartDataError records a fetch failure for (versionID, hash).
	StoreChartDataError(ctx context.Context, versionID int64, hash, errMsg string, erroredAt int64) error
	// EnqueueChartDataRefresh schedules a background fetch of a url chart's data.
	EnqueueChartDataRefresh(ctx context.Context, versionID int64, hash, url, body string) error
	// ReloadChartDataNotes re-renders the note set that serves public pages so
	// charts pick up freshly fetched rows.
	ReloadChartDataNotes(ctx context.Context) error
}

type ChartData struct {
	env    Env
	logger logger.Logger
	reload chan struct{}
}

func New(env Env) *ChartData {
	s := &ChartData{
		env:    env,
		logger: logger.WithPrefix(env.Logger(), "chartdata:"),
		reload: make(chan struct{}, 1),
	}
	go s.reloadLoop()
	return s
}

// ChartRows returns cached rows for a url/internal chart, enqueuing a background
// refresh on a miss for url sources. Implements mdloader.ChartDataProvider.
func (s *ChartData) ChartRows(versionID int64, chart model.NoteViewChart) model.ChartRowsResult {
	ctx := context.Background()

	data, lastError, found, err := s.env.CachedChartData(ctx, versionID, chart.Hash)
	if err != nil {
		s.logger.Warn("cache read failed", "hash", chart.Hash, "err", err)
		return model.ChartRowsResult{}
	}
	if found {
		var rows json.RawMessage
		if data != "" {
			rows = json.RawMessage(data)
		}
		return model.ChartRowsResult{Rows: rows, Error: lastError}
	}

	// Miss: enqueue a refresh for url sources, off the render thread so the DB
	// write does not block or contend with rendering.
	if chart.Data.Source == model.ChartSourceURL {
		v, c := versionID, chart
		go func() {
			if enqErr := s.env.EnqueueChartDataRefresh(context.Background(), v, c.Hash, c.Data.URL, c.Data.Body); enqErr != nil {
				s.logger.Warn("refresh enqueue failed", "hash", c.Hash, "err", enqErr)
			}
		}()
	}
	return model.ChartRowsResult{}
}

// SaveChartDataError records a fetch failure and signals a debounced re-render.
// Implements refreshchartdata.Env (called by the background job on fetch errors).
func (s *ChartData) SaveChartDataError(ctx context.Context, versionID int64, hash, errMsg string) error {
	if err := s.env.StoreChartDataError(ctx, versionID, hash, errMsg, s.env.Now().Unix()); err != nil {
		return err
	}
	s.signalReload()
	return nil
}

// SaveChartData caches fetched rows and signals a debounced re-render.
// Implements refreshchartdata.Env (called by the background job).
func (s *ChartData) SaveChartData(ctx context.Context, versionID int64, hash, dataJSON string) error {
	if err := s.env.StoreChartData(ctx, versionID, hash, dataJSON, s.env.Now().Unix()); err != nil {
		return err
	}
	s.signalReload()
	return nil
}

func (s *ChartData) signalReload() {
	select {
	case s.reload <- struct{}{}:
	default:
	}
}

// reloadLoop coalesces bursts of cache writes into a single debounced re-render.
func (s *ChartData) reloadLoop() {
	for range s.reload {
		time.Sleep(reloadDebounce)
		// Drain any extra signals that arrived during the debounce window.
		select {
		case <-s.reload:
		default:
		}
		if err := s.env.ReloadChartDataNotes(context.Background()); err != nil {
			s.logger.Warn("notes reload failed", "err", err)
		}
	}
}
