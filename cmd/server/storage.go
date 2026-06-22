package main

import (
	"context"
	"fmt"
	"io"

	"trip2g/internal/appconfig"
	"trip2g/internal/db"
	"trip2g/internal/localstorage"
	"trip2g/internal/miniostorage"
	"trip2g/internal/model"
)

// Storage is the file-storage surface used by the app. Both the MinIO/S3 backend
// (miniostorage.FileStorage) and the local-filesystem backend
// (localstorage.LocalStorage) implement it, so the app can run with or without
// MinIO. The app embeds this interface; its methods promote onto *app.
type Storage interface {
	NoteAssetPath(asset db.NoteAsset) string
	OnURLExpiring(callback func())
	NoteAssetExists(ctx context.Context, asset db.NoteAsset) (bool, error)
	NoteAssetURL(ctx context.Context, asset db.NoteAsset) (model.PresignedURL, error)
	DeleteAssetObject(ctx context.Context, asset db.NoteAsset) error
	GetAssetObject(ctx context.Context, asset db.NoteAsset) (io.ReadCloser, error)
	PutAssetObject(ctx context.Context, reader io.Reader, asset db.NoteAsset) error
	PutPrivateObject(ctx context.Context, reader io.Reader, objectID string) error
	PrivateObjectExists(ctx context.Context, objectID string) (bool, error)
	GetPrivateObject(ctx context.Context, objectID string) (io.ReadCloser, error)
	ListPrivateObjects(ctx context.Context, opts miniostorage.ListOptions) ([]model.PrivateObject, error)
	DeletePrivateObject(ctx context.Context, objectID string) error
}

// Compile-time checks that both backends satisfy the Storage interface.
var (
	_ Storage = (*miniostorage.FileStorage)(nil)
	_ Storage = (*localstorage.LocalStorage)(nil)
)

// newStorage constructs the configured storage backend. With backend "local"
// it builds a filesystem-backed store (no MinIO required); otherwise it builds
// the MinIO/S3 backend.
func newStorage(ctx context.Context, config *appconfig.Config) (Storage, error) {
	if config.StorageBackend == appconfig.StorageBackendLocal {
		return localstorage.New(localstorage.Config{
			Dir:       config.StorageLocalDir,
			PublicURL: config.PublicURL,
			Prefix:    config.Storage.Prefix,
		})
	}

	store, err := miniostorage.New(ctx, config.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to init minio storage: %w", err)
	}
	return store, nil
}
