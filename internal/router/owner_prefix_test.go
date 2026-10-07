package router

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoRouteUnderTheOwnerPrefix(t *testing.T) {
	router := New(stubEnv{})

	for _, routes := range []map[string]Endpoint{router.getRoutes, router.postRoutes} {
		require.NotEmpty(t, routes)
		for path := range routes {
			require.False(t, strings.HasPrefix(path, "/_system/extra"), "%s: /_system/extra/ belongs to the site owner", path)
		}
	}
}
