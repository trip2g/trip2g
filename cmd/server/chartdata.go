package main

import (
	"context"
	"database/sql"
	"errors"

	"trip2g/internal/case/backjob/refreshchartdata"
	"trip2g/internal/chartdata"
	"trip2g/internal/db"
)

// app implements chartdata.Env; the *chartdata.Service it embeds promotes
// ChartData (renderer provider) and SaveChartData (job sink) onto the app.
var _ chartdata.Env = (*app)(nil)

func (a *app) CachedChartData(ctx context.Context, versionID int64, hash string) (string, string, bool, error) {
	row, err := a.Queries.GetChartData(ctx, db.GetChartDataParams{VersionID: versionID, ChartHash: hash})
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return row.DataJson, row.LastError, true, nil
}

func (a *app) StoreChartDataError(ctx context.Context, versionID int64, hash, errMsg string, erroredAt int64) error {
	return a.WriteQueries.SetChartDataError(ctx, db.SetChartDataErrorParams{
		VersionID:   versionID,
		ChartHash:   hash,
		LastError:   errMsg,
		LastErrorAt: erroredAt,
	})
}

func (a *app) StoreChartData(ctx context.Context, versionID int64, hash, dataJSON string, fetchedAt int64) error {
	return a.WriteQueries.UpsertChartData(ctx, db.UpsertChartDataParams{
		VersionID: versionID,
		ChartHash: hash,
		DataJson:  dataJSON,
		FetchedAt: fetchedAt,
	})
}

func (a *app) EnqueueChartDataRefresh(ctx context.Context, versionID int64, hash, url, body string) error {
	return a.refreshChartDataJob.EnqueueRefreshChartData(ctx, refreshchartdata.Params{
		VersionID: versionID,
		Hash:      hash,
		URL:       url,
		Body:      body,
	})
}

// ReloadChartDataNotes re-renders whichever note set serves public pages, the
// same way LiveNoteViews chooses it (latest under show_draft_versions, else live).
func (a *app) ReloadChartDataNotes(ctx context.Context) error {
	if a.SiteConfig(ctx).ShowDraftVersions {
		_, err := a.PrepareLatestNotes(ctx, true)
		return err
	}
	_, err := a.PrepareLiveNotes(ctx)
	return err
}
