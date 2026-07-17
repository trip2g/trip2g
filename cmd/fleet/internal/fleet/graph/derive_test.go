package graph

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/cmd/fleet/internal/fleet"
)

// changeRole builds a minimal valid change-mode RoleInput for derive tests.
func changeRole(path, fleetID string, writes, triggers []string) RoleInput {
	return RoleInput{
		Role: fleet.Role{
			NotePath:       path,
			Mode:           "change",
			TriggerOn:      []string{"create", "update"},
			TriggerInclude: triggers,
			WritePatterns:  writes,
		},
		FleetID: fleetID,
	}
}

func edgesOf(g Graph, kind string) []Edge {
	var out []Edge
	for _, e := range g.Edges {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

func findingsOf(g Graph, kind string) []Finding {
	var out []Finding
	for _, f := range g.Findings {
		if f.Kind == kind {
			out = append(out, f)
		}
	}
	return out
}

func TestDeriveTriggerEdge(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"wiki/**"}, []string{"inbox/*"})
	b := changeRole("roles/b.md", "f1", nil, []string{"wiki/topics/*"})
	g := Derive(Input{Roles: []RoleInput{a, b}})

	trig := edgesOf(g, "trigger")
	require.Len(t, trig, 1)
	require.Equal(t, "roles/a.md", trig[0].From)
	require.Equal(t, "roles/b.md", trig[0].To)
	require.Equal(t, "wiki/**", trig[0].Via.Write)
	require.Equal(t, "wiki/topics/*", trig[0].Via.Match)
	require.False(t, trig[0].Fanout)
	require.False(t, trig[0].MaybeExcluded)
}

func TestDeriveNoTriggerEdgeWhenRemoveOnly(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"wiki/**"}, []string{"inbox/*"})
	b := changeRole("roles/b.md", "f1", nil, []string{"wiki/**"})
	b.TriggerOn = []string{"remove"} // v1: agents never delete -> no edge
	g := Derive(Input{Roles: []RoleInput{a, b}})
	require.Empty(t, edgesOf(g, "trigger"))
}

func TestDeriveNoTriggerEdgeForCronOnlyTarget(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"wiki/**"}, []string{"inbox/*"})
	b := RoleInput{
		Role: fleet.Role{
			NotePath: "roles/b.md", Mode: "cron", CronSchedule: "@daily",
			TriggerInclude: []string{"wiki/**"}, // present but unused in cron mode
			ReadPatterns:   []string{"wiki/**"},
		},
		FleetID: "f1",
	}
	g := Derive(Input{Roles: []RoleInput{a, b}})
	require.Empty(t, edgesOf(g, "trigger"))
	// ...but the cron role still consumes state -> feed edge.
	feed := edgesOf(g, "feed")
	require.Len(t, feed, 1)
	require.Equal(t, "roles/a.md", feed[0].From)
	require.Equal(t, "roles/b.md", feed[0].To)
}

func TestDeriveFeedSuppressedByTriggerEdge(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"wiki/**"}, []string{"inbox/*"})
	b := changeRole("roles/b.md", "f1", nil, []string{"wiki/**"})
	b.ReadPatterns = []string{"wiki/**"}
	g := Derive(Input{Roles: []RoleInput{a, b}})
	require.Len(t, edgesOf(g, "trigger"), 1)
	require.Empty(t, edgesOf(g, "feed"))
}

func TestDeriveMaybeExcluded(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"wiki/drafts/**"}, []string{"inbox/*"})
	b := changeRole("roles/b.md", "f1", nil, []string{"wiki/**"})
	b.TriggerExclude = []string{"wiki/drafts/**"}
	g := Derive(Input{Roles: []RoleInput{a, b}})
	trig := edgesOf(g, "trigger")
	require.Len(t, trig, 1)
	require.True(t, trig[0].MaybeExcluded)
}

func TestDeriveFanoutBadge(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"wiki/**"}, []string{"inbox/*"})
	a.ForEach = "changed_files"
	b := changeRole("roles/b.md", "f1", nil, []string{"wiki/**"})
	g := Derive(Input{Roles: []RoleInput{a, b}})
	trig := edgesOf(g, "trigger")
	require.Len(t, trig, 1)
	require.True(t, trig[0].Fanout)
}

func TestDeriveSelfTrigger(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"wiki/**"}, []string{"wiki/**"})
	g := Derive(Input{Roles: []RoleInput{a}})

	trig := edgesOf(g, "trigger")
	require.Len(t, trig, 1)
	require.Equal(t, trig[0].From, trig[0].To)

	st := findingsOf(g, "self-trigger")
	require.Len(t, st, 1)
	require.Equal(t, []string{"roles/a.md"}, st[0].Nodes)
	// max_depth unset -> effective lap budget 1, no fan-out -> WARN.
	require.Equal(t, "warn", st[0].Severity)
}

func TestDeriveSelfTriggerErrorWithDepth(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"wiki/**"}, []string{"wiki/**"})
	a.MaxDepth = 3
	g := Derive(Input{Roles: []RoleInput{a}})
	st := findingsOf(g, "self-trigger")
	require.Len(t, st, 1)
	require.Equal(t, "error", st[0].Severity)
}

func TestDeriveCycle(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"b-in/**"}, []string{"a-in/**"})
	b := changeRole("roles/b.md", "f1", []string{"a-in/**"}, []string{"b-in/**"})
	g := Derive(Input{Roles: []RoleInput{a, b}})

	cyc := findingsOf(g, "cycle")
	require.Len(t, cyc, 1)
	require.Equal(t, []string{"roles/a.md", "roles/b.md"}, cyc[0].Nodes)
	require.Equal(t, "warn", cyc[0].Severity) // lap budget 1, no fan-out
}

func TestDeriveCycleErrorWithFanoutOverlap(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"b-in/**"}, []string{"a-in/**"})
	a.ForEach = "changed_files" // default concurrency = allow_overlap
	b := changeRole("roles/b.md", "f1", []string{"a-in/**"}, []string{"b-in/**"})
	g := Derive(Input{Roles: []RoleInput{a, b}})
	cyc := findingsOf(g, "cycle")
	require.Len(t, cyc, 1)
	require.Equal(t, "error", cyc[0].Severity)
}

func TestDeriveCycleNotFedByFeedEdges(t *testing.T) {
	// a triggers b; b only FEEDS a (b writes what a reads) -> no cycle.
	a := changeRole("roles/a.md", "f1", []string{"b-in/**"}, []string{"a-in/**"})
	a.ReadPatterns = []string{"ctx/**"}
	b := changeRole("roles/b.md", "f1", []string{"ctx/**"}, []string{"b-in/**"})
	g := Derive(Input{Roles: []RoleInput{a, b}})
	require.Empty(t, findingsOf(g, "cycle"))
	require.Len(t, edgesOf(g, "feed"), 1)
}

func TestDeriveOrphanWrite(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"logs/**", "wiki/**"}, []string{"inbox/*"})
	b := changeRole("roles/b.md", "f1", nil, []string{"wiki/**"})
	g := Derive(Input{Roles: []RoleInput{a, b}})

	orph := findingsOf(g, "orphan-write")
	require.Len(t, orph, 1)
	require.Equal(t, "info", orph[0].Severity)
	require.Equal(t, []string{"roles/a.md"}, orph[0].Nodes)
	require.Contains(t, orph[0].Detail, "logs/**")
	require.NotContains(t, orph[0].Detail, "wiki/**")
}

func TestDeriveDanglingTrigger(t *testing.T) {
	a := changeRole("roles/a.md", "f1", nil, []string{"inbox/*"})
	g := Derive(Input{Roles: []RoleInput{a}})
	dang := findingsOf(g, "dangling-trigger")
	require.Len(t, dang, 1)
	require.Equal(t, "info", dang[0].Severity)
	require.Contains(t, dang[0].Detail, "inbox/*")
}

func TestDeriveInvalidRoleKept(t *testing.T) {
	a := changeRole("roles/a.md", "f1", []string{"wiki/**"}, []string{"inbox/*"})
	a.Errors = []string{`tool "x" not offered by this fleet`}
	g := Derive(Input{Roles: []RoleInput{a}})

	require.Len(t, g.Nodes, 1)
	require.False(t, g.Nodes[0].Valid)
	inv := findingsOf(g, "invalid-role")
	require.Len(t, inv, 1)
	require.Equal(t, "error", inv[0].Severity)
}

func TestDeriveLooseErrors(t *testing.T) {
	g := Derive(Input{LooseErrors: []string{"parse roles/broken.md: bad frontmatter"}})
	inv := findingsOf(g, "invalid-role")
	require.Len(t, inv, 1)
	require.Contains(t, inv[0].Detail, "roles/broken.md")
}

func TestDeriveOwnershipConflict(t *testing.T) {
	a := changeRole("roles/a.md", "f1", nil, []string{"inbox/*"})
	g := Derive(Input{
		Roles:       []RoleInput{a},
		Markers:     []Marker{{FleetID: "f1", NotePath: "roles/a.md"}, {FleetID: "f2", NotePath: "roles/a.md"}},
		HasRegistry: true,
	})
	require.Len(t, g.Nodes, 1)
	require.Equal(t, []string{"f1", "f2"}, g.Nodes[0].FleetIDs)
	conf := findingsOf(g, "conflict")
	require.Len(t, conf, 1)
	require.Equal(t, "error", conf[0].Severity)
	// Fleet list is the union of role + marker owners.
	require.Equal(t, []Fleet{{ID: "f1"}, {ID: "f2"}}, g.Fleets)
}

func TestDeriveUnclaimedRole(t *testing.T) {
	a := changeRole("roles/a.md", "", nil, []string{"inbox/*"})
	g := Derive(Input{Roles: []RoleInput{a}})
	uncl := findingsOf(g, "unclaimed-role")
	require.Len(t, uncl, 1)
	require.Equal(t, "warn", uncl[0].Severity)
}

func TestDeriveDrift(t *testing.T) {
	registered := changeRole("roles/a.md", "f1", nil, []string{"inbox/*"})
	unregistered := changeRole("roles/b.md", "f1", nil, []string{"inbox/*"})
	g := Derive(Input{
		Roles: []RoleInput{registered, unregistered},
		Markers: []Marker{
			{FleetID: "f1", NotePath: "roles/a.md"},
			{FleetID: "f1", NotePath: "roles/ghost.md"}, // stale webhook
		},
		HasRegistry: true,
	})

	byPath := map[string]Node{}
	for _, n := range g.Nodes {
		byPath[n.NotePath] = n
	}
	require.Len(t, g.Nodes, 3)
	require.True(t, byPath["roles/a.md"].Registered)
	require.False(t, byPath["roles/b.md"].Registered)
	require.True(t, byPath["roles/ghost.md"].Ghost)
	require.True(t, byPath["roles/ghost.md"].Registered)

	drift := findingsOf(g, "drift")
	require.Len(t, drift, 2)
}

func TestDeriveDriftSkippedWithoutRegistry(t *testing.T) {
	a := changeRole("roles/a.md", "f1", nil, []string{"inbox/*"})
	g := Derive(Input{Roles: []RoleInput{a}})
	require.Empty(t, findingsOf(g, "drift"))
}

func TestDeriveCronRegisteredViaCronMarker(t *testing.T) {
	c := RoleInput{
		Role:    fleet.Role{NotePath: "roles/c.md", Mode: "cron", CronSchedule: "@daily"},
		FleetID: "f1",
	}
	g := Derive(Input{
		Roles:       []RoleInput{c},
		Markers:     []Marker{{FleetID: "f1", NotePath: "roles/c.md", Cron: true}},
		HasRegistry: true,
	})
	require.Len(t, g.Nodes, 1)
	require.True(t, g.Nodes[0].Registered)
	require.Empty(t, findingsOf(g, "drift"))
}

func TestDeriveGeneratedAtAndOrdering(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	b := changeRole("roles/b.md", "f1", nil, []string{"inbox/*"})
	a := changeRole("roles/a.md", "f1", nil, []string{"inbox/*"})
	g := Derive(Input{Roles: []RoleInput{b, a}, GeneratedAt: now})
	require.Equal(t, now, g.GeneratedAt)
	require.Equal(t, "roles/a.md", g.Nodes[0].NotePath)
	require.Equal(t, "roles/b.md", g.Nodes[1].NotePath)
}
