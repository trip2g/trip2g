package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"testing"
	"trip2g/internal/appconfig"
	"trip2g/internal/configregistry"
	"trip2g/internal/db"
	"trip2g/internal/frontmatterpatch"

	"github.com/stretchr/testify/require"
)

// newSingleLoaderTestApp extends newTxTestApp with the pieces the single-note
// loader reads through the app: config, site config and frontmatter patches.
func newSingleLoaderTestApp(t *testing.T) *app {
	t.Helper()

	a := newTxTestApp(t)
	a.config = &appconfig.Config{PublicURL: "https://example.com"}
	a.SiteConfigBuilder = configregistry.NewSiteConfigBuilder(a)
	a.frontmatterPatchLoader = frontmatterpatch.NewLoader(a)
	return a
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// insertLatestNote stores content as the latest version of path and returns
// the version id, the same three writes insertnote performs.
func insertLatestNote(t *testing.T, a *app, path, content string) int64 {
	t.Helper()
	ctx := context.Background()

	notePath, err := a.WriteQueries.InsertNotePath(ctx, db.InsertNotePathParams{
		Value:             path,
		ValueHash:         sha256Hex(path),
		LatestContentHash: sha256Hex(content),
	})
	require.NoError(t, err)

	version, err := a.WriteQueries.IncrementNoteVersionCount(ctx, db.IncrementNoteVersionCountParams{
		LatestContentHash: sha256Hex(content),
		ID:                notePath.ID,
	})
	require.NoError(t, err)

	versionID, err := a.WriteQueries.InsertNoteVersion(ctx, db.InsertNoteVersionParams{
		PathID:  notePath.ID,
		Version: version,
		Content: content,
	})
	require.NoError(t, err)
	return versionID
}

type vaultFixture struct {
	mdID, otherMDID, pageID, blocksID int64
}

// seedVault stores two markdown notes and a two-file layout where the page
// imports a component that references its own asset.
func seedVault(t *testing.T, a *app) vaultFixture {
	t.Helper()
	return vaultFixture{
		mdID:      insertLatestNote(t, a, "notes/a.md", "![[a.png]]\n\n![b](images/b.jpg)\n"),
		otherMDID: insertLatestNote(t, a, "notes/other.md", "![[other.png]]\n"),
		pageID: insertLatestNote(t, a, "_layouts/mesh/index.html",
			`{{ import "_blocks" }}<img src="{{ asset("hero.png") }}">{{ yield shell() content }}x{{ end }}`),
		blocksID: insertLatestNote(t, a, "_layouts/mesh/_blocks.html",
			`{{block shell()}}<img src="{{ asset("logo.svg") }}">{{yield content}}{{end}}`),
	}
}

// TestNoteVersionAssetPaths_SingleVersion pins what uploadNoteAsset validates
// against: the assets of exactly the requested version, for a markdown note and
// for a layout file (whose asset() literals include those of the components it
// imports).
func TestNoteVersionAssetPaths_SingleVersion(t *testing.T) {
	a := newSingleLoaderTestApp(t)
	f := seedVault(t, a)

	tests := []struct {
		name      string
		versionID int64
		want      []string
	}{
		{
			name:      "markdown note lists its own embeds only",
			versionID: f.mdID,
			want:      []string{"a.png", "images/b.jpg"},
		},
		{
			name:      "layout page lists its assets and those of imported components",
			versionID: f.pageID,
			want:      []string{"_layouts/mesh/hero.png", "_layouts/mesh/logo.svg"},
		},
		{
			name:      "layout component lists its own assets",
			versionID: f.blocksID,
			want:      []string{"_layouts/mesh/logo.svg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, err := a.NoteVersionAssetPaths(context.Background(), tt.versionID)
			require.NoError(t, err)

			got := make([]string, 0, len(paths))
			for p := range paths {
				got = append(got, p)
			}
			sort.Strings(got)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestSingleNoteLoaderEnv_RawNotes pins how much of the vault the single-note
// loader pulls: one note for a markdown version, and only the _layouts/ files
// for a layout version (a layout may import other layout files, never notes).
func TestSingleNoteLoaderEnv_RawNotes(t *testing.T) {
	a := newSingleLoaderTestApp(t)
	f := seedVault(t, a)

	tests := []struct {
		name      string
		versionID int64
		wantPaths []string
	}{
		{
			name:      "markdown version loads that note alone",
			versionID: f.mdID,
			wantPaths: []string{"notes/a.md"},
		},
		{
			name:      "layout version loads the layout files alone",
			versionID: f.pageID,
			wantPaths: []string{"_layouts/mesh/_blocks.html", "_layouts/mesh/index.html"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes, err := makeSingleNoteLoaderWrapper(a, tt.versionID).RawNotes(context.Background())
			require.NoError(t, err)

			got := make([]string, 0, len(notes))
			for _, n := range notes {
				got = append(got, n.Path)
			}
			sort.Strings(got)
			require.Equal(t, tt.wantPaths, got)
		})
	}
}
