package shortapitoken_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/shortapitoken"
)

func TestSignParse_RoundTripsDeliveryIdentity(t *testing.T) {
	in := shortapitoken.Data{
		Depth:         1,
		ReadPatterns:  []string{"boards/**"},
		WritePatterns: []string{"boards/**"},
		DeliveryKind:  "change",
		DeliveryID:    4242,
	}
	tok, err := shortapitoken.Sign(in, "secret", time.Minute)
	require.NoError(t, err)

	out, err := shortapitoken.Parse(tok, "secret")
	require.NoError(t, err)
	require.Equal(t, "change", out.DeliveryKind)
	require.EqualValues(t, 4242, out.DeliveryID)
}
