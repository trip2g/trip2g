package fleet

import (
	"context"
	"encoding/json"
	"fmt"
)

const discoverRolesQuery = `query Discover($like: String!) {
  notePaths(filter: {like: $like}) {
    value
    content
    latestNoteView { meta { key raw } }
  }
}`

// Discovery polls trip2g for role notes under AgentsFolder.
type Discovery struct {
	client       Client
	agentsFolder string
	offeredTools []string
}

// NewDiscovery builds a Discovery over the admin lane.
func NewDiscovery(client Client, agentsFolder string, offeredTools []string) *Discovery {
	return &Discovery{client: client, agentsFolder: agentsFolder, offeredTools: offeredTools}
}

type discoveredNote struct {
	Value          string `json:"value"`
	Content        string `json:"content"`
	LatestNoteView struct {
		Meta []struct {
			Key string `json:"key"`
			Raw string `json:"raw"`
		} `json:"meta"`
	} `json:"latestNoteView"`
}

// DiscoverRoles returns the valid roles plus a slice of per-note validation
// errors (invalid roles are excluded, never registered).
func (d *Discovery) DiscoverRoles(ctx context.Context) ([]Role, []error) {
	raw, err := d.client.GraphQLAdmin(ctx, discoverRolesQuery, map[string]any{"like": likePattern(d.agentsFolder)})
	if err != nil {
		return nil, []error{fmt.Errorf("discover: %w", err)}
	}
	var data struct {
		NotePaths []discoveredNote `json:"notePaths"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, []error{fmt.Errorf("discover decode: %w", err)}
	}

	var roles []Role
	var errs []error
	for _, n := range data.NotePaths {
		meta := map[string]string{}
		for _, m := range n.LatestNoteView.Meta {
			meta[m.Key] = m.Raw
		}
		role, perr := ParseRole(n.Value, n.Content, meta)
		if perr != nil {
			errs = append(errs, fmt.Errorf("parse %s: %w", n.Value, perr))
			continue
		}
		if verr := role.Validate(d.offeredTools); verr != nil {
			errs = append(errs, verr)
			continue
		}
		roles = append(roles, role)
	}
	return roles, errs
}

// likePattern turns "roles/" into the SQL LIKE pattern "roles/%".
func likePattern(folder string) string {
	return folder + "%"
}
