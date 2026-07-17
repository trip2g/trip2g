package fleet

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"

	"trip2g/cmd/fleet/internal/fleet/trip2ggql"
)

// Discovery polls trip2g for role notes under AgentsFolder over the admin lane.
type Discovery struct {
	gql          graphql.Client
	fleetID      string
	agentsFolder string
	offeredTools []string
}

// NewDiscovery builds a Discovery over the genqlient admin lane. fleetID is this
// fleet's partition key: DiscoverRoles processes only roles whose fleet_id
// matches it.
func NewDiscovery(gql graphql.Client, fleetID, agentsFolder string, offeredTools []string) *Discovery {
	return &Discovery{gql: gql, fleetID: fleetID, agentsFolder: agentsFolder, offeredTools: offeredTools}
}

// DiscoverRoles returns this fleet's valid roles plus a slice of per-note
// errors. Roles are partitioned by fleet_id: only roles whose fleet_id equals
// this fleet's --fleet-id are processed. A role with an empty fleet_id is
// skipped with a warning (untagged roles are never claimed — belonging to no
// fleet keeps two fleets from both processing it). A role tagged for another
// fleet is skipped silently. Invalid roles that match this fleet are excluded
// and reported. DiscoverParsed stays unpartitioned for cross-fleet introspection.
func (d *Discovery) DiscoverRoles(ctx context.Context) ([]Role, []error) {
	parsed, errs := d.DiscoverParsed(ctx)
	var roles []Role
	for _, role := range parsed {
		if role.FleetID == "" {
			errs = append(errs, fmt.Errorf(
				"role %s: fleet_id is empty; skipped (set fleet_id: %s to assign it to this fleet)",
				role.NotePath, d.fleetID))
			continue
		}
		if role.FleetID != d.fleetID {
			continue // belongs to another fleet
		}
		if verr := role.Validate(d.offeredTools); verr != nil {
			errs = append(errs, verr)
			continue
		}
		roles = append(roles, role)
	}
	return roles, errs
}

// DiscoverParsed fetches and parses every role note under AgentsFolder WITHOUT
// the Validate filter, returning all successfully-parsed roles plus per-note
// parse errors. The --dry-run report uses it so it can show (and flag) the
// resolved config of roles that DiscoverRoles would silently skip.
func (d *Discovery) DiscoverParsed(ctx context.Context) ([]Role, []error) {
	resp, err := trip2ggql.DiscoverRoles(ctx, d.gql, likePattern(d.agentsFolder))
	if err != nil {
		return nil, []error{fmt.Errorf("discover: %w", err)}
	}

	var roles []Role
	var errs []error
	for _, n := range resp.NotePaths {
		meta := map[string]string{}
		for _, m := range n.LatestNoteView.Meta {
			meta[m.Key] = m.Raw
		}
		role, perr := ParseRole(n.Value, n.Content, meta)
		if perr != nil {
			errs = append(errs, fmt.Errorf("parse %s: %w", n.Value, perr))
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
