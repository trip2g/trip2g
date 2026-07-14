package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/fleet"
)

// rgRole builds a change-triggerable role (mode change, fires on create+update).
func rgRole(path string, include, exclude, read, write []string) fleet.Role {
	return fleet.Role{
		NotePath:       path,
		Mode:           "change",
		TriggerOn:      []string{"create", "update"},
		TriggerInclude: include,
		TriggerExclude: exclude,
		ReadPatterns:   read,
		WritePatterns:  write,
	}
}

func TestDeriveRoleGraph(t *testing.T) {
	tests := []struct {
		name   string
		roles  []fleet.Role
		edges  []RGEdge
		cycles [][]string
		orphan map[string]bool // role -> expected orphan flag
	}{
		{
			name: "linear pipeline: writer triggers indexer",
			roles: []fleet.Role{
				rgRole("roles/writer.md", []string{"inbox/**"}, nil, nil, []string{"wiki/**"}),
				rgRole("roles/indexer.md", []string{"wiki/**"}, nil, nil, []string{"index/**"}),
			},
			edges: []RGEdge{
				{From: "roles/writer.md", To: "roles/indexer.md", Kind: edgeTrigger, Exact: true},
			},
			cycles: nil,
			// writer: no inbound trigger (nobody writes inbox/**) -> orphan.
			// indexer: no outbound trigger (nobody triggers on index/**) -> orphan.
			orphan: map[string]bool{"roles/writer.md": true, "roles/indexer.md": true},
		},
		{
			name: "two-role trigger cycle",
			roles: []fleet.Role{
				rgRole("roles/a.md", []string{"y/**"}, nil, nil, []string{"x/**"}),
				rgRole("roles/b.md", []string{"x/**"}, nil, nil, []string{"y/**"}),
			},
			edges: []RGEdge{
				{From: "roles/a.md", To: "roles/b.md", Kind: edgeTrigger, Exact: true},
				{From: "roles/b.md", To: "roles/a.md", Kind: edgeTrigger, Exact: true},
			},
			cycles: [][]string{{"roles/a.md", "roles/b.md"}},
			orphan: map[string]bool{"roles/a.md": false, "roles/b.md": false},
		},
		{
			name: "self-trigger loop",
			roles: []fleet.Role{
				rgRole("roles/loop.md", []string{"loop/**"}, nil, nil, []string{"loop/note.md"}),
			},
			edges: []RGEdge{
				{From: "roles/loop.md", To: "roles/loop.md", Kind: edgeTrigger, Exact: true},
			},
			cycles: [][]string{{"roles/loop.md"}},
			orphan: map[string]bool{"roles/loop.md": false},
		},
		{
			name: "dataflow edge (consumer not triggerable on the data)",
			roles: []fleet.Role{
				rgRole("roles/producer.md", []string{"inbox/**"}, nil, nil, []string{"data/**"}),
				// consumer reads data/** but triggers on control/**, so producer's
				// write feeds it without waking it.
				rgRole("roles/consumer.md", []string{"control/**"}, nil, []string{"data/**"}, nil),
			},
			edges: []RGEdge{
				{From: "roles/producer.md", To: "roles/consumer.md", Kind: edgeDataflow, Exact: true},
			},
			cycles: nil,
			orphan: map[string]bool{"roles/producer.md": true, "roles/consumer.md": true},
		},
		{
			name: "trigger_exclude covers the include verbatim -> no trigger edge",
			roles: []fleet.Role{
				rgRole("roles/writer.md", []string{"inbox/**"}, nil, nil, []string{"wiki/**"}),
				rgRole("roles/indexer.md", []string{"wiki/**"}, []string{"wiki/**"}, nil, []string{"index/**"}),
			},
			edges:  nil,
			cycles: nil,
			orphan: map[string]bool{"roles/writer.md": true, "roles/indexer.md": true},
		},
		{
			name: "non-triggerable target (cron) gets no inbound TRIGGER edge",
			roles: []fleet.Role{
				rgRole("roles/writer.md", []string{"inbox/**"}, nil, nil, []string{"wiki/**"}),
				{
					NotePath:       "roles/cron.md",
					Mode:           "cron",
					CronSchedule:   "0 * * * *",
					TriggerInclude: []string{"wiki/**"},
					WritePatterns:  []string{"report/**"},
				},
			},
			edges:  nil,
			cycles: nil,
			orphan: map[string]bool{"roles/writer.md": true, "roles/cron.md": true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := DeriveRoleGraph(tt.roles)

			require.Equal(t, tt.edges, g.Edges, "edges")
			require.Equal(t, tt.cycles, g.Cycles, "cycles")

			require.Len(t, g.Nodes, len(tt.roles))
			for _, n := range g.Nodes {
				want, ok := tt.orphan[n.Role]
				require.True(t, ok, "unexpected node %q", n.Role)
				require.Equal(t, want, n.Orphan, "orphan flag for %q", n.Role)
			}
		})
	}
}

// TestDeriveRoleGraphInboxOutbox pins the node glob projection: inboxGlob is
// trigger_include minus trigger_exclude, outboxGlob is write_patterns.
func TestDeriveRoleGraphInboxOutbox(t *testing.T) {
	roles := []fleet.Role{
		rgRole("roles/x.md",
			[]string{"a/**", "b/**"}, []string{"b/**"}, nil, []string{"out/**"}),
	}
	g := DeriveRoleGraph(roles)
	require.Len(t, g.Nodes, 1)
	require.Equal(t, []string{"a/**"}, g.Nodes[0].InboxGlob)
	require.Equal(t, []string{"out/**"}, g.Nodes[0].OutboxGlob)
}

// TestDeriveRoleGraphDeterministic verifies stable ordering regardless of input
// order (roles are sorted internally).
func TestDeriveRoleGraphDeterministic(t *testing.T) {
	a := rgRole("roles/a.md", []string{"y/**"}, nil, nil, []string{"x/**"})
	b := rgRole("roles/b.md", []string{"x/**"}, nil, nil, []string{"y/**"})
	g1 := DeriveRoleGraph([]fleet.Role{a, b})
	g2 := DeriveRoleGraph([]fleet.Role{b, a})
	require.Equal(t, g1, g2)
}
