package graph

import (
	"slices"
	"strings"

	"trip2g/internal/fleet"
)

// Edge kinds served by the fleet GraphQL roleGraph query.
const (
	edgeTrigger  = "TRIGGER"  // A writes a note that wakes B
	edgeDataflow = "DATAFLOW" // A produces data B reads but isn't woken by
)

// RoleGraph is the agent-interaction graph in the shape the fleet GraphQL
// roleGraph query serves. It is a focused projection distinct from Graph (the
// richer /graph.json debug graph with registry/ownership findings): nodes carry
// inbox/outbox globs and an orphan flag, edges are TRIGGER/DATAFLOW with an
// exact + cutByDepth annotation, and cycles are the strongly-connected
// components over TRIGGER edges (runaway-loop feedback made visible).
//
// It reuses the same exact glob matcher (Overlap) and Tarjan SCC
// (stronglyConnected) as Graph, so the two never diverge on overlap semantics.
type RoleGraph struct {
	Nodes  []RGNode
	Edges  []RGEdge
	Cycles [][]string
}

// RGNode is one role. Role is the note path (the graph identity, matching
// Edge.From/To). InboxGlob is trigger_include minus trigger_exclude; OutboxGlob
// is write_patterns. Orphan flags a role with no inbound TRIGGER edge (woken
// only manually/by cron) or no outbound TRIGGER edge (its writes wake nobody).
type RGNode struct {
	Role       string
	InboxGlob  []string
	OutboxGlob []string
	Orphan     bool
}

// RGEdge is a derived interaction. Exact is always true in v1: Overlap is
// exact for the role-frontmatter glob subset (literals, *, ?, [...], **,
// {a,b}) that role patterns use; the field is reserved for a future
// approximating matcher that would need to report non-exact decisions, not a
// signal this package currently produces. CutByDepth marks an edge the webhook
// max_depth guard would sever; it is always false in v1 (the max_depth wiring
// is future work) and kept as a forward-compatible field.
type RGEdge struct {
	From       string
	To         string
	Kind       string // edgeTrigger | edgeDataflow
	Exact      bool
	CutByDepth bool
}

// DeriveRoleGraph builds the agent-interaction graph from parsed roles. Pure:
// same input, same output (nodes, edges, cycles are ordered deterministically).
//
//   - TRIGGER edge A->B: A.write_patterns overlaps B's effective inbox
//     (trigger_include minus trigger_exclude) and B is change-triggerable.
//   - DATAFLOW edge A->B: A.write_patterns overlaps B.read_patterns (and no
//     TRIGGER edge already covers the pair; self-dataflow is dropped as noise).
//   - Orphan: no inbound TRIGGER edge OR no outbound TRIGGER edge.
//   - Cycles: SCCs (size >= 2) over TRIGGER edges, plus self-trigger loops.
//
// Overlap is conservative toward true, so a spurious edge is preferred over a
// hidden trigger loop (a missing edge would hide a runaway chain).
func DeriveRoleGraph(roles []fleet.Role) RoleGraph {
	sorted := slices.Clone(roles)
	slices.SortFunc(sorted, func(a, b fleet.Role) int { return strings.Compare(a.NotePath, b.NotePath) })

	inbox := make(map[string][]string, len(sorted))
	for _, r := range sorted {
		inbox[r.NotePath] = effectiveInbox(r)
	}

	var edges []RGEdge
	triggerAdj := map[string][]string{}
	selfTrigger := map[string]bool{}
	inboundTrigger := map[string]bool{}
	outboundTrigger := map[string]bool{}

	for _, a := range sorted {
		for _, b := range sorted {
			if triggerable(RoleInput{Role: b}) && anyOverlap(a.WritePatterns, inbox[b.NotePath]) {
				edges = append(edges, RGEdge{From: a.NotePath, To: b.NotePath, Kind: edgeTrigger, Exact: true})
				inboundTrigger[b.NotePath] = true
				outboundTrigger[a.NotePath] = true
				if a.NotePath == b.NotePath {
					selfTrigger[a.NotePath] = true
				} else {
					triggerAdj[a.NotePath] = append(triggerAdj[a.NotePath], b.NotePath)
				}
				continue
			}
			if a.NotePath == b.NotePath {
				continue // self-dataflow is noise; self-trigger handled above
			}
			if anyOverlap(a.WritePatterns, b.ReadPatterns) {
				edges = append(edges, RGEdge{From: a.NotePath, To: b.NotePath, Kind: edgeDataflow, Exact: true})
			}
		}
	}

	nodes := make([]RGNode, 0, len(sorted))
	for _, r := range sorted {
		nodes = append(nodes, RGNode{
			Role:       r.NotePath,
			InboxGlob:  inbox[r.NotePath],
			OutboxGlob: r.WritePatterns,
			Orphan:     !inboundTrigger[r.NotePath] || !outboundTrigger[r.NotePath],
		})
	}

	return RoleGraph{Nodes: nodes, Edges: edges, Cycles: deriveCycles(triggerAdj, selfTrigger)}
}

// deriveCycles returns the trigger-loop cycles: multi-node SCCs and self-trigger
// singletons, each sorted, with the cycle list ordered by its first member.
func deriveCycles(triggerAdj map[string][]string, selfTrigger map[string]bool) [][]string {
	var cycles [][]string
	for _, scc := range stronglyConnected(triggerAdj) {
		if len(scc) < 2 {
			continue
		}
		slices.Sort(scc)
		cycles = append(cycles, scc)
	}
	for path := range selfTrigger {
		cycles = append(cycles, []string{path})
	}
	slices.SortFunc(cycles, func(a, b []string) int { return strings.Compare(a[0], b[0]) })
	return cycles
}

// effectiveInbox is trigger_include with any pattern that appears verbatim in
// trigger_exclude removed. It is a v1 string-level difference: a glob in
// trigger_exclude that only partially covers an include is left in place
// (conservative — keeps the possibly-spurious trigger edge rather than hiding a
// loop). It doubles as the node's inboxGlob.
func effectiveInbox(r fleet.Role) []string {
	if len(r.TriggerExclude) == 0 {
		return r.TriggerInclude
	}
	var out []string
	for _, inc := range r.TriggerInclude {
		if !slices.Contains(r.TriggerExclude, inc) {
			out = append(out, inc)
		}
	}
	return out
}

// anyOverlap reports whether any write pattern overlaps any match pattern.
func anyOverlap(writes, matches []string) bool {
	for _, w := range writes {
		for _, m := range matches {
			if Overlap(w, m) {
				return true
			}
		}
	}
	return false
}
