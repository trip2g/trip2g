package miniostorage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path/filepath"
	"time"
	"trip2g/internal/db"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool

	URLExpiresIn time.Duration

	InitTimeout time.Duration
}

func (c *Config) ValidateConfig() error {
	return ozzo.ValidateStruct(c,
		ozzo.Field(&c.Endpoint, ozzo.Required, is.URL),
		ozzo.Field(&c.AccessKey, ozzo.Required),
		ozzo.Field(&c.SecretKey, ozzo.Required),
		ozzo.Field(&c.Bucket, ozzo.Required),
		ozzo.Field(&c.Region, ozzo.Required),
		ozzo.Field(&c.InitTimeout, ozzo.Required),
		ozzo.Field(&c.URLExpiresIn, ozzo.Required),
	)
}

type FileStorage struct {
	config *Config

	minioClient *minio.Client
}

func New(ctx context.Context, config Config) (*FileStorage, error) {
	validateErr := config.ValidateConfig()
	if validateErr != nil {
		return nil, fmt.Errorf("failed to validate config: %w", validateErr)
	}

	options := minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
	}

	minioClient, err := minio.New(config.Endpoint, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, config.InitTimeout)
	defer cancel()

	bucketExists, err := minioClient.BucketExists(ctx, config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !bucketExists {
		bucketOptions := minio.MakeBucketOptions{
			Region:        config.Region,
			ObjectLocking: true,
		}

		err = minioClient.MakeBucket(ctx, config.Bucket, bucketOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	s := FileStorage{
		config: &config,

		minioClient: minioClient,
	}

	return &s, nil
}

func (a *FileStorage) NoteAssetPath(asset db.NoteAsset) string {
	return fmt.Sprintf("na/%d/%s", asset.ID, asset.FileName)
}

// OnURLExpiring sets up a callback to be called when the URL is about to expire.
// The app must rebuild dependent pages before the URL expires.
func (a *FileStorage) OnURLExpiring(callback func()) {
	interval := a.config.URLExpiresIn - time.Minute
	if interval < 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)

	go func() {
		for range ticker.C {
			callback()
		}
	}()
}

func (a *FileStorage) NoteAssetExists(ctx context.Context, asset db.NoteAsset) (bool, error) {
	stats, err := a.minioClient.StatObject(
		ctx,
		a.config.Bucket,
		a.NoteAssetPath(asset),
		minio.StatObjectOptions{},
	)

	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}

		return false, fmt.Errorf("failed to check if asset exists: %w", err)
	}

	return stats.Size > 0, nil
}

func (a *FileStorage) NoteAssetURL(ctx context.Context, asset db.NoteAsset) (string, error) {
	presignedURL, err := a.minioClient.PresignedGetObject(
		ctx,
		a.config.Bucket,
		a.NoteAssetPath(asset),
		a.config.URLExpiresIn,
		url.Values{},
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.String(), nil
}

func (a *FileStorage) DeleteAssetObject(ctx context.Context, asset db.NoteAsset) error {
	path := a.NoteAssetPath(asset)

	err := a.minioClient.RemoveObject(ctx, a.config.Bucket, path, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object: %w", err)
	}

	return nil
}

// detectContentType determines MIME type from file name.
func detectContentType(fileName string) string {
	ext := filepath.Ext(fileName)
	mimeType := mime.TypeByExtension(ext)

	if mimeType != "" {
		return mimeType
	}

	return "application/octet-stream"
}

func (a *FileStorage) PutAssetObject(ctx context.Context, reader io.Reader, asset db.NoteAsset) error {
	options := minio.PutObjectOptions{
		ContentType: detectContentType(asset.FileName),
	}

	path := a.NoteAssetPath(asset)

	_, err := a.minioClient.PutObject(context.Background(), a.config.Bucket, path, reader, asset.Size, options)
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}

	return nil
}

// PutPrivateObject uploads a private object that is not publicly accessible.
func (a *FileStorage) PutPrivateObject(ctx context.Context, reader io.Reader, objectID string) error {
	options := minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}

	_, err := a.minioClient.PutObject(context.Background(), a.config.Bucket, objectID, reader, -1, options)
	if err != nil {
		return fmt.Errorf("failed to put private object: %w", err)
	}

	return nil
}
