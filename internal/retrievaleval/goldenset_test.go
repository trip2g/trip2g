package retrievaleval

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadGoldenSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "golden.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
      "queries": [
        {"query":"как ищут экзопланеты","lang":"ru","direction":"ru->ru","expected_urls":["/ru/ekzoplanety"],"verified":true},
        {"query":"unverified one","lang":"en","direction":"en->en","expected_urls":["/x"],"verified":false}
      ]
    }`), 0o644))

	gs, err := LoadGoldenSet(path)
	require.NoError(t, err)
	require.Len(t, gs.Queries, 2)

	require.Len(t, gs.Verified(), 1)
	require.Equal(t, "как ищут экзопланеты", gs.Verified()[0].Query)
}

func TestGoldenSetValidateRejectsEmptyExpected(t *testing.T) {
	gs := GoldenSet{Queries: []GoldenQuery{{Query: "q", Verified: true}}}
	require.Error(t, gs.Validate())
}
