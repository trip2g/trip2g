package gitapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"trip2g/internal/graph/model"

	"github.com/99designs/gqlgen/graphql"
)

const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904" // git's well-known empty tree

// diffChangedFiles returns added/modified and deleted paths between two commits.
// If oldRev is empty, everything in newRev counts as added.
func (api *API) diffChangedFiles(oldRev, newRev string) ([]string, []string, error) {
	base := oldRev
	if base == "" {
		base = emptyTree
	}
	out, err := api.gitCmd(os.Environ(), nil, "diff", "--name-status", "--no-renames", "-z", base, newRev)
	if err != nil {
		return nil, nil, fmt.Errorf("diff: %w", err)
	}
	// -z output: STATUS\0PATH\0STATUS\0PATH\0...
	parts := strings.Split(out, "\x00")
	var changed, deleted []string
	for i := 0; i+1 < len(parts); i += 2 {
		status, path := parts[i], parts[i+1]
		if path == "" {
			continue
		}
		switch {
		case strings.HasPrefix(status, "D"):
			deleted = append(deleted, path)
		default: // A, M, etc.
			changed = append(changed, path)
		}
	}
	return changed, deleted, nil
}

// applyDiff applies the changes between oldRev and newRev to the DB the way
// the sync plugin does: changed markdown/html notes are pushed (hash-skipped if
// already current) without committing, the assets those notes reference are
// uploaded and linked, then the batch is committed. Deletions are hidden.
func (api *API) applyDiff(ctx context.Context, oldRev, newRev string) error {
	changed, deleted, err := api.diffChangedFiles(oldRev, newRev)
	if err != nil {
		return err
	}

	notePaths, err := api.env.AllVisibleNotePaths(ctx)
	if err != nil {
		return fmt.Errorf("apply: note paths: %w", err)
	}
	dbHash := map[string]string{}
	for _, np := range notePaths {
		dbHash[np.Value] = np.LatestContentHash
	}

	skipCommit := true
	input := model.PushNotesInput{SkipCommit: &skipCommit}
	changedAssets := 0
	for _, f := range changed {
		if !isNoteFile(f) {
			changedAssets++
			continue
		}
		content, rerr := api.showFile(newRev, f)
		if rerr != nil {
			continue
		}
		sum := sha256.Sum256(content)
		hash := base64.URLEncoding.EncodeToString(sum[:])
		if dbHash[f] == hash {
			continue // already current in DB — breaks the materialize<->apply loop
		}
		input.Updates = append(input.Updates, model.PushNoteInput{Path: f, Content: string(content)})
	}

	if len(input.Updates) > 0 || changedAssets > 0 {
		err = api.pushAndCommit(ctx, input, newRev, changed)
		if err != nil {
			return err
		}
	}

	var deletedNotes []string
	for _, f := range deleted {
		if isNoteFile(f) {
			deletedNotes = append(deletedNotes, f)
		}
	}
	if len(deletedNotes) > 0 {
		if herr := api.env.HideNotePaths(ctx, deletedNotes); herr != nil {
			return fmt.Errorf("apply: hide notes: %w", herr)
		}
	}

	return nil
}

func isNoteFile(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	return ext == ".md" || ext == ".html"
}

func (api *API) pushAndCommit(ctx context.Context, input model.PushNotesInput, rev string, changed []string) error {
	payload, err := api.env.PushNotes(ctx, input)
	if err != nil {
		return fmt.Errorf("apply: push notes: %w", err)
	}
	if ep, ok := payload.(*model.ErrorPayload); ok {
		return fmt.Errorf("apply: push notes: %s", ep.Message)
	}
	pushed, ok := payload.(*model.PushNotesPayload)
	if !ok {
		return fmt.Errorf("apply: push notes: unexpected payload %T", payload)
	}

	err = api.uploadAssets(ctx, rev, changed, pushed.Notes)
	if err != nil {
		return err
	}

	commit, err := api.env.CommitNotes(ctx)
	if err != nil {
		return fmt.Errorf("apply: commit notes: %w", err)
	}
	if ep, refused := commit.(*model.ErrorPayload); refused {
		return fmt.Errorf("apply: commit notes: %s", ep.Message)
	}

	return nil
}

// uploadAssets stores the files of rev that notes reference, for every note
// changed in this push and every referenced file changed in it. An asset the
// server already holds at the same path and hash is skipped, as the plugin
// skips it.
func (api *API) uploadAssets(ctx context.Context, rev string, changed []string, notes []model.PushedNote) error {
	changedSet := make(map[string]struct{}, len(changed))
	for _, f := range changed {
		changedSet[f] = struct{}{}
	}

	files, err := api.treeFiles(rev)
	if err != nil {
		return err
	}

	for _, note := range notes {
		_, noteChanged := changedSet[note.Path]
		for _, asset := range note.Assets {
			resolved := resolveAssetPath(files, asset.Path, note.Path)
			if resolved == "" {
				continue
			}
			if _, assetChanged := changedSet[resolved]; !noteChanged && !assetChanged {
				continue
			}

			err = api.uploadAsset(ctx, rev, note, asset, resolved)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (api *API) uploadAsset(ctx context.Context, rev string, note model.PushedNote, asset model.PushedNoteAsset, resolved string) error {
	content, err := api.showFile(rev, resolved)
	if err != nil {
		return fmt.Errorf("apply: read asset %s: %w", resolved, err)
	}

	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])
	if asset.Sha256Hash != nil && *asset.Sha256Hash == hash && strings.TrimPrefix(asset.AbsolutePath, "/") == resolved {
		return nil
	}

	skipCommit := true
	input := model.UploadNoteAssetInput{
		SkipCommit: &skipCommit,
		File: graphql.Upload{
			File:     bytes.NewReader(content),
			Filename: path.Base(resolved),
			Size:     int64(len(content)),
		},
		NoteID:       note.ID,
		Sha256Hash:   hash,
		Path:         asset.Path,
		AbsolutePath: resolved,
	}

	payload, err := api.env.UploadNoteAsset(ctx, input)
	if err != nil {
		return fmt.Errorf("apply: upload asset %s: %w", resolved, err)
	}
	if ep, ok := payload.(*model.ErrorPayload); ok {
		return fmt.Errorf("apply: upload asset %s: %s", resolved, ep.Message)
	}

	return nil
}

// treeFiles lists every file path in rev.
func (api *API) treeFiles(rev string) (map[string]struct{}, error) {
	out, err := api.gitCmd(os.Environ(), nil, "ls-tree", "-r", "--name-only", "-z", rev)
	if err != nil {
		return nil, fmt.Errorf("apply: list tree: %w", err)
	}

	files := map[string]struct{}{}
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			files[p] = struct{}{}
		}
	}

	return files, nil
}

// showFile reads a file's content at a given commit.
func (api *API) showFile(rev, path string) ([]byte, error) {
	cmd := exec.Command("git", "--git-dir", api.config.RepoPath, "-c", "core.quotePath=false",
		"show", fmt.Sprintf("%s:%s", rev, path))
	cmd.Stderr = nil
	return cmd.Output()
}
