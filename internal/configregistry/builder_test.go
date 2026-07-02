package configregistry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"trip2g/internal/configregistry"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type mockEnv struct {
	strings []db.AllLatestConfigStringsRow
	bools   []db.AllLatestConfigBoolsRow
	ints    []db.AllLatestConfigIntsRow

	stringsErr error
	boolsErr   error
	intsErr    error

	callCount int
}

func (m *mockEnv) AllLatestConfigStrings(ctx context.Context) ([]db.AllLatestConfigStringsRow, error) {
	m.callCount++
	return m.strings, m.stringsErr
}

func (m *mockEnv) AllLatestConfigBools(ctx context.Context) ([]db.AllLatestConfigBoolsRow, error) {
	m.callCount++
	return m.bools, m.boolsErr
}

func (m *mockEnv) AllLatestConfigInts(ctx context.Context) ([]db.AllLatestConfigIntsRow, error) {
	m.callCount++
	return m.ints, m.intsErr
}

func (m *mockEnv) Logger() logger.Logger {
	return &logger.TestLogger{Prefix: "test"}
}

func TestSiteConfigBuilder_DefaultsFromRegistry(t *testing.T) {
	env := &mockEnv{}
	b := configregistry.NewSiteConfigBuilder(env)
	cfg := b.SiteConfig(context.Background())

	require.Equal(t, "%s", cfg.SiteTitleTemplate)
	require.Equal(t, "UTC", cfg.Timezone)
	require.Empty(t, cfg.DefaultLayout)
	require.True(t, cfg.ShowDraftVersions)
	require.True(t, cfg.EnableRSS)
	require.Equal(t, 820, cfg.VectorMinSimilarity)
	require.Equal(t, model.DefaultURLNormalizationMethod, cfg.URLNormalizationMethod)
	require.NotNil(t, cfg.TimeLocation)
	require.Equal(t, time.UTC, cfg.TimeLocation)
}

func TestSiteConfigBuilder_DBValuesOverrideDefaults(t *testing.T) {
	env := &mockEnv{
		strings: []db.AllLatestConfigStringsRow{
			{ValueID: "site_title_template", Value: "%s — My Site"},
			{ValueID: "timezone", Value: "Europe/Moscow"},
		},
		bools: []db.AllLatestConfigBoolsRow{
			{ValueID: "show_draft_versions", Value: false},
			{ValueID: "enable_rss", Value: false},
		},
		ints: []db.AllLatestConfigIntsRow{
			{ValueID: "vector_min_similarity", Value: 500},
		},
	}

	b := configregistry.NewSiteConfigBuilder(env)
	cfg := b.SiteConfig(context.Background())

	require.Equal(t, "%s — My Site", cfg.SiteTitleTemplate)
	require.Equal(t, "Europe/Moscow", cfg.Timezone)
	require.False(t, cfg.ShowDraftVersions)
	require.False(t, cfg.EnableRSS)
	require.Equal(t, 500, cfg.VectorMinSimilarity)

	moscow, _ := time.LoadLocation("Europe/Moscow")
	require.Equal(t, moscow, cfg.TimeLocation)
}

func TestSiteConfigBuilder_SetupFuncConversion(t *testing.T) {
	env := &mockEnv{
		strings: []db.AllLatestConfigStringsRow{
			{ValueID: "url_normalization_method", Value: "cyrillic"},
		},
	}

	b := configregistry.NewSiteConfigBuilder(env)
	cfg := b.SiteConfig(context.Background())

	require.Equal(t, model.URLNormCyrillic, cfg.URLNormalizationMethod)
}

func TestSiteConfigBuilder_ValidateRejectsInvalid(t *testing.T) {
	env := &mockEnv{
		strings: []db.AllLatestConfigStringsRow{
			{ValueID: "url_normalization_method", Value: "bogus_method"},
		},
	}

	b := configregistry.NewSiteConfigBuilder(env)
	cfg := b.SiteConfig(context.Background())

	// Invalid value rejected, default kept.
	require.Equal(t, model.DefaultURLNormalizationMethod, cfg.URLNormalizationMethod)
}

func TestSiteConfigBuilder_InvalidTimezoneKeepsUTC(t *testing.T) {
	env := &mockEnv{
		strings: []db.AllLatestConfigStringsRow{
			{ValueID: "timezone", Value: "Invalid/Zone"},
		},
	}

	b := configregistry.NewSiteConfigBuilder(env)
	cfg := b.SiteConfig(context.Background())

	// Invalid timezone rejected by Validate, default "UTC" kept.
	require.Equal(t, "UTC", cfg.Timezone)
	require.Equal(t, time.UTC, cfg.TimeLocation)
}

func TestSiteConfigBuilder_CacheBehavior(t *testing.T) {
	env := &mockEnv{}
	b := configregistry.NewSiteConfigBuilder(env)

	_ = b.SiteConfig(context.Background())
	_ = b.SiteConfig(context.Background())
	_ = b.SiteConfig(context.Background())

	// DB queried only once (3 queries for first call).
	require.Equal(t, 3, env.callCount)
}

func TestSiteConfigBuilder_InvalidateResetsCache(t *testing.T) {
	env := &mockEnv{}
	b := configregistry.NewSiteConfigBuilder(env)

	_ = b.SiteConfig(context.Background())
	require.Equal(t, 3, env.callCount)

	b.InvalidateSiteConfig()

	_ = b.SiteConfig(context.Background())
	require.Equal(t, 6, env.callCount) // Queried again after invalidation.
}

func TestSiteConfigBuilder_DBErrorKeepsDefaults(t *testing.T) {
	env := &mockEnv{
		stringsErr: errors.New("db error"),
		boolsErr:   errors.New("db error"),
		intsErr:    errors.New("db error"),
	}

	b := configregistry.NewSiteConfigBuilder(env)
	cfg := b.SiteConfig(context.Background())

	// All defaults preserved on DB errors.
	require.Equal(t, "%s", cfg.SiteTitleTemplate)
	require.True(t, cfg.ShowDraftVersions)
	require.Equal(t, 820, cfg.VectorMinSimilarity)
}

func TestTypedRegistryLookup(t *testing.T) {
	sm, ok := configregistry.GetString("timezone")
	require.True(t, ok)
	require.Equal(t, "UTC", sm.Default)

	bm, ok := configregistry.GetBool("enable_rss")
	require.True(t, ok)
	require.True(t, bm.Default)

	im, ok := configregistry.GetInt("vector_min_similarity")
	require.True(t, ok)
	require.Equal(t, 820, im.Default)

	_, ok = configregistry.GetString("nonexistent")
	require.False(t, ok)

	meta, ok := configregistry.Get("timezone")
	require.True(t, ok)
	require.Equal(t, "timezone", meta.ID)
}
