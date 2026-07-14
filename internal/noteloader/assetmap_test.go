package noteloader

import (
	"testing"

	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
)

func TestBuildAssetMap_EmitsStableContentAddressedURLs(t *testing.T) {
	hash := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	assets := []RawAsset{
		{
			VersionID:    42,
			Path:         "attachments/pic.png",
			AbsolutePath: "vault/attachments/pic.png",
			NoteAsset:    db.NoteAsset{ID: 7, FileName: "pic.png", Sha256Hash: hash, Size: 10},
		},
	}

	m := buildAssetMap(assets)

	ar := m[42]["attachments/pic.png"]
	require.NotNil(t, ar)
	require.Equal(t, "/_system/assets/"+hash+"/pic.png", ar.URL)
	require.Equal(t, hash, ar.Hash)
	require.Equal(t, int64(7), ar.ID)
	require.Equal(t, "vault/attachments/pic.png", ar.AbsolutePath)
}
