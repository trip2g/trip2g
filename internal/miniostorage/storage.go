package miniostorage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
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

type lockInfo struct {
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	TTL       int64     `json:"ttl_seconds"`
}

type FileStorage struct {
	config Config

	minioClient *minio.Client
	instanceID  string
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

	// Generate unique instance ID
	instanceID, err := generateInstanceID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate instance ID: %w", err)
	}

	s := FileStorage{
		config: config,

		minioClient: minioClient,
		instanceID:  instanceID,
	}

	return &s, nil
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
	objectID := a.NoteAssetPath(asset)

	stats, err := a.minioClient.StatObject(
		ctx,
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

func (a *FileStorage) GetAssetObject(ctx context.Context, asset db.NoteAsset) (io.ReadCloser, error) {
	objectID := a.NoteAssetPath(asset)

	object, err := a.minioClient.GetObject(ctx, a.config.Bucket, objectID, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get asset object: %w", err)
	}

	return object, nil
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
	objectID = a.config.Prefix + objectID

	options := minio.PutObjectOptions{
		ContentType: "application/octet-stream",
		PartSize:    16 * 1024 * 1024,
		NumThreads:  1,

		DisableMultipart: false,
	}

	_, err := a.minioClient.PutObject(context.Background(), a.config.Bucket, objectID, reader, -1, options)
	if err != nil {
		return fmt.Errorf("failed to put private object: %w", err)
	}

	return nil
}

func (a *FileStorage) PrivateObjectExists(ctx context.Context, objectID string) (bool, error) {
	objectID = a.config.Prefix + objectID

	stats, err := a.minioClient.StatObject(
		ctx,
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
	objectID = a.config.Prefix + objectID

	object, err := a.minioClient.GetObject(ctx, a.config.Bucket, objectID, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get private object: %w", err)
	}

	return object, nil
}

// PutLock creates a lock file atomically.
// Returns error if lock already exists.
func (a *FileStorage) PutLock(ctx context.Context, objectID string, ttl time.Duration) error {
	objectID = a.config.Prefix + objectID

	// Check if lock exists and get its state
	expired, ownedByUs, etag, err := a.isLockExpired(ctx, objectID, ttl)
	if err != nil {
		// If we can't read the lock, assume it doesn't exist and try to create new one
		return a.createNewLock(ctx, objectID, ttl, "")
	}

	if !expired && !ownedByUs {
		// Lock exists, not expired, and not owned by us - can't acquire
		return errors.New("lock already exists and is owned by another instance")
	}

	if expired && !ownedByUs {
		// Lock exists but expired and owned by someone else - try to replace it
		return a.createNewLock(ctx, objectID, ttl, etag)
	}

	if ownedByUs {
		// We own this lock - renew it using the current ETag
		return a.renewLock(ctx, objectID, ttl, etag)
	}

	// This shouldn't happen, but fallback to creating new lock
	return a.createNewLock(ctx, objectID, ttl, "")
}

// createNewLock creates a new lock, optionally replacing an existing one with the given ETag.
func (a *FileStorage) createNewLock(ctx context.Context, objectID string, ttl time.Duration, expectedETag string) error {
	lock := lockInfo{
		OwnerID:   a.instanceID,
		CreatedAt: time.Now().UTC(),
		TTL:       int64(ttl.Seconds()),
	}

	lockData, err := json.Marshal(lock)
	if err != nil {
		return fmt.Errorf("failed to marshal lock data: %w", err)
	}

	options := minio.PutObjectOptions{
		ContentType: "application/json",
	}

	if expectedETag == "" {
		// Creating new lock - should not exist
		options.SetMatchETagExcept("*")
	} else {
		// Replacing existing lock - must match expected ETag
		options.SetMatchETag(expectedETag)
	}

	content := strings.NewReader(string(lockData))
	_, err = a.minioClient.PutObject(ctx, a.config.Bucket, objectID, content, int64(len(lockData)), options)
	if err != nil {
		return fmt.Errorf("failed to create lock: %w", err)
	}

	return nil
}

// renewLock renews an existing lock that we own.
func (a *FileStorage) renewLock(ctx context.Context, objectID string, ttl time.Duration, expectedETag string) error {
	lock := lockInfo{
		OwnerID:   a.instanceID,
		CreatedAt: time.Now().UTC(),
		TTL:       int64(ttl.Seconds()),
	}

	lockData, err := json.Marshal(lock)
	if err != nil {
		return fmt.Errorf("failed to marshal lock data: %w", err)
	}

	options := minio.PutObjectOptions{
		ContentType: "application/json",
	}

	// Must match the current ETag to ensure we're still the owner
	options.SetMatchETag(expectedETag)

	content := strings.NewReader(string(lockData))
	_, err = a.minioClient.PutObject(ctx, a.config.Bucket, objectID, content, int64(len(lockData)), options)
	if err != nil {
		return fmt.Errorf("failed to renew lock: %w", err)
	}

	return nil
}

// RemoveLock removes the lock file.
func (a *FileStorage) RemoveLock(ctx context.Context, objectID string) error {
	objectID = a.config.Prefix + objectID

	err := a.minioClient.RemoveObject(ctx, a.config.Bucket, objectID, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove lock: %w", err)
	}

	return nil
}

// generateInstanceID creates a unique identifier for this storage instance.
func generateInstanceID() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// getLockInfo reads and parses lock information from the object.
func (a *FileStorage) getLockInfo(ctx context.Context, objectID string) (*lockInfo, string, error) {
	obj, err := a.minioClient.GetObject(ctx, a.config.Bucket, objectID, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	defer obj.Close()

	// Get object info to extract ETag
	objInfo, err := obj.Stat()
	if err != nil {
		return nil, "", err
	}

	// Read object content
	content, err := io.ReadAll(obj)
	if err != nil {
		return nil, "", err
	}

	var lock lockInfo
	err = json.Unmarshal(content, &lock)
	if err != nil {
		return nil, "", err
	}

	return &lock, objInfo.ETag, nil
}

// isLockExpired checks if a lock exists and if it's expired
// Returns: (expired, ownedByUs, etag, error).
func (a *FileStorage) isLockExpired(ctx context.Context, objectID string, _ time.Duration) (bool, bool, string, error) {
	lock, etag, err := a.getLockInfo(ctx, objectID)
	if err != nil {
		// If object doesn't exist, consider it as expired
		return true, false, "", fmt.Errorf("failed to get lock info: %w", err)
	}

	// Check if we own this lock
	ownedByUs := lock.OwnerID == a.instanceID

	// Check if lock is expired
	expiresAt := lock.CreatedAt.Add(time.Duration(lock.TTL) * time.Second)
	expired := time.Now().UTC().After(expiresAt)

	return expired, ownedByUs, etag, nil
}
