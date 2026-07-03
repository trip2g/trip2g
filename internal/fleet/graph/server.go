package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"

	"trip2g/assets"
	"trip2g/internal/fleet"
	"trip2g/internal/fleet/trip2ggql"
)

// Server serves the fleet dependency graph on a localhost-only debug surface:
//   - GET /graph.json — machine JSON (see docs/dev/2026-07-02_fleet_dependency_graph.md)
//   - GET /mermaid.min.js — bundled Mermaid from trip2g/assets (no CDN)
//   - GET / — self-contained HTML visualization
//
// It is an introspection tool — never mount it on the public delivery listener.
type Server struct {
	discovery *fleet.Discovery
	gql       graphql.Client
	cfg       fleet.Config
}

// NewServer builds a Server over the admin lane.
func NewServer(discovery *fleet.Discovery, gql graphql.Client, cfg fleet.Config) *Server {
	return &Server{discovery: discovery, gql: gql, cfg: cfg}
}

// Handler returns the debug mux: / (UI), /graph.json (API),
// /mermaid.min.js (bundled, no CDN dependency).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/graph.json", s.serveJSON)
	mux.HandleFunc("/mermaid.min.js", serveMermaid)
	mux.HandleFunc("/", s.serveUI)
	return mux
}

func (s *Server) serveJSON(w http.ResponseWriter, r *http.Request) {
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

// serveMermaid reads mermaid.min.js out of the embedded assets FS and serves
// it — same bundle the default template uses, no network required.
func serveMermaid(w http.ResponseWriter, _ *http.Request) {
	data, err := fs.ReadFile(assets.FS, "mermaid.min.js")
	if err != nil {
		http.Error(w, "mermaid.min.js not found in embedded assets", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) serveUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(UIHTML)
}

// BuildGraph discovers roles and registry markers fresh (it is a debug
// endpoint — always current, no caching) and derives the dependency graph.
func (s *Server) BuildGraph(ctx context.Context) (Graph, error) {
	parsed, errs := s.discovery.DiscoverParsed(ctx)
	var loose []string
	for _, e := range errs {
		loose = append(loose, e.Error())
	}

	roles := make([]RoleInput, 0, len(parsed))
	for _, r := range parsed {
		var roleErrs []string
		if verr := r.Validate(s.cfg.OfferedTools); verr != nil {
			roleErrs = []string{verr.Error()}
		}
		roles = append(roles, RoleInput{
			Role:    r,
			FleetID: s.cfg.FleetID,
			Errors:  roleErrs,
		})
	}

	markers, err := s.listMarkers(ctx)
	if err != nil {
		return Graph{}, fmt.Errorf("graph: list registry markers: %w", err)
	}

	return Derive(Input{
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
func (s *Server) listMarkers(ctx context.Context) ([]Marker, error) {
	var markers []Marker
	change, err := trip2ggql.ListChangeWebhooks(ctx, s.gql)
	if err != nil {
		return nil, err
	}
	for _, n := range change.Admin.AllChangeWebhooks.Nodes {
		if m, ok := ParseFleetMarker(n.Description); ok {
			markers = append(markers, m)
		}
	}
	cron, err := trip2ggql.ListCronWebhooks(ctx, s.gql)
	if err != nil {
		return nil, err
	}
	for _, n := range cron.Admin.AllCronWebhooks.Nodes {
		if m, ok := ParseFleetMarker(n.Description); ok {
			markers = append(markers, m)
		}
	}
	return markers, nil
}

// ParseFleetMarker parses a reconcile ownership marker
// ("fleet:<id>:<path>#<ver>" / "fleetcron:<id>:<path>#<ver>") from a webhook
// description, for any fleet id. The spec version suffix is dropped: the graph
// matches markers by note path (spec drift self-heals on the next poll).
// Exported so cmd/fleet and tests can share the parsing logic.
func ParseFleetMarker(desc string) (Marker, bool) {
	// Check "fleetcron:" first — "fleet:" is its prefix.
	cron := false
	rest, ok := strings.CutPrefix(desc, "fleetcron:")
	if ok {
		cron = true
	} else {
		rest, ok = strings.CutPrefix(desc, "fleet:")
		if !ok {
			return Marker{}, false
		}
	}
	fleetID, pathVer, ok := strings.Cut(rest, ":")
	if !ok || fleetID == "" || pathVer == "" {
		return Marker{}, false
	}
	path := pathVer
	if i := strings.LastIndexByte(pathVer, '#'); i >= 0 {
		path = pathVer[:i]
	}
	if path == "" {
		return Marker{}, false
	}
	return Marker{FleetID: fleetID, NotePath: path, Cron: cron}, true
}
