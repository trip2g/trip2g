package rendernotepage_test

import (
	"context"
	"net/http"
	"testing"

	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

// A closed note used to answer 200 OK with the wall page, so a client had to
// parse the HTML to tell "here is the note" from "you cannot read this note".
// The wall pages still render in full — only the status line changes:
//
//	401 — nobody is signed in (sign-in wall, or paywall hit anonymously)
//	403 — signed in, but this viewer has no access to the note
//
// Cache-Control: no-store and noindex, nofollow stay on both walls.
func TestWallPagesAnswerWithAccessStatus(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(note *model.NoteView)
		token      *usertoken.Data
		canRead    bool
		wantStatus int
		wantMarker string
	}{
		{
			name:       "anonymous + non-free note -> 401",
			setup:      func(note *model.NoteView) { note.Free = false },
			wantStatus: http.StatusUnauthorized,
			wantMarker: "paywall",
		},
		{
			name: "anonymous + require_signin subgraph -> 401",
			setup: func(note *model.NoteView) {
				note.Subgraphs = map[string]*model.NoteSubgraph{
					"members": {Name: "members", RequireSignin: true},
				}
			},
			wantStatus: http.StatusUnauthorized,
			wantMarker: "signinwall",
		},
		{
			name:       "signed in without access -> 403",
			token:      &usertoken.Data{ID: 1, Role: "user"},
			canRead:    false,
			wantStatus: http.StatusForbidden,
			wantMarker: "paywall",
		},
		{
			name:       "signed in with access -> 200",
			token:      &usertoken.Data{ID: 1, Role: "user"},
			canRead:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "anonymous + free note -> 200",
			canRead:    true,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, views := cacheTestNote()
			if tt.setup != nil {
				tt.setup(note)
			}
			env, _, _ := cacheTestEnv(views, nil)
			env.CanReadNoteFunc = func(context.Context, *model.NoteView) (bool, error) {
				return tt.canRead, nil
			}

			ctx := newReqCtx(reqOpts{})
			runHandle(t, env, ctx, tt.token)

			require.Equal(t, tt.wantStatus, ctx.Response.StatusCode())

			if tt.wantStatus == http.StatusOK {
				require.NotEqual(t, "no-store", string(ctx.Response.Header.Peek("Cache-Control")))
				return
			}

			require.Equal(t, "no-store", string(ctx.Response.Header.Peek("Cache-Control")),
				"a closed page must never be stored")

			html := string(body(t, ctx))
			require.Contains(t, html, `name="robots" content="noindex, nofollow"`)
			require.Contains(t, html, tt.wantMarker, "the wall page itself is still rendered")
			require.NotContains(t, html, "<h1>Test</h1>", "the note body stays hidden")
		})
	}
}
