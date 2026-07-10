package main

import (
	"context"
	"testing"
	"time"

	"trip2g/internal/mdloader"
	"trip2g/internal/metrics"
	"trip2g/internal/noteloader"
	"trip2g/internal/pagecache"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// mcpToolNoteLoaderEnv loads one note exposing a dynamic MCP tool via mcp_method.
type mcpToolNoteLoaderEnv struct{ emptyNoteLoaderEnv }

func (mcpToolNoteLoaderEnv) RawNotes(context.Context) ([]noteloader.RawNote, error) {
	return []noteloader.RawNote{{
		Path:      "tools/my.md",
		PathID:    1,
		VersionID: 1,
		Content:   "---\nmcp_method: my_tool\n---\nbody",
		CreatedAt: time.Now(),
	}}, nil
}

func gaugeValue(t *testing.T, reg *prometheus.Registry, name string) float64 {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range families {
		if mf.GetName() == name {
			require.Len(t, mf.GetMetric(), 1)
			return mf.GetMetric()[0].GetGauge().GetValue()
		}
	}
	var zero *dto.Metric
	require.NotNil(t, zero, "gauge %s not found in registry", name)
	return 0
}

// TestMCPDynamicToolsGauge_FreshAfterReload pins the wiring: the gauge reads
// the currently published note snapshot at scrape time, so a notes reload is
// reflected immediately, without requiring any tools/list MCP call.
func TestMCPDynamicToolsGauge_FreshAfterReload(t *testing.T) {
	reg := prometheus.NewRegistry()
	a := &app{appState: &appState{
		pageCache:  pagecache.New(),
		mcpMetrics: metrics.NewMCPMetrics(reg),
	}}
	a.latestNoteLoader = noteloader.New("latest", mcpToolNoteLoaderEnv{}, mdloader.Config{})
	a.wireMCPDynamicToolsGauge()

	require.InDelta(t, 0, gaugeValue(t, reg, "trip2g_mcp_dynamic_tools_registered"), 1e-9,
		"no notes loaded yet")

	_, err := a.PrepareLatestNotes(context.Background(), true)
	require.NoError(t, err)

	require.InDelta(t, 1, gaugeValue(t, reg, "trip2g_mcp_dynamic_tools_registered"), 1e-9,
		"scrape after reload must reflect the current note index")
}
