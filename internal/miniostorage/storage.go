package miniostorage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"trip2g/internal/db"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/valyala/fasthttp"

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
	Prefix    string

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
	config Config

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

	config.Prefix = strings.Trim(config.Prefix, "/")
	if len(config.Prefix) > 0 {
		config.Prefix += "/"
	}

	s := FileStorage{
		config: config,

		minioClient: minioClient,
	}

	return &s, nil
}

// ctx creates a safe context for MinIO operations.
//
// WHY THIS IS NEEDED:
// When using fasthttpadaptor to convert fasthttp requests to net/http (for GraphQL),
// the fasthttp.RequestCtx is passed directly as context.Context to the net/http handler.
// MinIO SDK uses net/http client which creates persistent HTTP connections with goroutines.
// These goroutines may outlive the fasthttp request. When fasthttp finishes the request,
// it calls RequestCtx.Reset() which causes a DATA RACE:
//   - fasthttp goroutine: writes to RequestCtx.userData (in Reset())
//   - MinIO HTTP goroutine: reads from RequestCtx.userData (for cancellation)
//
// SOLUTION:
// Detect fasthttp.RequestCtx and replace it with an independent context.Background()
// with timeout, so MinIO HTTP connections don't hold references to the recycled RequestCtx.
func (a *FileStorage) ctx(ctx context.Context) (context.Context, context.CancelFunc) {
	// Check if this is a fasthttp context
	if _, ok := ctx.(*fasthttp.RequestCtx); ok {
		// Create independent context with timeout to avoid data race
		return context.WithTimeout(context.Background(), 60*time.Second)
	}

	// For non-fasthttp contexts, use as-is but add timeout for safety
	return context.WithTimeout(ctx, 60*time.Second)
}

func (a *FileStorage) NoteAssetPath(asset db.NoteAsset) string {
	return fmt.Sprintf("%sna/%d/%s", a.config.Prefix, asset.ID, asset.FileName)
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
	safeCtx, cancel := a.ctx(ctx)
	defer cancel()

	objectID := a.NoteAssetPath(asset)

	stats, err := a.minioClient.StatObject(
		safeCtx,
		a.config.Bucket,
		objectID,
		minio.StatObjectOptions{},
	)

	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}

		return false, fmt.Errorf("failed to check if asset exists: %w %s", err, objectID)
	}

	return stats.Size > 0, nil
}

func (a *FileStorage) NoteAssetURL(ctx context.Context, asset db.NoteAsset) (string, error) {
	safeCtx, cancel := a.ctx(ctx)
	defer cancel()

	presignedURL, err := a.minioClient.PresignedGetObject(
		safeCtx,
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
	safeCtx, cancel := a.ctx(ctx)
	defer cancel()

	path := a.NoteAssetPath(asset)

	err := a.minioClient.RemoveObject(safeCtx, a.config.Bucket, path, minio.RemoveObjectOptions{})
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

func (a *FileStorage) GetAssetObject(ctx context.Context, asset db.NoteAsset) (io.ReadCloser, error) {
	safeCtx, cancel := a.ctx(ctx)
	defer cancel()

	objectID := a.NoteAssetPath(asset)

	object, err := a.minioClient.GetObject(safeCtx, a.config.Bucket, objectID, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get asset object: %w", err)
	}

	return object, nil
}

func (a *FileStorage) PutAssetObject(ctx context.Context, reader io.Reader, asset db.NoteAsset) error {
	safeCtx, cancel := a.ctx(ctx)
	defer cancel()

	options := minio.PutObjectOptions{
		ContentType: detectContentType(asset.FileName),
	}

	path := a.NoteAssetPath(asset)

	_, err := a.minioClient.PutObject(safeCtx, a.config.Bucket, path, reader, asset.Size, options)
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}

	return nil
}

// PutPrivateObject uploads a private object that is not publicly accessible.
func (a *FileStorage) PutPrivateObject(ctx context.Context, reader io.Reader, objectID string) error {
	safeCtx, cancel := a.ctx(ctx)
	defer cancel()

	objectID = a.config.Prefix + objectID

	options := minio.PutObjectOptions{
		ContentType: "application/octet-stream",
		PartSize:    16 * 1024 * 1024,
		NumThreads:  1,

		DisableMultipart: false,
	}

	_, err := a.minioClient.PutObject(safeCtx, a.config.Bucket, objectID, reader, -1, options)
	if err != nil {
		return fmt.Errorf("failed to put private object: %w", err)
	}

	return nil
}

func (a *FileStorage) PrivateObjectExists(ctx context.Context, objectID string) (bool, error) {
	safeCtx, cancel := a.ctx(ctx)
	defer cancel()

	objectID = a.config.Prefix + objectID

	stats, err := a.minioClient.StatObject(
		safeCtx,
		a.config.Bucket,
		objectID,
		minio.StatObjectOptions{},
	)

	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}

		return false, fmt.Errorf("failed to check if private object exists: %w", err)
	}

	return stats.Size > 0, nil
}

func (a *FileStorage) GetPrivateObject(ctx context.Context, objectID string) (io.ReadCloser, error) {
	safeCtx, cancel := a.ctx(ctx)
	defer cancel()

	objectID = a.config.Prefix + objectID

	object, err := a.minioClient.GetObject(safeCtx, a.config.Bucket, objectID, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get private object: %w", err)
	}

	return object, nil
}
