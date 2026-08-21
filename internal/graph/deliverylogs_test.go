package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/ptr"
)

func TestDecodeDeliveryLogs(t *testing.T) {
	t.Run("nil and empty yield no entries", func(t *testing.T) {
		require.Empty(t, decodeDeliveryLogs(nil))
		require.Empty(t, decodeDeliveryLogs(ptr.To("")))
	})

	t.Run("garbage never reaches the screen as an error", func(t *testing.T) {
		require.Empty(t, decodeDeliveryLogs(ptr.To("not json")))
		require.Empty(t, decodeDeliveryLogs(ptr.To(`{"not":"an array"}`)))
	})

	t.Run("entries keep their data bag verbatim", func(t *testing.T) {
		got := decodeDeliveryLogs(ptr.To(
			`[{"ts":"2026-08-21T10:00:01Z","level":"warn","msg":"denied","data":{"tool":"write_note","outcome":"denied"}}]`))

		require.Len(t, got, 1)
		require.Equal(t, "warn", got[0].Level)
		require.Equal(t, "denied", got[0].Msg)
		require.NotNil(t, got[0].Ts)
		require.NotNil(t, got[0].Data)
		require.JSONEq(t, `{"tool":"write_note","outcome":"denied"}`, *got[0].Data)
	})

	t.Run("an entry without a data bag is still an entry", func(t *testing.T) {
		got := decodeDeliveryLogs(ptr.To(`[{"level":"info","msg":"finish"}]`))
		require.Len(t, got, 1)
		require.Nil(t, got[0].Data)
		require.Nil(t, got[0].Ts, "an agent that reports no time must not get a fake one")
	})
}
