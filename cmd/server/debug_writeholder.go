package main

import (
	"encoding/json"
	"net/http"
)

// handleDebugWriteHolder reports who currently occupies the single write
// connection: open write transactions and in-flight write statements, oldest
// first (the oldest is the actual holder; younger ones queue behind it).
// Dev-only: registered on the internal debug mux only when DevMode is set,
// and the underlying WriteHolder captures nothing otherwise.
func (a *app) handleDebugWriteHolder(w http.ResponseWriter, _ *http.Request) {
	type holderJSON struct {
		Label   string `json:"label"`
		HeldFor string `json:"held_for"`
		Stack   string `json:"stack"`
	}
	type response struct {
		Held    bool         `json:"held"`
		Holders []holderJSON `json:"holders,omitempty"`
	}

	infos := a.writeHolder.Snapshot()
	resp := response{Held: len(infos) > 0}
	for _, info := range infos {
		resp.Holders = append(resp.Holders, holderJSON{
			Label:   info.Label,
			HeldFor: info.HeldFor.String(),
			Stack:   info.Stack,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		a.log.Error("failed to encode write-holder response", "error", err)
	}
}
