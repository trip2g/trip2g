package gitapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// materialize rebuilds refs/heads/<master> from the DB (notes + assets) as one
// commit. It is idempotent: if the resulting tree equals the current HEAD tree,
// no commit is made. Callers must hold the note-write lock.
func (api *API) materialize(ctx context.Context) (err error) {
	notes, err := api.env.LatestNoteSources(ctx)
	if err != nil {
		return fmt.Errorf("materialize: note sources: %w", err)
	}
	assets, err := api.env.LatestAssetSources(ctx)
	if err != nil {
		return fmt.Errorf("materialize: asset sources: %w", err)
	}

	indexPath := filepath.Join(api.config.RepoPath, "index.materialize")
	_ = os.Remove(indexPath) // start from an empty index so deletions drop out
	defer os.Remove(indexPath)
	// On failure, remove any loose objects written by hash-object -w that were
	// never reachable (no commit was created). Without cleanup these orphaned
	// blobs accumulate on disk indefinitely — the root cause of the production
	// incident where a single interrupted rebuild stranded ~1.7 G of dangling
	// objects and filled the disk.
	defer func() {
		if err == nil {
			return
		}
		// Remove git temp object files left by an interrupted hash-object run.
		for _, pattern := range []string{
			filepath.Join(api.config.RepoPath, "objects", "*", "tmp_obj_*"),
			filepath.Join(api.config.RepoPath, "objects", "tmp_obj_*"),
		} {
			if matches, gErr := filepath.Glob(pattern); gErr == nil {
				for _, f := range matches {
					_ = os.Remove(f)
				}
			}
		}
		// Drop unreachable loose objects immediately (default grace is 2 weeks).
		if _, pErr := api.gitCmd(os.Environ(), nil, "prune", "--expire=now"); pErr != nil {
			api.logger.Warn("materialize: prune after failure", "error", pErr)
		}
	}()

	gitEnv := append(os.Environ(),
		"GIT_INDEX_FILE="+indexPath,
		"GIT_AUTHOR_NAME=trip2g", "GIT_AUTHOR_EMAIL=server@trip2g",
		"GIT_COMMITTER_NAME=trip2g", "GIT_COMMITTER_EMAIL=server@trip2g",
	)

	for _, n := range notes {
		err = api.addBlob(gitEnv, n.Path, n.Content)
		if err != nil {
			return fmt.Errorf("materialize note %s: %w", n.Path, err)
		}
	}
	for _, a := range assets {
		var rc io.ReadCloser
		rc, err = api.env.ReadAssetObject(ctx, a.Asset)
		if err != nil {
			api.logger.Warn("materialize: skip unreadable asset", "path", a.AbsolutePath, "error", err)
			continue
		}
		content, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			// ReadAssetObject already drained the object under a live context, so
			// this reads from an in-memory buffer and a failure here is a genuine
			// fault, not a missing/transient object (missing objects fail at the
			// open above and are skipped). Surface it rather than silently
			// dropping the asset from the tree.
			return fmt.Errorf("materialize asset %s: %w", a.AbsolutePath, readErr)
		}
		repoPath := strings.TrimPrefix(a.AbsolutePath, "/")
		err = api.addBlob(gitEnv, repoPath, content)
		if err != nil {
			return fmt.Errorf("materialize asset %s: %w", repoPath, err)
		}
	}

	tree, err := api.gitCmd(gitEnv, nil, "write-tree")
	if err != nil {
		return fmt.Errorf("materialize: write-tree: %w", err)
	}
	tree = strings.TrimSpace(tree)

	parentArgs := []string{}
	if api.isRefExists("refs/heads/" + api.config.MasterBranch) {
		headTree, terr := api.gitCmd(gitEnv, nil, "rev-parse", api.config.MasterBranch+"^{tree}")
		if terr == nil && strings.TrimSpace(headTree) == tree {
			return nil // no change
		}
		parentArgs = []string{"-p", api.config.MasterBranch}
	}

	commitArgs := append([]string{"commit-tree", tree, "-m", "server sync"}, parentArgs...)
	commit, err := api.gitCmd(gitEnv, nil, commitArgs...)
	if err != nil {
		return fmt.Errorf("materialize: commit-tree: %w", err)
	}
	commit = strings.TrimSpace(commit)

	_, err = api.gitCmd(gitEnv, nil, "update-ref", "refs/heads/"+api.config.MasterBranch, commit)
	if err != nil {
		return fmt.Errorf("materialize: update-ref: %w", err)
	}
	return nil
}

// addBlob writes content as a blob and stages it at path in the temp index.
func (api *API) addBlob(gitEnv []string, path string, content []byte) error {
	sha, err := api.gitCmd(gitEnv, bytes.NewReader(content), "hash-object", "-w", "--stdin")
	if err != nil {
		return fmt.Errorf("hash-object: %w", err)
	}
	sha = strings.TrimSpace(sha)
	_, err = api.gitCmd(gitEnv, nil, "update-index", "--add", "--cacheinfo", "100644,"+sha+","+path)
	if err != nil {
		return fmt.Errorf("update-index: %w", err)
	}
	return nil
}

// Materialize is the exported entrypoint for scheduled mirror refreshes.
// After a successful rebuild it runs gc to pack loose objects and prune any
// unreachable ones left from rolled-back pushes, keeping the mirror compact
// on disk-constrained servers.
func (api *API) Materialize(ctx context.Context) error {
	api.env.LockNoteWrites()
	defer api.env.UnlockNoteWrites()
	if err := api.initRepo(); err != nil {
		return err
	}
	if err := api.materialize(ctx); err != nil {
		return err
	}
	api.gc()
	return nil
}

// gc packs loose objects and prunes unreachable ones. It is non-fatal: a
// failure (e.g. ENOSPC) is logged as a warning and does not turn a successful
// Materialize into an error.
func (api *API) gc() {
	if _, err := api.gitCmd(os.Environ(), nil, "gc", "--prune=now"); err != nil {
		api.logger.Warn("materialize: git gc failed (non-fatal)", "error", err)
	}
}

// gitCmd runs a git command in the bare repo with the given env and optional stdin.
func (api *API) gitCmd(gitEnv []string, stdin io.Reader, args ...string) (string, error) {
	full := append([]string{"--git-dir", api.config.RepoPath, "-c", "core.quotePath=false"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = gitEnv
	cmd.Stdin = stdin
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %v: %w: %s", args, err, errOut.String())
	}
	return out.String(), nil
}
