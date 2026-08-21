package graph

import (
	"encoding/json"
	"time"

	"trip2g/internal/graph/model"
)

// storedDeliveryLog mirrors webhookutil.AgentLog as it was stored. Data stays a
// RawMessage all the way to the client: trip2g has no opinion about what an
// agent puts there, and re-encoding a map would reorder and reshape it.
type storedDeliveryLog struct {
	TS    *time.Time      `json:"ts"`
	Level string          `json:"level"`
	Msg   string          `json:"msg"`
	Data  json.RawMessage `json:"data"`
}

// decodeDeliveryLogs turns a delivery's stored run log into API entries. A
// malformed log yields an empty list rather than an error, for the same reason
// decodeCosts does: a careless agent must not break the page that shows its run.
func decodeDeliveryLogs(raw *string) []model.AdminDeliveryLog {
	if raw == nil || *raw == "" {
		return []model.AdminDeliveryLog{}
	}
	var stored []storedDeliveryLog
	if err := json.Unmarshal([]byte(*raw), &stored); err != nil {
		return []model.AdminDeliveryLog{}
	}

	out := make([]model.AdminDeliveryLog, 0, len(stored))
	for _, e := range stored {
		entry := model.AdminDeliveryLog{Ts: e.TS, Level: e.Level, Msg: e.Msg}
		if len(e.Data) > 0 {
			data := string(e.Data)
			entry.Data = &data
		}
		out = append(out, entry)
	}
	return out
}
