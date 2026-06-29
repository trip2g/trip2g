package fleet

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Reconciler drives the desired-vs-actual webhook diff over the admin lane.
type Reconciler struct {
	client Client
	cfg    Config
}

// NewReconciler builds a Reconciler.
func NewReconciler(client Client, cfg Config) *Reconciler {
	return &Reconciler{client: client, cfg: cfg}
}

const listChangeWebhooksQuery = `query { allChangeWebhooks { nodes { id description } } }`

const createChangeWebhookMutation = `mutation Create($input: ChangeWebhookCreateInput!) {
  changeWebhookCreate(input: $input) { ... on ChangeWebhookCreatePayload { webhook { id } } ... on ErrorPayload { message } }
}`

const deleteChangeWebhookMutation = `mutation Delete($input: ChangeWebhookDeleteInput!) {
  changeWebhookDelete(input: $input) { ... on ChangeWebhookDeletePayload { deletedId } ... on ErrorPayload { message } }
}`

type existingWebhook struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
}

// Reconcile makes the registered change-webhooks match desired roles. Foreign
// webhooks (description without this fleet's prefix) are never touched.
func (r *Reconciler) Reconcile(ctx context.Context, roles []Role) error {
	existing, err := r.listOwned(ctx)
	if err != nil {
		return err
	}
	desired := map[string]Role{} // marker -> role
	for _, role := range roles {
		if role.Mode != "change" && role.Mode != "both" {
			continue
		}
		desired[markerFor(r.cfg.FleetID, role)] = role
	}

	// Delete owned webhooks whose marker is no longer desired.
	for marker, id := range existing {
		if _, keep := desired[marker]; !keep {
			if derr := r.delete(ctx, id); derr != nil {
				return derr
			}
		}
	}
	// Create webhooks for desired markers not yet present.
	for marker, role := range desired {
		if _, present := existing[marker]; present {
			continue // marker already matches => spec unchanged (ver is content-derived)
		}
		if cerr := r.create(ctx, role); cerr != nil {
			return cerr
		}
	}
	return nil
}

// Deregister removes every webhook owned by this fleet (shutdown).
func (r *Reconciler) Deregister(ctx context.Context) error {
	existing, err := r.listOwned(ctx)
	if err != nil {
		return err
	}
	for _, id := range existing {
		if derr := r.delete(ctx, id); derr != nil {
			return derr
		}
	}
	return nil
}

func (r *Reconciler) listOwned(ctx context.Context) (map[string]int64, error) {
	raw, err := r.client.GraphQLAdmin(ctx, listChangeWebhooksQuery, nil)
	if err != nil {
		return nil, err
	}
	var data struct {
		AllChangeWebhooks struct {
			Nodes []existingWebhook `json:"nodes"`
		} `json:"allChangeWebhooks"`
	}
	if uerr := json.Unmarshal(raw, &data); uerr != nil {
		return nil, uerr
	}
	prefix := "fleet:" + r.cfg.FleetID + ":"
	owned := map[string]int64{}
	for _, n := range data.AllChangeWebhooks.Nodes {
		if strings.HasPrefix(n.Description, prefix) {
			owned[n.Description] = n.ID
		}
	}
	return owned, nil
}

func (r *Reconciler) create(ctx context.Context, role Role) error {
	input := map[string]any{
		"url":              r.cfg.CallbackURL + "/deliver/" + urlKey(role.NotePath),
		"includePatterns":  orEmpty(role.TriggerInclude),
		"excludePatterns":  orEmpty(role.TriggerExclude),
		"readPatterns":     orEmpty(role.ReadPatterns),
		"writePatterns":    orEmpty(role.WritePatterns),
		"attachNotes":      orEmpty(role.AttachNotes),
		"transformJsonnet": "",
		"concurrencyMode":  orDefault(role.Concurrency, "allow_overlap"),
		"passApiKey":       true,
		"includeContent":   true,
		"maxDepth":         int64(role.MaxDepth),
		"onCreate":         contains(role.TriggerOn, "create"),
		"onUpdate":         contains(role.TriggerOn, "update"),
		"onRemove":         contains(role.TriggerOn, "remove"),
		"description":      markerFor(r.cfg.FleetID, role),
		"secret":           deriveSecret(r.cfg.FleetSecret, r.cfg.FleetID, role.NotePath, specVer(role)),
	}
	raw, err := r.client.GraphQLAdmin(ctx, createChangeWebhookMutation, map[string]any{"input": input})
	if err != nil {
		return err
	}
	// F9(c): surface GraphQL-level ErrorPayload instead of swallowing it.
	var resp struct {
		ChangeWebhookCreate struct {
			Message string `json:"message"`
		} `json:"changeWebhookCreate"`
	}
	if uerr := json.Unmarshal(raw, &resp); uerr == nil && resp.ChangeWebhookCreate.Message != "" {
		return fmt.Errorf("changeWebhookCreate: %s", resp.ChangeWebhookCreate.Message)
	}
	return nil
}

func (r *Reconciler) delete(ctx context.Context, id int64) error {
	raw, err := r.client.GraphQLAdmin(ctx, deleteChangeWebhookMutation,
		map[string]any{"input": map[string]any{"id": id}})
	if err != nil {
		return err
	}
	// F9(c): surface GraphQL-level ErrorPayload instead of swallowing it.
	var resp struct {
		ChangeWebhookDelete struct {
			Message string `json:"message"`
		} `json:"changeWebhookDelete"`
	}
	if uerr := json.Unmarshal(raw, &resp); uerr == nil && resp.ChangeWebhookDelete.Message != "" {
		return fmt.Errorf("changeWebhookDelete: %s", resp.ChangeWebhookDelete.Message)
	}
	return nil
}

// markerFor is the reconcile dedup key stored in the webhook description.
func markerFor(fleetID string, role Role) string {
	return "fleet:" + fleetID + ":" + role.NotePath + "#" + specVer(role)
}

// specSchemaVer is a manual schema-version token folded into the specVer hash.
// Bump it when an always-on create() constant changes (e.g. includeContent,
// passApiKey, transformJsonnet) so every marker rotates exactly once and all
// existing webhooks are recreated to pick up the new shape. Bump=2 rotates the
// markers that predate the include_content/passApiKey constants.
const specSchemaVer = "schema=2"

// specVer is a short content hash of the parts of a role that define its
// reconciled webhook; bumping any of them rotates the marker (delete+recreate).
// specSchemaVer is folded in so always-on constants can force a one-time rotate.
func specVer(role Role) string {
	h := sha256.Sum256([]byte(strings.Join([]string{
		specSchemaVer,
		strings.Join(role.TriggerInclude, ","),
		strings.Join(role.TriggerExclude, ","),
		strings.Join(role.ReadPatterns, ","),
		strings.Join(role.WritePatterns, ","),
		strings.Join(role.AttachNotes, ","),
		strings.Join(role.TriggerOn, ","),
		fmt.Sprintf("%d", role.MaxDepth),
		role.Concurrency,
	}, "|")))
	return base64.RawURLEncoding.EncodeToString(h[:6])
}

// urlKey encodes a note path into a URL-safe delivery key.
func urlKey(notePath string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(notePath))
}

// deriveSecret produces a rotatable per-role HMAC secret.
func deriveSecret(fleetSecret, fleetID, notePath, ver string) string {
	mac := hmac.New(sha256.New, []byte(fleetSecret))
	mac.Write([]byte(fleetID + ":" + notePath + ":" + ver))
	return hex.EncodeToString(mac.Sum(nil))
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
