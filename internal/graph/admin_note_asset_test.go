package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/assetindex"
	"trip2g/internal/db"
	appmodel "trip2g/internal/model"
)

type assetOwnershipEnv struct {
	Env
	owners map[assetindex.Key]assetindex.Ownership
}

func (e *assetOwnershipEnv) AssetOwnership(sha256Hash, fileName string) (assetindex.Ownership, bool) {
	ownership, ok := e.owners[assetindex.Key{Hash: sha256Hash, FileName: fileName}]
	return ownership, ok
}

func TestAdminNoteAssetNotes(t *testing.T) {
	first := &appmodel.NoteView{Path: "blog/one.md"}
	second := &appmodel.NoteView{Path: "uploads/_files.md"}

	env := &assetOwnershipEnv{
		owners: map[assetindex.Key]assetindex.Ownership{
			{Hash: "abc", FileName: "photo.png"}: {Notes: []*appmodel.NoteView{first, second}},
		},
	}
	r := &adminNoteAssetResolver{&Resolver{DefaultEnv: env}}
	ctx := inLinksCtx(env)

	tests := []struct {
		name  string
		asset db.NoteAsset
		want  []string
	}{
		{
			name:  "every note that references the file",
			asset: db.NoteAsset{Sha256Hash: "abc", FileName: "photo.png"},
			want:  []string{"blog/one.md", "uploads/_files.md"},
		},
		{
			name:  "same bytes under another name are another file",
			asset: db.NoteAsset{Sha256Hash: "abc", FileName: "other.png"},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Notes(ctx, &tt.asset)
			require.NoError(t, err)

			paths := make([]string, 0, len(got))
			for _, note := range got {
				paths = append(paths, note.Path)
			}

			require.Equal(t, tt.want, paths)
		})
	}
}

func TestAdminNoteAssetContentType(t *testing.T) {
	r := &adminNoteAssetResolver{&Resolver{}}

	got, err := r.ContentType(context.Background(), &db.NoteAsset{FileName: "report.pdf"})
	require.NoError(t, err)
	require.Equal(t, "application/pdf", got)
}
