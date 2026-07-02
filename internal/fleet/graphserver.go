package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"

	"trip2g/internal/fleet/graph"
	"trip2g/internal/fleet/trip2ggql"
)

// GraphServer serves the fleet dependency graph on a localhost-only debug
// surface: GET /graph.json (machine JSON, doc'd in
// docs/dev/2026-07-02_fleet_dependency_graph.md) and GET / (self-contained
// HTML visualization). It is an introspection tool — never mount it on the
// public delivery listener.
type GraphServer struct {
	discovery *Discovery
	gql       graphql.Client
	cfg       Config
}

// NewGraphServer builds a GraphServer over the admin lane.
func NewGraphServer(discovery *Discovery, gql graphql.Client, cfg Config) *GraphServer {
	return &GraphServer{discovery: discovery, gql: gql, cfg: cfg}
}

// Handler returns the debug mux: / (UI) and /graph.json (API).
func (s *GraphServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/graph.json", s.serveJSON)
	mux.HandleFunc("/", s.serveUI)
	return mux
}

func (s *GraphServer) serveJSON(w http.ResponseWriter, r *http.Request) {
	g, err := s.BuildGraph(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(g)
}

func (s *GraphServer) serveUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(graph.UIHTML)
}

// BuildGraph discovers roles and registry markers fresh (it is a debug
// endpoint — always current, no caching) and derives the dependency graph.
func (s *GraphServer) BuildGraph(ctx context.Context) (graph.Graph, error) {
	parsed, errs := s.discovery.DiscoverParsed(ctx)
	var loose []string
	for _, e := range errs {
		loose = append(loose, e.Error())
	}

	roles := make([]graph.Role, 0, len(parsed))
	for _, r := range parsed {
		var roleErrs []string
		if verr := r.Validate(s.cfg.OfferedTools); verr != nil {
			roleErrs = []string{verr.Error()}
		}
		roles = append(roles, graph.Role{
			NotePath:       r.NotePath,
			FleetID:        s.cfg.FleetID,
			Mode:           r.Mode,
			Executor:       r.Executor,
			ForEach:        r.ForEach,
			Concurrency:    r.Concurrency,
			MaxDepth:       r.MaxDepth,
			CronSchedule:   r.CronSchedule,
			TriggerInclude: r.TriggerInclude,
			TriggerExclude: r.TriggerExclude,
			TriggerOn:      r.TriggerOn,
			ReadPatterns:   r.ReadPatterns,
			WritePatterns:  r.WritePatterns,
			Errors:         roleErrs,
		})
	}

	markers, err := s.listMarkers(ctx)
	if err != nil {
		return graph.Graph{}, fmt.Errorf("graph: list registry markers: %w", err)
	}

	return graph.Derive(graph.Input{
		Roles:       roles,
		Markers:     markers,
		HasRegistry: true,
		LooseErrors: loose,
		GeneratedAt: time.Now().UTC(),
	}), nil
}

// listMarkers scans ALL change- and cron-webhook descriptions for fleet
// ownership markers of ANY fleet — the registry is the only place multi-fleet
// ownership (and conflicts) is visible from one daemon.
func (s *GraphServer) listMarkers(ctx context.Context) ([]graph.Marker, error) {
	var markers []graph.Marker
	change, err := trip2ggql.ListChangeWebhooks(ctx, s.gql)
	if err != nil {
		return nil, err
	}
	for _, n := range change.Admin.AllChangeWebhooks.Nodes {
		if m, ok := parseFleetMarker(n.Description); ok {
			markers = append(markers, m)
		}
	}
	cron, err := trip2ggql.ListCronWebhooks(ctx, s.gql)
	if err != nil {
		return nil, err
	}
	for _, n := range cron.Admin.AllCronWebhooks.Nodes {
		if m, ok := parseFleetMarker(n.Description); ok {
			markers = append(markers, m)
		}
	}
	return markers, nil
}

// parseFleetMarker parses a reconcile ownership marker
// ("fleet:<id>:<path>#<ver>" / "fleetcron:<id>:<path>#<ver>") from a webhook
// description, for any fleet id. The spec version suffix is dropped: the graph
// matches markers by note path (spec drift self-heals on the next poll).
func parseFleetMarker(desc string) (graph.Marker, bool) {
	// Check "fleetcron:" first — "fleet:" is its prefix.
	cron := false
	rest, ok := strings.CutPrefix(desc, "fleetcron:")
	if ok {
		cron = true
	} else {
		rest, ok = strings.CutPrefix(desc, "fleet:")
		if !ok {
			return graph.Marker{}, false
		}
	}
	fleetID, pathVer, ok := strings.Cut(rest, ":")
	if !ok || fleetID == "" || pathVer == "" {
		return graph.Marker{}, false
	}
	path := pathVer
	if i := strings.LastIndexByte(pathVer, '#'); i >= 0 {
		path = pathVer[:i]
	}
	if path == "" {
		return graph.Marker{}, false
	}
	return graph.Marker{FleetID: fleetID, NotePath: path, Cron: cron}, true
}
