package acmecache

import (
	"context"
	"trip2g/internal/db"

	"golang.org/x/crypto/acme/autocert"
)

type Env interface {
	AcmeCertByKey(ctx context.Context, key string) ([]byte, error)
	DeleteAcmeCert(ctx context.Context, key string) error
	InsertAcmeCert(ctx context.Context, arg db.InsertAcmeCertParams) error
}

type Cache struct {
	env Env
}

func New(env Env) *Cache {
	return &Cache{
		env: env,
	}
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := c.env.AcmeCertByKey(ctx, key)
	if db.IsNoFound(err) {
		return nil, autocert.ErrCacheMiss
	}

	return data, err
}

func (s *Cache) Put(ctx context.Context, key string, data []byte) error {
	params := db.InsertAcmeCertParams{
		Key:   key,
		Value: data,
	}

	return s.env.InsertAcmeCert(ctx, params)
}

func (s *Cache) Delete(ctx context.Context, key string) error {
	return s.env.DeleteAcmeCert(ctx, key)
}
