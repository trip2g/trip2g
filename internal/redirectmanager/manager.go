package redirectmanager

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"trip2g/internal/db"
)

type Env interface {
	ListAllRedirects(ctx context.Context) ([]db.Redirect, error)
}

type item struct {
	data   db.Redirect
	regexp *regexp.Regexp
}

type Manager struct {
	sync.RWMutex

	env   Env
	items []item
}

func New(ctx context.Context, env Env) (*Manager, error) {
	manager := Manager{
		env: env,
	}

	err := manager.Refresh(ctx)
	if err != nil {
		return nil, err
	}

	return &manager, nil
}

func (m *Manager) Refresh(ctx context.Context) error {
	redirects, err := m.env.ListAllRedirects(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh redirects: %w", err)
	}

	items := make([]item, 0, len(redirects))

	for _, redirect := range redirects {
		item := item{
			data: redirect,
		}

		// Compile regex if needed
		if redirect.IsRegex {
			pattern := redirect.Pattern
			if redirect.IgnoreCase {
				pattern = "(?i)" + pattern
			}

			compiledRegex, err := regexp.Compile(pattern)
			if err != nil {
				// Skip invalid regex patterns
				continue
			}
			item.regexp = compiledRegex
		}

		items = append(items, item)
	}

	m.Lock()
	m.items = items
	m.Unlock()

	return nil
}

func (m *Manager) Match(path string) *string {
	m.RLock()
	defer m.RUnlock()

	for _, item := range m.items {
		target, match := item.Match(path)
		if match {
			return target
		}
	}

	return nil
}

func (i *item) Match(path string) (*string, bool) {
	if i.data.IsRegex {
		// Use compiled regex
		if i.regexp == nil {
			return nil, false
		}

		if i.regexp.MatchString(path) {
			// Replace $1, $2, etc. with captured groups
			target := i.regexp.ReplaceAllString(path, i.data.Target)
			return &target, true
		}

		return nil, false
	} else {
		// Simple string matching
		pattern := i.data.Pattern
		matchPath := path

		if i.data.IgnoreCase {
			pattern = strings.ToLower(pattern)
			matchPath = strings.ToLower(path)
		}

		if pattern == matchPath {
			return &i.data.Target, true
		}

		return nil, false
	}
}
