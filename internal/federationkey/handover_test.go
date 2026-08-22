package federationkey_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"trip2g/internal/federationkey"

	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	handover := federationkey.Handover{
		KID:       "alice-2026",
		KBURL:     "https://bob.team.io/_system/mcp",
		SecretHex: "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
		KBID:      "bob.team.io",
	}

	key, err := federationkey.Encode(handover)
	require.NoError(t, err)

	decoded, err := federationkey.Decode(key)
	require.NoError(t, err)
	require.Equal(t, federationkey.Version, decoded.Version)
	require.Equal(t, handover.KID, decoded.KID)
	require.Equal(t, handover.KBURL, decoded.KBURL)
	require.Equal(t, handover.SecretHex, decoded.SecretHex)
	require.Equal(t, handover.KBID, decoded.KBID)
}

// A key travels through a chat window and a clipboard, both of which introduce
// whitespace and line breaks. Refusing over that would send an operator hunting
// for a corruption that is not there.
func TestDecodeTolerantOfWhitespace(t *testing.T) {
	t.Parallel()

	key, err := federationkey.Encode(federationkey.Handover{
		KID:       "alice",
		KBURL:     "https://bob.example/_system/mcp",
		SecretHex: "aa",
	})
	require.NoError(t, err)

	mangled := "  " + key[:10] + "\n" + key[10:] + "  \n"

	decoded, err := federationkey.Decode(mangled)

	require.NoError(t, err)
	require.Equal(t, "alice", decoded.KID)
}

// Half a key is worse than none: it would install a pairing neither side
// described and fail later, somewhere else.
func TestDecodeRefusesIncomplete(t *testing.T) {
	t.Parallel()

	for name, handover := range map[string]federationkey.Handover{
		"no kid":    {Version: federationkey.Version, KBURL: "https://b/_system/mcp", SecretHex: "aa"},
		"no url":    {Version: federationkey.Version, KID: "alice", SecretHex: "aa"},
		"no secret": {Version: federationkey.Version, KID: "alice", KBURL: "https://b/_system/mcp"},
	} {
		raw, err := json.Marshal(handover)
		require.NoError(t, err, name)

		_, err = federationkey.Decode(base64.RawURLEncoding.EncodeToString(raw))

		require.Error(t, err, name)
	}
}

// A key from a newer trip2g says so rather than decoding into fields this
// version has never seen.
func TestDecodeRefusesAnUnknownVersion(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(federationkey.Handover{
		Version:   federationkey.Version + 1,
		KID:       "alice",
		KBURL:     "https://b/_system/mcp",
		SecretHex: "aa",
	})
	require.NoError(t, err)

	_, err = federationkey.Decode(base64.RawURLEncoding.EncodeToString(raw))

	require.ErrorIs(t, err, federationkey.ErrUnsupportedVersion)
}

func TestDecodeRefusesRubbish(t *testing.T) {
	t.Parallel()

	for _, given := range []string{"", "   ", "not-base64!!", base64.RawURLEncoding.EncodeToString([]byte("{"))} {
		_, err := federationkey.Decode(given)
		require.Error(t, err, given)
	}
}
