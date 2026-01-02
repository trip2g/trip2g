package similarnotes_test

import (
	"context"
	"testing"

	"trip2g/internal/case/similarnotes"
	"trip2g/internal/features"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty when vector search disabled", func(t *testing.T) {
		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: false},
				}
			},
		}

		result, err := similarnotes.Resolve(ctx, env, model.SimilarNotesInput{Path: "/test"})
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("returns empty when note not found", func(t *testing.T) {
		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: true},
				}
			},
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					Map:  map[string]*appmodel.NoteView{},
					List: []*appmodel.NoteView{},
				}
			},
		}

		result, err := similarnotes.Resolve(ctx, env, model.SimilarNotesInput{Path: "/nonexistent"})
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("returns empty when source note has no embedding", func(t *testing.T) {
		noteView := &appmodel.NoteView{
			VersionID: 1,
			Permalink: "/test",
			Embedding: nil, // no embedding
		}

		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: true},
				}
			},
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					Map:  map[string]*appmodel.NoteView{"/test": noteView},
					List: []*appmodel.NoteView{noteView},
				}
			},
		}

		result, err := similarnotes.Resolve(ctx, env, model.SimilarNotesInput{Path: "/test"})
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("returns similar notes sorted by similarity", func(t *testing.T) {
		sourceNote := &appmodel.NoteView{
			VersionID: 1,
			Permalink: "/source",
			Embedding: []float32{1.0, 0.0, 0.0},
		}
		similarNote := &appmodel.NoteView{
			VersionID: 2,
			Permalink: "/similar",
			Embedding: []float32{0.9, 0.1, 0.0}, // High similarity
		}
		lessSimilarNote := &appmodel.NoteView{
			VersionID: 3,
			Permalink: "/less-similar",
			Embedding: []float32{0.5, 0.5, 0.5}, // Lower similarity
		}

		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: true},
				}
			},
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					Map: map[string]*appmodel.NoteView{
						"/source":       sourceNote,
						"/similar":      similarNote,
						"/less-similar": lessSimilarNote,
					},
					List: []*appmodel.NoteView{sourceNote, similarNote, lessSimilarNote},
				}
			},
			CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
				return true, nil
			},
		}

		result, err := similarnotes.Resolve(ctx, env, model.SimilarNotesInput{Path: "/source"})
		require.NoError(t, err)
		require.Len(t, result, 2)

		// First result should be the more similar one
		require.Equal(t, "/similar", result[0].Note.NoteView.Permalink)
		require.Equal(t, "/less-similar", result[1].Note.NoteView.Permalink)

		// Scores should be in descending order
		require.Greater(t, result[0].Score, result[1].Score)
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		notes := make([]*appmodel.NoteView, 10)
		noteMap := make(map[string]*appmodel.NoteView)

		for i := range 10 {
			notes[i] = &appmodel.NoteView{
				VersionID: int64(i + 1),
				Permalink: "/note" + string(rune('0'+i)),
				Embedding: []float32{0.9, 0.1, 0.0},
			}
			if i == 0 {
				notes[i].Embedding = []float32{1.0, 0.0, 0.0} // source
			}
			noteMap[notes[i].Permalink] = notes[i]
		}

		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: true},
				}
			},
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					Map:  noteMap,
					List: notes,
				}
			},
			CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
				return true, nil
			},
		}

		limit := int32(3)
		result, err := similarnotes.Resolve(ctx, env, model.SimilarNotesInput{
			Path: "/note0",
			Limit:  &limit,
		})
		require.NoError(t, err)
		require.Len(t, result, 3)
	})

	t.Run("filters notes user cannot read", func(t *testing.T) {
		sourceNote := &appmodel.NoteView{
			VersionID: 1,
			Permalink: "/source",
			Embedding: []float32{1.0, 0.0, 0.0},
		}
		readableNote := &appmodel.NoteView{
			VersionID: 2,
			Permalink: "/readable",
			Embedding: []float32{0.9, 0.1, 0.0},
		}
		restrictedNote := &appmodel.NoteView{
			VersionID: 3,
			Permalink: "/restricted",
			Embedding: []float32{0.9, 0.1, 0.0},
		}

		env := &EnvMock{
			FeaturesFunc: func() features.Features {
				return features.Features{
					VectorSearch: features.VectorSearchConfig{Enabled: true},
				}
			},
			LatestNoteViewsFunc: func() *appmodel.NoteViews {
				return &appmodel.NoteViews{
					Map: map[string]*appmodel.NoteView{
						"/source":     sourceNote,
						"/readable":   readableNote,
						"/restricted": restrictedNote,
					},
					List: []*appmodel.NoteView{sourceNote, readableNote, restrictedNote},
				}
			},
			CanReadNoteFunc: func(ctx context.Context, note *appmodel.NoteView) (bool, error) {
				// Only readable note is allowed
				return note.Permalink == "/readable", nil
			},
		}

		result, err := similarnotes.Resolve(ctx, env, model.SimilarNotesInput{Path: "/source"})
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Equal(t, "/readable", result[0].Note.NoteView.Permalink)
	})
}
