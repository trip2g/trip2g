package gitapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"trip2g/internal/graph/model"
)

const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904" // git's well-known empty tree

// diffChangedFiles returns added/modified and deleted paths between two commits.
// If oldRev is empty, everything in newRev counts as added.
func (api *API) diffChangedFiles(oldRev, newRev string) (changed, deleted []string, err error) {
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

// applyDiff applies the changes between oldRev and newRev to the DB:
// changed markdown/html notes are pushed (hash-skipped if already current),
// deletions are hidden. Returns the list of changed note paths.
func (api *API) applyDiff(ctx context.Context, oldRev, newRev string) ([]string, error) {
	changed, deleted, err := api.diffChangedFiles(oldRev, newRev)
	if err != nil {
		return nil, err
	}

	notePaths, err := api.env.AllVisibleNotePaths(ctx)
	if err != nil {
		return nil, fmt.Errorf("apply: note paths: %w", err)
	}
	dbHash := map[string]string{}
	for _, np := range notePaths {
		dbHash[np.Value] = np.LatestContentHash
	}

	input := model.PushNotesInput{}
	applied := []string{}
	for _, f := range changed {
		ext := strings.ToLower(filepath.Ext(f))
		if ext != ".md" && ext != ".html" {
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
		applied = append(applied, f)
	}

	if len(input.Updates) > 0 {
		payload, perr := api.env.PushNotes(ctx, input)
		if perr != nil {
			return nil, fmt.Errorf("apply: push notes: %w", perr)
		}
		if ep, ok := payload.(*model.ErrorPayload); ok {
			return nil, fmt.Errorf("apply: push notes: %s", ep.Message)
		}
	}

	var deletedNotes []string
	for _, f := range deleted {
		ext := strings.ToLower(filepath.Ext(f))
		if ext == ".md" || ext == ".html" {
			deletedNotes = append(deletedNotes, f)
		}
	}
	if len(deletedNotes) > 0 {
		if herr := api.env.HideNotePaths(ctx, deletedNotes); herr != nil {
			return nil, fmt.Errorf("apply: hide notes: %w", herr)
		}
	}

	return applied, nil
}

// showFile reads a file's content at a given commit.
func (api *API) showFile(rev, path string) ([]byte, error) {
	cmd := exec.Command("git", "--git-dir", api.config.RepoPath, "-c", "core.quotePath=false",
		"show", fmt.Sprintf("%s:%s", rev, path))
	cmd.Stderr = nil
	return cmd.Output()
}
