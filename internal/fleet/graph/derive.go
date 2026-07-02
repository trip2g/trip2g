package graph

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// Role mode values (mirrors internal/fleet role modes).
const (
	modeChange = "change"
	modeCron   = "cron"
	modeBoth   = "both"
)

// Role is the graph's view of a parsed role note. The caller (fleet daemon or
// CLI) maps its own role type into this shape; the package stays IO-free.
type Role struct {
	NotePath       string
	FleetID        string // owning fleet; "" = unclaimed
	Mode           string // "change" | "cron" | "both"
	Executor       string
	ForEach        string
	Concurrency    string
	MaxDepth       int
	CronSchedule   string
	TriggerInclude []string
	TriggerExclude []string
	TriggerOn      []string
	ReadPatterns   []string
	WritePatterns  []string
	Errors         []string // parse/validate errors; non-empty = invalid role
}

// Marker is one registry ownership marker parsed from a webhook description
// (fleet:<id>:<path>#<ver> or fleetcron:<id>:<path>#<ver>).
type Marker struct {
	FleetID  string
	NotePath string
	Cron     bool
}

// Input is everything Derive needs. Markers/HasRegistry are optional: without
// registry data, registered/drift/conflict-by-marker derivation is skipped.
type Input struct {
	Roles       []Role
	Markers     []Marker
	HasRegistry bool
	LooseErrors []string // parse errors not attributable to a note (kept as findings)
	GeneratedAt time.Time
}

// Graph is the derived dependency graph (the /graph.json payload).
type Graph struct {
	GeneratedAt time.Time `json:"generated_at"`
	Fleets      []Fleet   `json:"fleets"`
	Nodes       []Node    `json:"nodes"`
	Edges       []Edge    `json:"edges"`
	Findings    []Finding `json:"findings"`
}

// Fleet is one owning fleet seen in roles or registry markers.
type Fleet struct {
	ID string `json:"id"`
}

// Node is one role note (or a ghost node for a stale registry marker).
type Node struct {
	NotePath       string   `json:"note_path"`
	FleetIDs       []string `json:"fleet_ids"`
	Mode           string   `json:"mode"`
	Executor       string   `json:"executor,omitempty"`
	ForEach        string   `json:"for_each,omitempty"`
	Concurrency    string   `json:"concurrency,omitempty"`
	MaxDepth       int      `json:"max_depth"`
	CronSchedule   string   `json:"cron_schedule,omitempty"`
	TriggerInclude []string `json:"trigger_include,omitempty"`
	TriggerExclude []string `json:"trigger_exclude,omitempty"`
	ReadPatterns   []string `json:"read_patterns,omitempty"`
	WritePatterns  []string `json:"write_patterns,omitempty"`
	Registered     bool     `json:"registered"`
	Valid          bool     `json:"valid"`
	Ghost          bool     `json:"ghost,omitempty"` // registry marker with no current role
	Errors         []string `json:"errors"`
}

// Edge is a derived dependency: From's writes activate (trigger) or feed To.
type Edge struct {
	From          string  `json:"from"`
	To            string  `json:"to"`
	Kind          string  `json:"kind"` // "trigger" | "feed"
	Via           EdgeVia `json:"via"`
	Fanout        bool    `json:"fanout"`
	MaybeExcluded bool    `json:"maybe_excluded"`
}

// EdgeVia is the witnessing pattern pair behind an edge.
type EdgeVia struct {
	Write string `json:"write"`
	Match string `json:"match"`
}

// Finding is one flagged problem (or informational observation).
type Finding struct {
	Severity string   `json:"severity"` // "error" | "warn" | "info"
	Kind     string   `json:"kind"`     // cycle | self-trigger | conflict | unclaimed-role | orphan-write | dangling-trigger | drift | invalid-role
	Nodes    []string `json:"nodes"`
	Detail   string   `json:"detail"`
}

// Derive builds the dependency graph from parsed roles and optional registry
// markers. Pure: same input, same output (nodes, edges, findings ordered
// deterministically).
func Derive(in Input) Graph {
	g := Graph{GeneratedAt: in.GeneratedAt}
	roles := slices.Clone(in.Roles)
	slices.SortFunc(roles, func(a, b Role) int { return strings.Compare(a.NotePath, b.NotePath) })

	g.Edges = deriveEdges(roles)
	g.Nodes = deriveNodes(roles, in.Markers, in.HasRegistry)
	g.Fleets = deriveFleets(g.Nodes)
	g.Findings = deriveFindings(roles, g, in)
	return g
}

// triggerable reports whether the role registers a change webhook that fires
// on agent-producible events (agents create/update, never remove in v1).
func triggerable(r Role) bool {
	if r.Mode != modeChange && r.Mode != modeBoth {
		return false
	}
	return slices.Contains(r.TriggerOn, "create") || slices.Contains(r.TriggerOn, "update")
}

// overlapAnyPair returns the first (write, match) pair that overlaps, if any.
func overlapAnyPair(writes, matches []string) (EdgeVia, bool) {
	for _, w := range writes {
		for _, m := range matches {
			if Overlap(w, m) {
				return EdgeVia{Write: w, Match: m}, true
			}
		}
	}
	return EdgeVia{}, false
}

func deriveEdges(roles []Role) []Edge {
	var edges []Edge
	for _, a := range roles {
		for _, b := range roles {
			if via, ok := triggerEdge(a, b); ok {
				edges = append(edges, via)
				continue
			}
			if a.NotePath == b.NotePath {
				continue // self-feed is noise; self-trigger handled above
			}
			if via, ok := overlapAnyPair(a.WritePatterns, b.ReadPatterns); ok {
				edges = append(edges, Edge{
					From: a.NotePath, To: b.NotePath, Kind: "feed",
					Via: via, Fanout: a.ForEach != "",
				})
			}
		}
	}
	return edges
}

func triggerEdge(a, b Role) (Edge, bool) {
	if !triggerable(b) {
		return Edge{}, false
	}
	via, ok := overlapAnyPair(a.WritePatterns, b.TriggerInclude)
	if !ok {
		return Edge{}, false
	}
	_, maybeExcluded := overlapAnyPair(a.WritePatterns, b.TriggerExclude)
	return Edge{
		From: a.NotePath, To: b.NotePath, Kind: "trigger",
		Via: via, Fanout: a.ForEach != "", MaybeExcluded: maybeExcluded,
	}, true
}

func deriveNodes(roles []Role, markers []Marker, hasRegistry bool) []Node {
	markerFleets := map[string][]string{} // notePath -> fleet ids from markers
	changeMarked := map[string]map[string]bool{}
	cronMarked := map[string]map[string]bool{}
	for _, m := range markers {
		if !slices.Contains(markerFleets[m.NotePath], m.FleetID) {
			markerFleets[m.NotePath] = append(markerFleets[m.NotePath], m.FleetID)
		}
		set := changeMarked
		if m.Cron {
			set = cronMarked
		}
		if set[m.NotePath] == nil {
			set[m.NotePath] = map[string]bool{}
		}
		set[m.NotePath][m.FleetID] = true
	}

	var nodes []Node
	seen := map[string]bool{}
	for _, r := range roles {
		seen[r.NotePath] = true
		fleets := markerFleets[r.NotePath]
		if r.FleetID != "" && !slices.Contains(fleets, r.FleetID) {
			fleets = append(fleets, r.FleetID)
		}
		slices.Sort(fleets)
		nodes = append(nodes, Node{
			NotePath:       r.NotePath,
			FleetIDs:       fleets,
			Mode:           r.Mode,
			Executor:       r.Executor,
			ForEach:        r.ForEach,
			Concurrency:    r.Concurrency,
			MaxDepth:       r.MaxDepth,
			CronSchedule:   r.CronSchedule,
			TriggerInclude: r.TriggerInclude,
			TriggerExclude: r.TriggerExclude,
			ReadPatterns:   r.ReadPatterns,
			WritePatterns:  r.WritePatterns,
			Registered:     hasRegistry && roleRegistered(r, changeMarked, cronMarked),
			Valid:          len(r.Errors) == 0,
			Errors:         r.Errors,
		})
	}
	// Ghost nodes: markers whose note path no longer resolves to a role.
	ghostPaths := []string{}
	for path := range markerFleets {
		if !seen[path] {
			ghostPaths = append(ghostPaths, path)
		}
	}
	slices.Sort(ghostPaths)
	for _, path := range ghostPaths {
		fleets := slices.Clone(markerFleets[path])
		slices.Sort(fleets)
		nodes = append(nodes, Node{
			NotePath: path, FleetIDs: fleets,
			Registered: true, Valid: true, Ghost: true,
		})
	}
	return nodes
}

// roleRegistered reports whether every webhook the role needs (change and/or
// cron, per mode) has a registry marker for the role's own fleet. Marker match
// is by note path only; spec-version drift self-heals on the next reconcile
// poll and is not flagged.
func roleRegistered(r Role, changeMarked, cronMarked map[string]map[string]bool) bool {
	if r.Mode == modeChange || r.Mode == modeBoth {
		if !changeMarked[r.NotePath][r.FleetID] {
			return false
		}
	}
	if r.Mode == modeCron || r.Mode == modeBoth {
		if !cronMarked[r.NotePath][r.FleetID] {
			return false
		}
	}
	return true
}

func deriveFleets(nodes []Node) []Fleet {
	var ids []string
	for _, n := range nodes {
		for _, id := range n.FleetIDs {
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	slices.Sort(ids)
	fleets := make([]Fleet, 0, len(ids))
	for _, id := range ids {
		fleets = append(fleets, Fleet{ID: id})
	}
	return fleets
}

func deriveFindings(roles []Role, g Graph, in Input) []Finding {
	var out []Finding
	out = append(out, invalidFindings(roles, in.LooseErrors)...)
	out = append(out, cycleFindings(roles, g.Edges)...)
	out = append(out, ownershipFindings(g.Nodes)...)
	out = append(out, orphanWriteFindings(roles)...)
	out = append(out, danglingTriggerFindings(roles)...)
	if in.HasRegistry {
		out = append(out, driftFindings(roles, g.Nodes)...)
	}
	return out
}

func invalidFindings(roles []Role, loose []string) []Finding {
	var out []Finding
	for _, r := range roles {
		if len(r.Errors) > 0 {
			out = append(out, Finding{
				Severity: "error", Kind: "invalid-role",
				Nodes:  []string{r.NotePath},
				Detail: strings.Join(r.Errors, "; "),
			})
		}
	}
	for _, e := range loose {
		out = append(out, Finding{Severity: "error", Kind: "invalid-role", Detail: e})
	}
	return out
}

// effectiveDepth is the depth budget the reconciler registers: MaxDepth if set,
// else the default 1.
func effectiveDepth(r Role) int {
	if r.MaxDepth > 0 {
		return r.MaxDepth
	}
	return 1
}

// fanoutOverlap reports whether the role fans out per delivery AND allows
// overlapping runs ("" defaults to allow_overlap in the reconciler).
func fanoutOverlap(r Role) bool {
	return r.ForEach != "" && (r.Concurrency == "" || r.Concurrency == "allow_overlap")
}

// cycleFindings flags self-triggers and multi-node cycles over trigger edges
// only (feed edges don't cause runs). Severity: WARN when the lap budget
// (min effective max_depth over members) is 1 and no member fans out with
// overlap allowed; ERROR otherwise (real token-burn risk).
func cycleFindings(roles []Role, edges []Edge) []Finding {
	byPath := map[string]Role{}
	for _, r := range roles {
		byPath[r.NotePath] = r
	}
	adj := map[string][]string{}
	var out []Finding
	for _, e := range edges {
		if e.Kind != "trigger" {
			continue
		}
		if e.From == e.To {
			r := byPath[e.From]
			out = append(out, Finding{
				Severity: cycleSeverity([]Role{r}),
				Kind:     "self-trigger",
				Nodes:    []string{e.From},
				Detail: fmt.Sprintf("role's writes (%s) overlap its own trigger_include (%s); lap budget %d",
					e.Via.Write, e.Via.Match, effectiveDepth(r)),
			})
			continue
		}
		adj[e.From] = append(adj[e.From], e.To)
	}
	for _, scc := range stronglyConnected(adj) {
		if len(scc) < 2 {
			continue
		}
		slices.Sort(scc)
		members := make([]Role, 0, len(scc))
		for _, p := range scc {
			members = append(members, byPath[p])
		}
		lap := effectiveDepth(members[0])
		for _, m := range members[1:] {
			lap = min(lap, effectiveDepth(m))
		}
		out = append(out, Finding{
			Severity: cycleSeverity(members),
			Kind:     "cycle",
			Nodes:    scc,
			Detail:   fmt.Sprintf("trigger cycle of %d roles; lap budget %d", len(scc), lap),
		})
	}
	return out
}

func cycleSeverity(members []Role) string {
	lap := effectiveDepth(members[0])
	fanout := false
	for _, m := range members {
		lap = min(lap, effectiveDepth(m))
		if fanoutOverlap(m) {
			fanout = true
		}
	}
	if lap > 1 || fanout {
		return "error"
	}
	return "warn"
}

// stronglyConnected returns Tarjan SCCs of size >= 2 (plus singletons, which
// callers filter) for the adjacency map.
func stronglyConnected(adj map[string][]string) [][]string {
	index := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	next := 0
	var sccs [][]string

	var strong func(v string)
	strong = func(v string) {
		index[v] = next
		low[v] = next
		next++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range adj[v] {
			if _, seen := index[w]; !seen {
				strong(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] == index[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}
	var nodes []string
	for v := range adj {
		nodes = append(nodes, v)
	}
	slices.Sort(nodes)
	for _, v := range nodes {
		if _, seen := index[v]; !seen {
			strong(v)
		}
	}
	return sccs
}

func ownershipFindings(nodes []Node) []Finding {
	var out []Finding
	for _, n := range nodes {
		if n.Ghost {
			continue
		}
		switch {
		case len(n.FleetIDs) > 1:
			out = append(out, Finding{
				Severity: "error", Kind: "conflict",
				Nodes:  []string{n.NotePath},
				Detail: fmt.Sprintf("role claimed by %d fleets: %s (duplicate execution)", len(n.FleetIDs), strings.Join(n.FleetIDs, ", ")),
			})
		case len(n.FleetIDs) == 0:
			out = append(out, Finding{
				Severity: "warn", Kind: "unclaimed-role",
				Nodes:  []string{n.NotePath},
				Detail: "role note owned by no fleet (folder not covered by any fleet)",
			})
		}
	}
	return out
}

// orphanWriteFindings flags write patterns no OTHER role's trigger_include (of
// a triggerable role) or read_patterns overlaps. Often fine (logs,
// human-facing output) — INFO only.
func orphanWriteFindings(roles []Role) []Finding {
	var out []Finding
	for _, a := range roles {
		var orphans []string
		for _, w := range a.WritePatterns {
			consumed := false
			for _, b := range roles {
				if b.NotePath == a.NotePath {
					continue
				}
				if triggerable(b) {
					if _, ok := overlapAnyPair([]string{w}, b.TriggerInclude); ok {
						consumed = true
						break
					}
				}
				if _, ok := overlapAnyPair([]string{w}, b.ReadPatterns); ok {
					consumed = true
					break
				}
			}
			if !consumed {
				orphans = append(orphans, w)
			}
		}
		if len(orphans) > 0 {
			out = append(out, Finding{
				Severity: "info", Kind: "orphan-write",
				Nodes:  []string{a.NotePath},
				Detail: "writes nothing reads or triggers: " + strings.Join(orphans, ", "),
			})
		}
	}
	return out
}

// danglingTriggerFindings flags trigger_include patterns no role writes into —
// rendered as human entry points, not errors.
func danglingTriggerFindings(roles []Role) []Finding {
	var out []Finding
	for _, b := range roles {
		if !triggerable(b) {
			continue
		}
		var dangling []string
		for _, t := range b.TriggerInclude {
			written := false
			for _, a := range roles {
				if _, ok := overlapAnyPair(a.WritePatterns, []string{t}); ok {
					written = true
					break
				}
			}
			if !written {
				dangling = append(dangling, t)
			}
		}
		if len(dangling) > 0 {
			out = append(out, Finding{
				Severity: "info", Kind: "dangling-trigger",
				Nodes:  []string{b.NotePath},
				Detail: "human entry point (no role writes here): " + strings.Join(dangling, ", "),
			})
		}
	}
	return out
}

func driftFindings(roles []Role, nodes []Node) []Finding {
	registered := map[string]bool{}
	for _, n := range nodes {
		registered[n.NotePath] = n.Registered
	}
	var out []Finding
	for _, n := range nodes {
		if n.Ghost {
			out = append(out, Finding{
				Severity: "warn", Kind: "drift",
				Nodes:  []string{n.NotePath},
				Detail: "registered webhook has no current role (stale marker: fleet down or keep-webhooks leftover)",
			})
		}
	}
	for _, r := range roles {
		if len(r.Errors) > 0 {
			continue // invalid roles are never registered; flagged separately
		}
		if !registered[r.NotePath] {
			out = append(out, Finding{
				Severity: "warn", Kind: "drift",
				Nodes:  []string{r.NotePath},
				Detail: "role has no registered webhook yet (fleet has not reconciled it)",
			})
		}
	}
	return out
}
