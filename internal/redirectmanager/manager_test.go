package redirectmanager

import (
	"context"
	"regexp"
	"testing"
	"time"
	"trip2g/internal/db"

	"github.com/stretchr/testify/require"
)

//go:generate moq -out mocks_test.go . Env

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*EnvMock)
		expectError bool
	}{
		{
			name: "success",
			setupMock: func(m *EnvMock) {
				m.ListAllRedirectsFunc = func(ctx context.Context) ([]db.Redirect, error) {
					return []db.Redirect{}, nil
				}
			},
			expectError: false,
		},
		{
			name: "env error",
			setupMock: func(m *EnvMock) {
				m.ListAllRedirectsFunc = func(ctx context.Context) ([]db.Redirect, error) {
					return nil, &testError{"database error"}
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.setupMock(env)

			ctx := context.Background()
			manager, err := New(ctx, env)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, manager)
			} else {
				require.NoError(t, err)
				require.NotNil(t, manager)
			}
		})
	}
}

func TestManager_Refresh(t *testing.T) {
	tests := []struct {
		name        string
		redirects   []db.Redirect
		setupMock   func(*EnvMock)
		expectError bool
		checkItems  func(*testing.T, *Manager)
	}{
		{
			name: "simple redirect",
			redirects: []db.Redirect{
				{
					ID:         1,
					CreatedAt:  time.Now(),
					CreatedBy:  1,
					Pattern:    "/old",
					IgnoreCase: false,
					IsRegex:    false,
					Target:     "/new",
				},
			},
			setupMock: func(m *EnvMock) {
				m.ListAllRedirectsFunc = func(ctx context.Context) ([]db.Redirect, error) {
					return []db.Redirect{
						{
							ID:         1,
							CreatedAt:  time.Now(),
							CreatedBy:  1,
							Pattern:    "/old",
							IgnoreCase: false,
							IsRegex:    false,
							Target:     "/new",
						},
					}, nil
				}
			},
			expectError: false,
			checkItems: func(t *testing.T, m *Manager) {
				require.Len(t, m.items, 1)
				require.Equal(t, "/old", m.items[0].data.Pattern)
				require.Nil(t, m.items[0].regexp)
			},
		},
		{
			name: "regex redirect",
			redirects: []db.Redirect{
				{
					ID:         1,
					CreatedAt:  time.Now(),
					CreatedBy:  1,
					Pattern:    "^/old/(.+)$",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/new/$1",
				},
			},
			setupMock: func(m *EnvMock) {
				m.ListAllRedirectsFunc = func(ctx context.Context) ([]db.Redirect, error) {
					return []db.Redirect{
						{
							ID:         1,
							CreatedAt:  time.Now(),
							CreatedBy:  1,
							Pattern:    "^/old/(.+)$",
							IgnoreCase: false,
							IsRegex:    true,
							Target:     "/new/$1",
						},
					}, nil
				}
			},
			expectError: false,
			checkItems: func(t *testing.T, m *Manager) {
				require.Len(t, m.items, 1)
				require.NotNil(t, m.items[0].regexp)
			},
		},
		{
			name: "case insensitive regex",
			redirects: []db.Redirect{
				{
					ID:         1,
					CreatedAt:  time.Now(),
					CreatedBy:  1,
					Pattern:    "^/OLD/(.+)$",
					IgnoreCase: true,
					IsRegex:    true,
					Target:     "/new/$1",
				},
			},
			setupMock: func(m *EnvMock) {
				m.ListAllRedirectsFunc = func(ctx context.Context) ([]db.Redirect, error) {
					return []db.Redirect{
						{
							ID:         1,
							CreatedAt:  time.Now(),
							CreatedBy:  1,
							Pattern:    "^/OLD/(.+)$",
							IgnoreCase: true,
							IsRegex:    true,
							Target:     "/new/$1",
						},
					}, nil
				}
			},
			expectError: false,
			checkItems: func(t *testing.T, m *Manager) {
				require.Len(t, m.items, 1)
				require.NotNil(t, m.items[0].regexp)
			},
		},
		{
			name: "invalid regex skipped",
			redirects: []db.Redirect{
				{
					ID:         1,
					CreatedAt:  time.Now(),
					CreatedBy:  1,
					Pattern:    "[invalid",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/new",
				},
				{
					ID:         2,
					CreatedAt:  time.Now(),
					CreatedBy:  1,
					Pattern:    "/valid",
					IgnoreCase: false,
					IsRegex:    false,
					Target:     "/new",
				},
			},
			setupMock: func(m *EnvMock) {
				m.ListAllRedirectsFunc = func(ctx context.Context) ([]db.Redirect, error) {
					return []db.Redirect{
						{
							ID:         1,
							CreatedAt:  time.Now(),
							CreatedBy:  1,
							Pattern:    "[invalid",
							IgnoreCase: false,
							IsRegex:    true,
							Target:     "/new",
						},
						{
							ID:         2,
							CreatedAt:  time.Now(),
							CreatedBy:  1,
							Pattern:    "/valid",
							IgnoreCase: false,
							IsRegex:    false,
							Target:     "/new",
						},
					}, nil
				}
			},
			expectError: false,
			checkItems: func(t *testing.T, m *Manager) {
				require.Len(t, m.items, 1)
				require.Equal(t, "/valid", m.items[0].data.Pattern)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			tt.setupMock(env)

			manager := &Manager{env: env}
			ctx := context.Background()

			err := manager.Refresh(ctx)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkItems != nil {
					tt.checkItems(t, manager)
				}
			}
		})
	}
}

func TestManager_Match(t *testing.T) {
	tests := []struct {
		name      string
		redirects []db.Redirect
		path      string
		expected  *string
	}{
		{
			name: "exact match",
			redirects: []db.Redirect{
				{
					ID:         1,
					Pattern:    "/old",
					IgnoreCase: false,
					IsRegex:    false,
					Target:     "/new",
				},
			},
			path:     "/old",
			expected: stringPtr("/new"),
		},
		{
			name: "no match",
			redirects: []db.Redirect{
				{
					ID:         1,
					Pattern:    "/old",
					IgnoreCase: false,
					IsRegex:    false,
					Target:     "/new",
				},
			},
			path:     "/other",
			expected: nil,
		},
		{
			name: "case sensitive no match",
			redirects: []db.Redirect{
				{
					ID:         1,
					Pattern:    "/old",
					IgnoreCase: false,
					IsRegex:    false,
					Target:     "/new",
				},
			},
			path:     "/OLD",
			expected: nil,
		},
		{
			name: "case insensitive match",
			redirects: []db.Redirect{
				{
					ID:         1,
					Pattern:    "/old",
					IgnoreCase: true,
					IsRegex:    false,
					Target:     "/new",
				},
			},
			path:     "/OLD",
			expected: stringPtr("/new"),
		},
		{
			name: "regex with capture groups",
			redirects: []db.Redirect{
				{
					ID:         1,
					Pattern:    "^/old/(.+)$",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/new/$1",
				},
			},
			path:     "/old/page",
			expected: stringPtr("/new/page"),
		},
		{
			name: "regex case insensitive",
			redirects: []db.Redirect{
				{
					ID:         1,
					Pattern:    "^/old/(.+)$",
					IgnoreCase: true,
					IsRegex:    true,
					Target:     "/new/$1",
				},
			},
			path:     "/OLD/page",
			expected: stringPtr("/new/page"),
		},
		{
			name: "regex no match",
			redirects: []db.Redirect{
				{
					ID:         1,
					Pattern:    "^/old/(.+)$",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/new/$1",
				},
			},
			path:     "/other/page",
			expected: nil,
		},
		{
			name: "multiple capture groups",
			redirects: []db.Redirect{
				{
					ID:         1,
					Pattern:    "^/old/([^/]+)/([^/]+)$",
					IgnoreCase: false,
					IsRegex:    true,
					Target:     "/new/$2/$1",
				},
			},
			path:     "/old/foo/bar",
			expected: stringPtr("/new/bar/foo"),
		},
		{
			name: "first match wins",
			redirects: []db.Redirect{
				{
					ID:         1,
					Pattern:    "/old",
					IgnoreCase: false,
					IsRegex:    false,
					Target:     "/first",
				},
				{
					ID:         2,
					Pattern:    "/old",
					IgnoreCase: false,
					IsRegex:    false,
					Target:     "/second",
				},
			},
			path:     "/old",
			expected: stringPtr("/first"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				ListAllRedirectsFunc: func(ctx context.Context) ([]db.Redirect, error) {
					return tt.redirects, nil
				},
			}

			ctx := context.Background()
			manager, err := New(ctx, env)
			require.NoError(t, err)

			result := manager.Match(tt.path)

			if tt.expected == nil {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestItem_Match(t *testing.T) {
	tests := []struct {
		name     string
		redirect db.Redirect
		path     string
		expected *string
		match    bool
	}{
		{
			name: "exact string match",
			redirect: db.Redirect{
				Pattern:    "/test",
				IgnoreCase: false,
				IsRegex:    false,
				Target:     "/target",
			},
			path:     "/test",
			expected: stringPtr("/target"),
			match:    true,
		},
		{
			name: "string no match",
			redirect: db.Redirect{
				Pattern:    "/test",
				IgnoreCase: false,
				IsRegex:    false,
				Target:     "/target",
			},
			path:     "/other",
			expected: nil,
			match:    false,
		},
		{
			name: "case sensitive string no match",
			redirect: db.Redirect{
				Pattern:    "/test",
				IgnoreCase: false,
				IsRegex:    false,
				Target:     "/target",
			},
			path:     "/TEST",
			expected: nil,
			match:    false,
		},
		{
			name: "case insensitive string match",
			redirect: db.Redirect{
				Pattern:    "/test",
				IgnoreCase: true,
				IsRegex:    false,
				Target:     "/target",
			},
			path:     "/TEST",
			expected: stringPtr("/target"),
			match:    true,
		},
		{
			name: "regex match with substitution",
			redirect: db.Redirect{
				Pattern:    "^/old/(.+)$",
				IgnoreCase: false,
				IsRegex:    true,
				Target:     "/new/$1",
			},
			path:     "/old/page",
			expected: stringPtr("/new/page"),
			match:    true,
		},
		{
			name: "regex no match",
			redirect: db.Redirect{
				Pattern:    "^/old/(.+)$",
				IgnoreCase: false,
				IsRegex:    true,
				Target:     "/new/$1",
			},
			path:     "/other/page",
			expected: nil,
			match:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := item{
				data: tt.redirect,
			}

			// Compile regex if needed
			if tt.redirect.IsRegex {
				pattern := tt.redirect.Pattern
				if tt.redirect.IgnoreCase {
					pattern = "(?i)" + pattern
				}
				var err error
				item.regexp, err = regexp.Compile(pattern)
				require.NoError(t, err)
			}

			result, match := item.Match(tt.path)

			require.Equal(t, tt.match, match)
			if tt.expected == nil {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, *tt.expected, *result)
			}
		})
	}
}

// Helper functions and types

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

func stringPtr(s string) *string {
	return &s
}
