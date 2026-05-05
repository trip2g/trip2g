package personaltoken_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"trip2g/internal/personaltoken"
)

func TestGenerate(t *testing.T) {
	tok := personaltoken.Generate()
	require.True(t, personaltoken.IsPersonal(tok), "generated token must start with t2g_ prefix")
	// Prefix (4) + 64 alnum = 68 chars
	require.Len(t, tok, 68)
}

func TestGenerateUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for range 100 {
		tok := personaltoken.Generate()
		_, dup := seen[tok]
		require.False(t, dup, "duplicate token generated")
		seen[tok] = struct{}{}
	}
}

func TestHash(t *testing.T) {
	plaintext := "t2g_sometoken"
	h1 := personaltoken.Hash(plaintext)
	h2 := personaltoken.Hash(plaintext)
	require.Equal(t, h1, h2, "hash must be idempotent")
	require.NotEmpty(t, h1)

	other := personaltoken.Hash("t2g_othertoken")
	require.NotEqual(t, h1, other, "different plaintexts must produce different hashes")
}

func TestDisplayPrefix(t *testing.T) {
	tok := personaltoken.Generate()
	prefix := personaltoken.DisplayPrefix(tok)
	require.Equal(t, tok[:8], prefix)
	require.Len(t, prefix, 8)
}

func TestIsPersonal(t *testing.T) {
	require.True(t, personaltoken.IsPersonal("t2g_abc"))
	require.True(t, personaltoken.IsPersonal("t2g_"+string(make([]byte, 64))))
	require.False(t, personaltoken.IsPersonal(""))
	require.False(t, personaltoken.IsPersonal("Bearer t2g_abc"))
	require.False(t, personaltoken.IsPersonal("eyJhbGciOiJSUzI1NiJ9.eyJ9.sig"))
	require.False(t, personaltoken.IsPersonal("T2g_abc")) // case sensitive
}
