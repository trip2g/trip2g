// Package localstorage provides a local-filesystem storage backend that
// mirrors the public surface of miniostorage.FileStorage. It lets trip2g run
// without MinIO/S3: assets and private objects (backups, git repo tarballs)
// live on local disk instead of an S3 bucket.
//
// Layout under the configured base dir:
//
//	<dir>/assets/<NoteAssetPath>   — note assets (served via /_system/assets/)
//	<dir>/private/<objectID>       — private objects (backups, git tarballs)
package localstorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/miniostorage"
	"trip2g/internal/model"
)

// Config configures the local-filesystem storage backend.
type Config struct {
	// Dir is the base directory for stored objects. Assets go under
	// <Dir>/assets and private objects under <Dir>/private.
	Dir string

	// Prefix is an optional object key prefix, mirroring miniostorage.Config.Prefix.
	// It keeps keys compatible across backends (assets and private objects share
	// the same prefix as the MinIO backend would use).
	Prefix string
}

const (
	assetsSubdir  = "assets"
	privateSubdir = "private"
)

// LocalStorage is a filesystem-backed implementation of the storage surface
// used by trip2g (the same 12 public methods as miniostorage.FileStorage).
type LocalStorage struct {
	dir    string
	prefix string
}

// New creates a LocalStorage rooted at config.Dir, creating the base
// assets/private directories if they don't exist.
func New(config Config) (*LocalStorage, error) {
	if config.Dir == "" {
		return nil, errors.New("localstorage: base dir is required")
	}

	prefix := strings.Trim(config.Prefix, "/")
	if len(prefix) > 0 {
		prefix += "/"
	}

	s := &LocalStorage{
		dir:    config.Dir,
		prefix: prefix,
	}

	for _, sub := range []string{assetsSubdir, privateSubdir} {
		if err := os.MkdirAll(filepath.Join(config.Dir, sub), 0o750); err != nil {
			return nil, fmt.Errorf("localstorage: failed to create %s dir: %w", sub, err)
		}
	}

	return s, nil
}

// NoteAssetPath returns the storage key for an asset, matching the MinIO
// backend's key scheme ("<prefix>na/<id>/<filename>").
func (s *LocalStorage) NoteAssetPath(asset db.NoteAsset) string {
	return fmt.Sprintf("%sna/%d/%s", s.prefix, asset.ID, asset.FileName)
}

// assetFile returns the absolute on-disk path for an asset key, kept inside the
// assets dir (path traversal in the key is neutralized by filepath.Clean +
// stripping leading separators).
func (s *LocalStorage) assetFile(key string) string {
	return filepath.Join(s.dir, assetsSubdir, safeRel(key))
}

// privateFile returns the absolute on-disk path for a private object key.
func (s *LocalStorage) privateFile(objectID string) string {
	return filepath.Join(s.dir, privateSubdir, safeRel(s.prefix+objectID))
}

// safeRel cleans a storage key into a relative path that cannot escape its
// parent dir via "..". The result is always treated as relative.
func safeRel(key string) string {
	cleaned := filepath.Clean("/" + filepath.FromSlash(key))
	return strings.TrimPrefix(cleaned, string(filepath.Separator))
}

// NoteAssetExists reports whether an asset file is present and non-empty.
func (s *LocalStorage) NoteAssetExists(_ context.Context, asset db.NoteAsset) (bool, error) {
	info, err := os.Stat(s.assetFile(s.NoteAssetPath(asset)))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat asset: %w", err)
	}
	return info.Size() > 0, nil
}

// DeleteAssetObject removes an asset file. Missing files are not an error.
func (s *LocalStorage) DeleteAssetObject(_ context.Context, asset db.NoteAsset) error {
	err := os.Remove(s.assetFile(s.NoteAssetPath(asset)))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove asset: %w", err)
	}
	return nil
}

// GetAssetObject opens an asset for reading.
func (s *LocalStorage) GetAssetObject(_ context.Context, asset db.NoteAsset) (io.ReadCloser, error) {
	f, err := os.Open(s.assetFile(s.NoteAssetPath(asset)))
	if err != nil {
		return nil, fmt.Errorf("failed to open asset: %w", err)
	}
	return f, nil
}

// StreamAssetObject streams length bytes of an asset starting at offset,
// directly from disk.
func (s *LocalStorage) StreamAssetObject(_ context.Context, asset db.NoteAsset, offset, length int64) (io.ReadCloser, error) {
	f, err := os.Open(s.assetFile(s.NoteAssetPath(asset)))
	if err != nil {
		return nil, fmt.Errorf("failed to open asset: %w", err)
	}

	if offset > 0 {
		if _, err = f.Seek(offset, io.SeekStart); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("failed to seek asset: %w", err)
		}
	}

	return &limitedReadCloser{Reader: io.LimitReader(f, length), closer: f}, nil
}

// limitedReadCloser bounds reads to a byte window while closing the
// underlying file when done.
type limitedReadCloser struct {
	io.Reader
	closer io.Closer
}

func (l *limitedReadCloser) Close() error { return l.closer.Close() }

// PutAssetObject writes an asset atomically (temp file + rename).
func (s *LocalStorage) PutAssetObject(_ context.Context, reader io.Reader, asset db.NoteAsset) error {
	return writeFileAtomic(s.assetFile(s.NoteAssetPath(asset)), reader)
}

// PutPrivateObject writes a private object atomically.
func (s *LocalStorage) PutPrivateObject(_ context.Context, reader io.Reader, objectID string) error {
	return writeFileAtomic(s.privateFile(objectID), reader)
}

// PrivateObjectExists reports whether a private object is present and non-empty.
func (s *LocalStorage) PrivateObjectExists(_ context.Context, objectID string) (bool, error) {
	info, err := os.Stat(s.privateFile(objectID))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat private object: %w", err)
	}
	return info.Size() > 0, nil
}

// GetPrivateObject opens a private object for reading. The returned reader can
// be large (e.g. a git repo tarball); it streams directly from disk.
func (s *LocalStorage) GetPrivateObject(_ context.Context, objectID string) (io.ReadCloser, error) {
	f, err := os.Open(s.privateFile(objectID))
	if err != nil {
		return nil, fmt.Errorf("failed to open private object: %w", err)
	}
	return f, nil
}

// ListPrivateObjects walks <dir>/private filtered by opts.Prefix and returns
// metadata with keys relative to the configured storage prefix (prefix
// stripped), mirroring the MinIO backend. Results are not sorted.
func (s *LocalStorage) ListPrivateObjects(_ context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error) {
	root := filepath.Join(s.dir, privateSubdir)
	fullPrefix := s.prefix + opts.Prefix

	var objects []model.PrivateObject

	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		// Storage keys use forward slashes regardless of OS.
		key := filepath.ToSlash(rel)

		if !strings.HasPrefix(key, fullPrefix) {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}

		objects = append(objects, model.PrivateObject{
			Key:          strings.TrimPrefix(key, s.prefix),
			Size:         info.Size(),
			LastModified: info.ModTime(),
		})

		return nil
	})
	if walkErr != nil {
		if os.IsNotExist(walkErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list private objects: %w", walkErr)
	}

	return objects, nil
}

// DeletePrivateObject removes a private object by its relative key. Missing
// objects are not an error.
func (s *LocalStorage) DeletePrivateObject(_ context.Context, objectID string) error {
	err := os.Remove(s.privateFile(objectID))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove private object: %w", err)
	}
	return nil
}

// writeFileAtomic writes reader's contents to path via a temp file in the same
// directory followed by an atomic rename, creating parent dirs as needed.
func writeFileAtomic(path string, reader io.Reader) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Best-effort cleanup if we don't reach the successful rename.
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = io.Copy(tmp, reader); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err = tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	cleanup = false
	return nil
}
