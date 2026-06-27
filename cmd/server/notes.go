package main

import (
	"context"
	"errors"
	"fmt"
	"trip2g/internal/appreq"
	"trip2g/internal/case/admin/renderpreview"
	"trip2g/internal/case/insertnote"
	"trip2g/internal/case/pushnotes"
	"trip2g/internal/db"
	"trip2g/internal/formspec"
	"trip2g/internal/frontmatterpatch"
	"trip2g/internal/gitapi"
	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/model"
	"trip2g/internal/noteloader"
)

func (a *app) ApplyGitChanges(ctx context.Context) ([]string, error) {
	if a.gitAPI == nil {
		return nil, nil
	}
	if err := a.gitAPI.Materialize(ctx); err != nil {
		return nil, err
	}
	return nil, nil
}

func (a *app) PushNotes(ctx context.Context, input graphmodel.PushNotesInput) (graphmodel.PushNotesOrErrorPayload, error) {
	return pushnotes.Resolve(ctx, a, input)
}

func (a *app) LockNoteWrites() { a.noteWriteMu.Lock() }

func (a *app) UnlockNoteWrites() { a.noteWriteMu.Unlock() }

// LatestNoteSources returns the raw markdown source of every visible note,
// for materializing the git mirror.
func (a *app) LatestNoteSources(ctx context.Context) ([]gitapi.NoteSource, error) {
	nvs := a.LatestNoteViews()
	if nvs == nil {
		return nil, nil
	}
	out := make([]gitapi.NoteSource, 0, len(nvs.List))
	for _, nv := range nvs.List {
		out = append(out, gitapi.NoteSource{Path: nv.Path, Content: nv.Content})
	}
	return out, nil
}

// LatestAssetSources returns every latest note asset for the git mirror.
func (a *app) LatestAssetSources(ctx context.Context) ([]gitapi.AssetSource, error) {
	rows, err := a.Queries.AllLatestNoteAssets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list latest note assets: %w", err)
	}
	out := make([]gitapi.AssetSource, 0, len(rows))
	for _, r := range rows {
		out = append(out, gitapi.AssetSource{
			AbsolutePath: r.NoteAsset.AbsolutePath,
			Asset:        r.NoteAsset,
		})
	}
	return out, nil
}

// HideNotePaths hides the given note paths (used when git deletes files) and
// reloads the in-memory note views so they stop resolving.
func (a *app) HideNotePaths(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	// note_paths.hidden_by is a FK to admins(user_id) and the public visibility
	// filter is `hidden_by IS NULL`. A hide with hidden_by=NULL is therefore a
	// no-op. Git deletions are system-initiated (authenticated by a git token
	// tied to an admin), so attribute the hide to an admin user. We pick the
	// oldest admin (smallest user_id, i.e. the bootstrap owner) so attribution
	// is deterministic regardless of how many admins exist.
	// TODO: thread the authenticating git token's admin_id through the apply
	// path for precise per-actor attribution (see internal/gitapi checkAuth).
	admins, err := a.Queries.ListAllAdmins(ctx)
	if err != nil {
		return fmt.Errorf("failed to list admins for hide: %w", err)
	}
	if len(admins) == 0 {
		return errors.New("cannot hide note paths: no admin user to attribute the hide to")
	}
	// ListAllAdmins orders by user_id DESC, so the last element is the oldest.
	hiddenBy := admins[len(admins)-1].UserID

	for _, p := range paths {
		err = a.WriteQueries.HideNotePath(ctx, db.HideNotePathParams{HiddenBy: &hiddenBy, Value: p})
		if err != nil {
			return fmt.Errorf("failed to hide note path %s: %w", p, err)
		}
	}

	_, err = a.PrepareLatestNotes(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to prepare latest notes after hide: %w", err)
	}
	return nil
}

func (a *app) InsertNote(ctx context.Context, note model.RawNote) (int64, error) {
	return insertnote.Resolve(ctx, a, note)
}

// NoteVersionActor resolves who is pushing the current note version from the
// request context: the acting user id (web/site session, or the API key owner
// for admin-actor calls) and the authenticating API key id (obsidian-sync
// pushes). Either is nil when not applicable, recording an "unknown editor".
func (a *app) NoteVersionActor(ctx context.Context) (createdByUserID *int64, createdByApiKeyID *int64) {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return nil, nil
	}

	if token, tokenErr := req.UserToken(); tokenErr == nil && token != nil && token.ID != 0 {
		uid := int64(token.ID)
		createdByUserID = &uid
	} else if req.AdminActorUserID != 0 {
		uid := int64(req.AdminActorUserID)
		createdByUserID = &uid
	}

	if req.ApiKeyID != nil {
		keyID := *req.ApiKeyID
		createdByApiKeyID = &keyID
	}

	return createdByUserID, createdByApiKeyID
}

func (a *app) InsertUncommittedPath(ctx context.Context, notePathID int64) error {
	return a.WriteQueries.InsertUncommittedPath(ctx, notePathID)
}

func (a *app) ListUncommittedPaths(ctx context.Context) ([]int64, error) {
	return a.Queries.ListUncommittedPaths(ctx)
}

func (a *app) ClearUncommittedPaths(ctx context.Context) error {
	return a.WriteQueries.ClearUncommittedPaths(ctx)
}

// LoadFrontmatterPatches implements noteloader.Env interface.
// It delegates to the frontmatterPatchLoader which handles loading and compilation.
func (a *app) LoadFrontmatterPatches(ctx context.Context) ([]frontmatterpatch.CompiledPatch, error) {
	return a.frontmatterPatchLoader.LoadFrontmatterPatches(ctx)
}

func (a *app) LoadNoteViewByVersionID(ctx context.Context, id int64) (*model.NoteView, error) {
	wrapper := makeSingleNoteLoaderWrapper(a, id)
	loader := noteloader.New("single", wrapper, a.config.MDLoaderConfig)

	err := loader.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true})
	if err != nil {
		return nil, fmt.Errorf("failed to load note version %d: %w", id, err)
	}

	return loader.NoteViews().List[0], nil
}

func (a *app) NoteVersionAssetPaths(ctx context.Context, id int64) (map[string]struct{}, error) {
	wrapper := makeSingleNoteLoaderWrapper(a, id)
	loader := noteloader.New("single", wrapper, a.config.MDLoaderConfig)

	err := loader.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true})
	if err != nil {
		return nil, fmt.Errorf("failed to load note version %d: %w", id, err)
	}

	nvs := loader.NoteViews()
	layouts := loader.Layouts()

	res := map[string]struct{}{}

	if len(layouts.Map) > 0 {
		for _, layout := range layouts.Map {
			// TODO: fix it. the singleNoteLoaderEnv loads all notes for _layouts
			if layout.VersionID != id {
				continue
			}

			for _, asset := range layout.Assets {
				res[asset.Path] = struct{}{}
			}
		}

		return res, nil
	}

	if len(res) == 0 && len(nvs.List) > 0 {
		return nvs.List[0].Assets, nil
	}

	// something strange happened
	return nil, fmt.Errorf("unknown source type #%d not found", id)
}

func (a *app) AllNotePaths(ctx context.Context) ([]db.NotePath, error) {
	return a.queries.AllNotePaths(ctx)
}

func (a *app) NoteURL(note *model.NoteView) string {
	return a.LatestNoteViews().ResolveFullURL(note, a.config.PublicURL)
}

func (a *app) NoteByPath(path string) *model.NoteView {
	return a.latestNoteLoader.NoteByPath(path)
}

func (a *app) noteLoaderForRequest(ctx context.Context) *noteloader.Loader {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return a.liveNoteLoader
	}

	token, err := req.UserToken()
	if err != nil || token == nil {
		return a.liveNoteLoader
	}

	// Check if admin wants to see latest version
	if token.IsAdmin() {
		showLatest := string(req.Req.QueryArgs().Peek("version")) == "latest"
		if showLatest {
			return a.latestNoteLoader
		}
	}

	// Default to live for everyone (including admins without ?version=latest)
	return a.liveNoteLoader
}

func (a *app) NoteByPathForRequest(ctx context.Context, path string) *model.NoteView {
	loader := a.noteLoaderForRequest(ctx)
	return loader.NoteByPath(path)
}

func (a *app) NoteViewsForRequest(ctx context.Context) *model.NoteViews {
	loader := a.noteLoaderForRequest(ctx)
	return loader.NoteViews()
}

func (a *app) LoadLatestLayout(source model.LayoutSourceFile) model.Layout {
	load := a.latestNoteLoader.Layouts().Load
	if load != nil {
		return load(source)
	}
	return model.Layout{}
}

func (a *app) PreviewBuffer() *renderpreview.PreviewBuffer {
	return a.previewBuffer
}

func (a *app) LoadPreviewLayout(main model.LayoutSourceFile, extra map[string]string) (model.Layout, []string) {
	layouts := a.latestNoteLoader.Layouts()
	if layouts.LoadPreview != nil {
		return layouts.LoadPreview(main, extra)
	}
	return model.Layout{}, []string{"preview renderer not initialized"}
}

// GetFormSpec loads form spec for the given note version and form id.
func (a *app) GetFormSpec(ctx context.Context, noteVersionID int64, formID string) (*formspec.FormSpec, error) {
	notes := a.LatestNoteViews()
	note := notes.GetByVersionID(noteVersionID)
	if note == nil {
		return nil, nil
	}
	return resolveFormSpec(note, notes, formID)
}

func resolveFormSpec(note *model.NoteView, notes *model.NoteViews, formID string) (*formspec.FormSpec, error) {
	rawMeta := note.RawMeta
	if kind, value, ok := formspec.ParseFormRef(rawMeta); ok {
		var ref *model.NoteView
		switch kind {
		case "wikilink":
			ref = notes.Map[value]
		case "path":
			ref = notes.GetByPath(value)
		}
		if ref == nil {
			return nil, nil
		}
		rawMeta = ref.RawMeta
	}
	if formsRaw, ok := rawMeta["forms"]; ok {
		formsMap, isMap := formsRaw.(map[string]interface{})
		if !isMap {
			return nil, nil
		}
		formRaw, hasForm := formsMap[formID]
		if !hasForm {
			return nil, nil
		}
		return formspec.ParseFromRawMeta(map[string]interface{}{"form": formRaw})
	}
	return formspec.ParseFromRawMeta(rawMeta)
}
