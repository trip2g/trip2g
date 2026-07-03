package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"trip2g/internal/db"
	"trip2g/internal/logger"

	"github.com/stretchr/testify/require"
)

func TestHandleDebugWriteHolder(t *testing.T) {
	holder := db.NewWriteHolder(true)
	a := &app{
		appState: &appState{
			log:         &logger.TestLogger{},
			writeHolder: holder,
		},
	}

	type response struct {
		Held    bool `json:"held"`
		Holders []struct {
			Label   string `json:"label"`
			HeldFor string `json:"held_for"`
			Stack   string `json:"stack"`
		} `json:"holders"`
	}

	get := func() response {
		rec := httptest.NewRecorder()
		a.handleDebugWriteHolder(rec, httptest.NewRequest(http.MethodGet, "/debug/write-holder", nil))
		require.Equal(t, 200, rec.Code)
		var resp response
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		return resp
	}

	resp := get()
	require.False(t, resp.Held)
	require.Empty(t, resp.Holders)

	release := holder.Acquire("tx pushNotes")
	resp = get()
	require.True(t, resp.Held)
	require.Len(t, resp.Holders, 1)
	require.Equal(t, "tx pushNotes", resp.Holders[0].Label)
	require.NotEmpty(t, resp.Holders[0].HeldFor)
	require.Contains(t, resp.Holders[0].Stack, "TestHandleDebugWriteHolder")

	release()
	resp = get()
	require.False(t, resp.Held)
}
