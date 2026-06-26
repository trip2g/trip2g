package defaulttemplate

import (
	"encoding/json"
	"testing"

	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

func TestCtx_KanbanDataJSON(t *testing.T) {
	t.Run("nil note returns empty string", func(t *testing.T) {
		ctx := &Ctx{}
		require.Equal(t, "", ctx.KanbanDataJSON())
	})

	t.Run("non-kanban note returns empty string", func(t *testing.T) {
		nvs := makeNVS([]*model.NoteView{makeNote("tasks.md", map[string]interface{}{"title": "Tasks"})})
		ctx := &Ctx{Note: nvs.ByPath("tasks.md")}
		require.Equal(t, "", ctx.KanbanDataJSON())
	})

	t.Run("kanban note with admin token sets editable true", func(t *testing.T) {
		nv := makeNote("board.md", map[string]interface{}{"kanban-plugin": "basic"})
		nv.VersionID = 7
		nv.Content = []byte("## To Do\n- [ ] task")
		nvs := makeNVS([]*model.NoteView{nv})
		ctx := &Ctx{
			Note:      nvs.ByPath("board.md"),
			UserToken: &usertoken.Data{ID: 1, Role: "admin"},
		}
		out := ctx.KanbanDataJSON()
		require.NotEmpty(t, out)

		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(out), &payload))
		require.Equal(t, true, payload["editable"])
		require.Equal(t, float64(7), payload["versionId"])
		require.Equal(t, "board.md", payload["path"])
		require.Equal(t, "## To Do\n- [ ] task", payload["markdown"])
	})

	t.Run("kanban note with non-admin token sets editable false", func(t *testing.T) {
		nv := makeNote("board.md", map[string]interface{}{"kanban-plugin": "basic"})
		nv.VersionID = 3
		nvs := makeNVS([]*model.NoteView{nv})
		ctx := &Ctx{
			Note:      nvs.ByPath("board.md"),
			UserToken: &usertoken.Data{ID: 2, Role: "user"},
		}
		out := ctx.KanbanDataJSON()
		require.NotEmpty(t, out)

		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(out), &payload))
		require.Equal(t, false, payload["editable"])
	})

	t.Run("kanban note with nil token sets editable false", func(t *testing.T) {
		nv := makeNote("board.md", map[string]interface{}{"kanban-plugin": "basic"})
		nvs := makeNVS([]*model.NoteView{nv})
		ctx := &Ctx{
			Note:      nvs.ByPath("board.md"),
			UserToken: nil,
		}
		out := ctx.KanbanDataJSON()
		require.NotEmpty(t, out)

		var payload map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(out), &payload))
		require.Equal(t, false, payload["editable"])
	})
}
