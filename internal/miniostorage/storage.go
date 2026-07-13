package miniostorage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/model"

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

	// PublicURL overrides the scheme and host in presigned URLs.
	// Useful when the SDK connects to an internal endpoint but URLs must use
	// a public-facing address (e.g. to preserve HTTPS signatures through a CDN).
	PublicURL string

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
		Region: config.Region,
	}

	minioClient, err := minio.New(config.Endpoint, &options)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Bucket check/create talks to MinIO over the network. At boot MinIO (or the
	// route to it) can be transiently unreachable — a co-located alloc with extra
	// sidecars boots slower than MinIO's readiness. Retry with backoff instead of
	// panicking on the first failure, so the instance doesn't die (and trigger a
	// deployment auto-revert) before MinIO is ready. Fail loudly only once the
	// retry window is exhausted.
	err = ensureBucketWithRetry(ctx, minioClient, config)
	if err != nil {
		return nil, err
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

// MinIO init retry policy. The total window covers a MinIO/network that is not
// yet ready when a slow-booting alloc starts, without hanging boot forever.
const (
	initRetryMaxElapsed     = 45 * time.Second
	initRetryInitialBackoff = 500 * time.Millisecond
	initRetryMaxBackoff     = 5 * time.Second
)

// errNonTransientStorage wraps a bucket check/create error that a retry cannot
// fix (bad credentials, malformed bucket name, ...), so ensureBucketWithRetry
// can fail fast on it instead of spending the whole retry window on a
// deterministic misconfiguration.
var errNonTransientStorage = errors.New("non-transient minio storage error")

// isNonTransientMinioCode reports whether the MinIO/S3 error code indicates
// misconfiguration (bad credentials, bad bucket name) rather than a
// transiently-unreachable server, so retrying it cannot help.
func isNonTransientMinioCode(code string) bool {
	switch code {
	case "AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch", "InvalidBucketName":
		return true
	default:
		return false
	}
}

// ensureBucketWithRetry checks the bucket exists (creating it if needed),
// retrying transient failures with capped exponential backoff until the retry
// window elapses. Non-transient errors (bad credentials, bad bucket name) fail
// immediately instead of being retried. The final error is returned so callers
// fail loudly rather than silently proceeding with a broken storage backend.
func ensureBucketWithRetry(ctx context.Context, client *minio.Client, config Config) error {
	deadline := time.Now().Add(initRetryMaxElapsed)
	backoff := initRetryInitialBackoff

	var lastErr error
	for {
		lastErr = ensureBucket(ctx, client, config)
		if lastErr == nil {
			return nil
		}

		if errors.Is(lastErr, errNonTransientStorage) {
			return lastErr
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("minio storage not ready after %s: %w", initRetryMaxElapsed, lastErr)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled while waiting for minio storage: %w", ctx.Err())
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > initRetryMaxBackoff {
			backoff = initRetryMaxBackoff
		}
	}
}

// ensureBucket performs a single check/create attempt against MinIO, bounded by
// the configured init timeout.
func ensureBucket(ctx context.Context, client *minio.Client, config Config) error {
	ctx, cancel := context.WithTimeout(ctx, config.InitTimeout)
	defer cancel()

	bucketExists, err := client.BucketExists(ctx, config.Bucket)
	if err != nil {
		if isNonTransientMinioCode(minio.ToErrorResponse(err).Code) {
			return fmt.Errorf("%w: failed to check if bucket exists: %w", errNonTransientStorage, err)
		}
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if bucketExists {
		return nil
	}

	bucketOptions := minio.MakeBucketOptions{
		Region:        config.Region,
		ObjectLocking: true,
	}

	err = client.MakeBucket(ctx, config.Bucket, bucketOptions)
	if err != nil && minio.ToErrorResponse(err).Code != "BucketAlreadyOwnedByYou" {
		if isNonTransientMinioCode(minio.ToErrorResponse(err).Code) {
			return fmt.Errorf("%w: failed to create bucket: %w", errNonTransientStorage, err)
		}
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
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
		return context.WithTimeout(context.Background(), 5*time.Minute)
	}

	// For non-fasthttp contexts, use as-is but add timeout for safety
	return context.WithTimeout(ctx, 5*time.Minute)
}

func (a *FileStorage) NoteAssetPath(asset db.NoteAsset) string {
	return fmt.Sprintf("%sna/%d/%s", a.config.Prefix, asset.ID, asset.FileName)
}

// OnURLExpiring sets up a callback to be called when the URL is about to expire.
// The app must rebuild dependent pages before the URL expires.
func (a *FileStorage) OnURLExpiring(callback func()) {
	interval := a.config.URLExpiresIn - time.Minute*2
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

func (a *FileStorage) NoteAssetURL(ctx context.Context, asset db.NoteAsset) (model.PresignedURL, error) {
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
		return model.PresignedURL{}, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	if a.config.PublicURL != "" {
		public, parseErr := url.Parse(a.config.PublicURL)
		if parseErr == nil {
			presignedURL.Scheme = public.Scheme
			presignedURL.Host = public.Host
		}
	}

	return model.PresignedURL{
		Value:     presignedURL.String(),
		ExpiresAt: time.Now().Add(a.config.URLExpiresIn),
	}, nil
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
	defer object.Close()

	// minio's GetObject returns a lazy reader that fetches on first Read; we must
	// drain it here, while safeCtx is still alive. Returning the lazy reader would
	// let the deferred cancel() fire first, so any later Read fails with
	// "context canceled" (e.g. the git mirror materializer reads after this returns).
	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset object: %w", err)
	}

	return io.NopCloser(bytes.NewReader(data)), nil
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

	objectID = a.config.Prefix + objectID

	object, err := a.minioClient.GetObject(safeCtx, a.config.Bucket, objectID, minio.GetObjectOptions{})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to get private object: %w", err)
	}

	// minio's GetObject is a lazy reader: it streams on Read, so safeCtx must
	// stay alive until the caller finishes reading. These objects (e.g. the git
	// repo tar.gz) can be large, so we stream rather than buffer — tie cancel()
	// to Close() instead of canceling on return (which would break the read with
	// "context canceled", as it did for callers like gitapi.downloadRepo).
	return &cancelOnCloseReadCloser{ReadCloser: object, cancel: cancel}, nil
}

// cancelOnCloseReadCloser wraps a reader so its context is canceled when the
// caller closes it, not when the producing function returns.
type cancelOnCloseReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnCloseReadCloser) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

// ListOptions specifies filtering options for listing private objects.
type ListOptions struct {
	Prefix string // Filter objects by prefix (relative to configured storage prefix)
}

// ListPrivateObjects lists private objects matching the given options.
// Returns objects with keys relative to the configured storage prefix (prefix stripped).
// Results are NOT sorted - caller should sort as needed.
func (a *FileStorage) ListPrivateObjects(ctx context.Context, opts ListOptions) ([]model.PrivateObject, error) {
	safeCtx, cancel := a.ctx(ctx)
	defer cancel()

	fullPrefix := a.config.Prefix + opts.Prefix

	listOpts := minio.ListObjectsOptions{
		Prefix:    fullPrefix,
		Recursive: false,
	}

	var objects []model.PrivateObject

	for object := range a.minioClient.ListObjects(safeCtx, a.config.Bucket, listOpts) {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}

		relativeKey := strings.TrimPrefix(object.Key, a.config.Prefix)

		objects = append(objects, model.PrivateObject{
			Key:          relativeKey,
			Size:         object.Size,
			LastModified: object.LastModified,
			ETag:         object.ETag,
		})
	}

	return objects, nil
}

// DeletePrivateObject deletes a private object by its relative key.
func (a *FileStorage) DeletePrivateObject(ctx context.Context, objectID string) error {
	safeCtx, cancel := a.ctx(ctx)
	defer cancel()

	objectID = a.config.Prefix + objectID

	err := a.minioClient.RemoveObject(safeCtx, a.config.Bucket, objectID, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove private object: %w", err)
	}

	return nil
}
