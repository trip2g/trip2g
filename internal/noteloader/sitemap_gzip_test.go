package noteloader

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type publicURLEnv struct {
	Env
}

func (publicURLEnv) PublicURL() string {
	return "https://example.com"
}

func TestGenerateSitemapStoresGzipCopy(t *testing.T) {
	l := &Loader{
		env: publicURLEnv{},
		log: &logger.TestLogger{},
	}
	nvs := model.NewNoteViews()

	err := l.generateSitemap(nvs)
	require.NoError(t, err)
	require.NotEmpty(t, nvs.Sitemap)

	zr, err := gzip.NewReader(bytes.NewReader(nvs.SitemapGzip))
	require.NoError(t, err)
	unzipped, err := io.ReadAll(zr)
	require.NoError(t, err)
	require.Equal(t, nvs.Sitemap, unzipped)
}
