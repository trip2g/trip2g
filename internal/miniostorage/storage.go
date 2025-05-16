package miniostorage

import (
	"context"
	"fmt"
	"io"
	"time"
	"trip2g/internal/model"

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

func (a *FileStorage) PutAssetObject(ctx context.Context, reader io.Reader, info model.FileInfo) error {
	options := minio.PutObjectOptions{
		ContentType: info.ContentType,
	}

	_, err := a.minioClient.PutObject(context.Background(), a.config.Bucket, info.Path, reader, info.Size, options)
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}

	return nil
}
