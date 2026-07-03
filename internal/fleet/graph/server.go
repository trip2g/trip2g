package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"

	"trip2g/internal/fleet"
	"trip2g/internal/fleet/trip2ggql"
)

// Server serves the fleet dependency graph on a localhost-only debug surface:
//   - GET /graph.json — machine JSON (see docs/dev/2026-07-02_fleet_dependency_graph.md)
//   - GET / — HTML visualization (loads Mermaid from the connected trip2g hub)
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

// Handler returns the debug mux: / (UI) and /graph.json (API).
// Mermaid is loaded from the connected trip2g hub — no local bundle needed.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/graph.json", s.serveJSON)
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

func (s *Server) serveUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	tmpl, err := template.New("ui").Parse(string(UIHTML))
	if err != nil {
		http.Error(w, "template parse error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, struct{ Trip2gBaseURL string }{s.cfg.Trip2gBaseURL}); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
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
