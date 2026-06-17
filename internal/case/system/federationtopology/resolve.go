package federationtopology

//go:generate go tool github.com/mailru/easyjson/easyjson -snake_case -all -no_std_marshalers ./resolve.go
//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg federationtopology_test . Env

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/model"
)

type Env interface {
	PublicURL() string
	LoadSiteConfig(ctx context.Context) (model.SiteConfig, error)
	ListAllSubgraphs(ctx context.Context) ([]db.Subgraph, error)
	ListFederationSecrets(ctx context.Context) ([]db.ListFederationSecretsRow, error)
	ListAllFederationSecretScopes(ctx context.Context) ([]db.ListAllFederationSecretScopesRow, error)
	LatestNoteViews() *model.NoteViews
}

type Subgraph struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Self struct {
	Name      string     `json:"name"`
	KBID      string     `json:"kb_id"`
	MCPURL    string     `json:"mcp_url"`
	Subgraphs []Subgraph `json:"subgraphs"`
}

type Secret struct {
	ID        int64      `json:"id"`
	KID       string     `json:"kid"`
	KBURL     *string    `json:"kb_url,omitempty"`
	RevokedAt *string    `json:"revoked_at"`
	Subgraphs []Subgraph `json:"subgraphs"`
}

type KBNote struct {
	KBID        string `json:"kb_id"`
	KBURL       string `json:"kb_url"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

type Response struct {
	Self     Self     `json:"self"`
	Outbound []Secret `json:"outbound"`
	Inbound  []Secret `json:"inbound"`
	KBNotes  []KBNote `json:"kb_notes"`
}

func Resolve(ctx context.Context, env Env) (*Response, error) {
	publicURL := strings.TrimRight(env.PublicURL(), "/")
	mcpURL := publicURL + "/_system/mcp"

	host := publicURL
	if u, err := url.Parse(publicURL); err == nil && u.Host != "" {
		host = u.Host
	}

	cfg, err := env.LoadSiteConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load site config: %w", err)
	}
	kbID := strings.TrimSpace(cfg.KBID)
	if kbID == "" {
		kbID = host
	}

	subs, err := env.ListAllSubgraphs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list subgraphs: %w", err)
	}
	selfSubs := make([]Subgraph, 0, len(subs))
	for _, s := range subs {
		selfSubs = append(selfSubs, Subgraph{ID: s.ID, Name: s.Name})
	}

	scopes, err := env.ListAllFederationSecretScopes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list scopes: %w", err)
	}
	scopeByKID := map[string][]Subgraph{}
	for _, sc := range scopes {
		scopeByKID[sc.Kid] = append(scopeByKID[sc.Kid], Subgraph{ID: sc.SubgraphID, Name: sc.SubgraphName})
	}

	rows, err := env.ListFederationSecrets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	resp := &Response{
		Self:     Self{Name: kbID, KBID: kbID, MCPURL: mcpURL, Subgraphs: selfSubs},
		Outbound: []Secret{},
		Inbound:  []Secret{},
		KBNotes:  []KBNote{},
	}
	for _, r := range rows {
		sec := Secret{ID: r.ID, KID: r.Kid, KBURL: r.KbUrl, Subgraphs: scopeByKID[r.Kid]}
		if sec.Subgraphs == nil {
			sec.Subgraphs = []Subgraph{}
		}
		if r.RevokedAt != nil {
			s := r.RevokedAt.UTC().Format("2006-01-02T15:04:05Z")
			sec.RevokedAt = &s
		}
		if r.KbUrl != nil {
			resp.Outbound = append(resp.Outbound, sec)
		} else {
			resp.Inbound = append(resp.Inbound, sec)
		}
	}

	views := env.LatestNoteViews()
	if views == nil {
		views = &model.NoteViews{}
	}
	for _, n := range views.MCPFederationNotes {
		note := KBNote{KBID: n.ID, KBURL: n.URL, Path: notePath(n), Description: noteDesc(n)}
		resp.KBNotes = append(resp.KBNotes, note)
	}

	return resp, nil
}

// notePath returns the KB note's source path, or "" when the note is missing.
func notePath(n *model.MCPFederationNote) string {
	if n.Note == nil {
		return ""
	}
	return n.Note.Path
}

// noteDesc returns the KB note's meta description, or "" when absent.
func noteDesc(n *model.MCPFederationNote) string {
	if n.Note == nil || n.Note.Description == nil {
		return ""
	}
	return *n.Note.Description
}
