package fleet

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/Khan/genqlient/graphql"

	"trip2g/internal/fleet/trip2ggql"
)

// Reconciler drives the desired-vs-actual webhook diff over the genqlient admin
// lane.
type Reconciler struct {
	gql graphql.Client
	cfg Config
}

// NewReconciler builds a Reconciler over the genqlient admin lane.
func NewReconciler(gql graphql.Client, cfg Config) *Reconciler {
	return &Reconciler{gql: gql, cfg: cfg}
}

// Reconcile makes the registered change-webhooks and cron-webhooks match
// desired roles. Foreign webhooks (description without this fleet's prefix)
// are never touched.
func (r *Reconciler) Reconcile(ctx context.Context, roles []Role) error {
	if err := r.reconcileChange(ctx, roles); err != nil {
		return err
	}
	return r.reconcileCron(ctx, roles)
}

// reconcileChange syncs change-webhooks for change-mode and both-mode roles.
func (r *Reconciler) reconcileChange(ctx context.Context, roles []Role) error {
	existing, err := r.listOwned(ctx)
	if err != nil {
		return err
	}
	desired := map[string]Role{} // marker -> role
	for _, role := range roles {
		if role.Mode != modeChange && role.Mode != modeBoth {
			continue
		}
		desired[markerFor(r.cfg.FleetID, role)] = role
	}

	// Delete owned webhooks whose marker is no longer desired (also drops extras
	// sharing this fleet's /<h>/ from a superseded spec version).
	for marker, have := range existing {
		if _, keep := desired[marker]; !keep {
			if derr := r.delete(ctx, have.id); derr != nil {
				return derr
			}
		}
	}
	// Create webhooks for desired markers not yet present; take over stale ones.
	for marker, role := range desired {
		have, present := existing[marker]
		if !present {
			if cerr := r.create(ctx, role); cerr != nil {
				return cerr
			}
			continue
		}
		// Marker present => spec unchanged (ver is content-derived). But a
		// restarted/moved fleet with the same fleet_id computes the same marker
		// yet the stored url may point at the old address. Re-claim it
		// (last-writer-wins takeover) by pointing the /<h>/ webhook at this
		// fleet's current delivery url. An empty stored url means the row shape
		// carries no url to compare (test fakes), so leave it alone.
		want := changeDeliveryURL(r.cfg.CallbackURL, r.cfg.FleetID, role.NotePath)
		if have.url != "" && have.url != want {
			if uerr := r.update(ctx, have.id, want); uerr != nil {
				return uerr
			}
		}
	}
	return nil
}

// reconcileCron syncs cron-webhooks for cron-mode and both-mode roles.
func (r *Reconciler) reconcileCron(ctx context.Context, roles []Role) error {
	existing, err := r.listOwnedCron(ctx)
	if err != nil {
		return err
	}
	desired := map[string]Role{} // cronMarker -> role
	for _, role := range roles {
		if role.Mode != modeCron && role.Mode != modeBoth {
			continue
		}
		desired[cronMarkerFor(r.cfg.FleetID, role)] = role
	}

	// Delete owned cron-webhooks whose marker is no longer desired.
	for marker, id := range existing {
		if _, keep := desired[marker]; !keep {
			if derr := r.deleteCron(ctx, id); derr != nil {
				return derr
			}
		}
	}
	// Create cron-webhooks for desired markers not yet present.
	for marker, role := range desired {
		if _, present := existing[marker]; present {
			continue // marker already matches => spec unchanged
		}
		if cerr := r.createCron(ctx, role); cerr != nil {
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
	for _, have := range existing {
		if derr := r.delete(ctx, have.id); derr != nil {
			return derr
		}
	}
	existingCron, err := r.listOwnedCron(ctx)
	if err != nil {
		return err
	}
	for _, id := range existingCron {
		if derr := r.deleteCron(ctx, id); derr != nil {
			return derr
		}
	}
	return nil
}

// ownedWebhook is a listed change-webhook this fleet owns (by description
// marker): its id (for update/delete) and current delivery url (for takeover).
type ownedWebhook struct {
	id  int64
	url string
}

func (r *Reconciler) listOwned(ctx context.Context) (map[string]ownedWebhook, error) {
	resp, err := trip2ggql.ListChangeWebhooks(ctx, r.gql)
	if err != nil {
		return nil, err
	}
	prefix := "fleet:" + r.cfg.FleetID + ":"
	owned := map[string]ownedWebhook{}
	for _, n := range resp.Admin.AllChangeWebhooks.Nodes {
		if strings.HasPrefix(n.Description, prefix) {
			owned[n.Description] = ownedWebhook{id: n.Id, url: n.Url}
		}
	}
	return owned, nil
}

func (r *Reconciler) create(ctx context.Context, role Role) error {
	input := trip2ggql.ChangeWebhookCreateInput{
		Url:              changeDeliveryURL(r.cfg.CallbackURL, r.cfg.FleetID, role.NotePath),
		IncludePatterns:  orEmpty(role.TriggerInclude),
		ExcludePatterns:  orEmpty(role.TriggerExclude),
		ReadPatterns:     orEmpty(role.ReadPatterns),
		WritePatterns:    orEmpty(role.WritePatterns),
		AttachNotes:      orEmpty(role.AttachNotes),
		TransformJsonnet: "",
		ConcurrencyMode:  orDefault(role.Concurrency, concurrencyAllowOverlap),
		PassApiKey:       true,
		IncludeContent:   true,
		MaxDepth:         int64(role.MaxDepth),
		// TimeoutSeconds tells trip2g how long to wait for a delivery before
		// closing the connection. An LLM agent run can exceed the 60s default, so
		// register the role's (defaulted) timeout to avoid a mid-run cancel.
		TimeoutSeconds: int64(role.EffectiveTimeoutSeconds()),
		OnCreate:       contains(role.TriggerOn, "create"),
		OnUpdate:       contains(role.TriggerOn, "update"),
		OnRemove:       contains(role.TriggerOn, "remove"),
		Description:    markerFor(r.cfg.FleetID, role),
		Secret:         deriveSecret(r.cfg.FleetSecret, r.cfg.FleetID, role.NotePath, specVer(role)),
	}
	resp, err := trip2ggql.CreateChangeWebhook(ctx, r.gql, input)
	if err != nil {
		return err
	}
	// F9(c): surface a GraphQL-level ErrorPayload instead of swallowing it.
	if ep, ok := resp.Admin.ChangeWebhookCreate.(*trip2ggql.CreateChangeWebhookAdminAdminMutationChangeWebhookCreateErrorPayload); ok {
		return fmt.Errorf("changeWebhookCreate: %s", ep.Message)
	}
	return nil
}

func (r *Reconciler) delete(ctx context.Context, id int64) error {
	resp, err := trip2ggql.DeleteChangeWebhook(ctx, r.gql, trip2ggql.ChangeWebhookDeleteInput{Id: id})
	if err != nil {
		return err
	}
	// F9(c): surface a GraphQL-level ErrorPayload instead of swallowing it.
	if ep, ok := resp.Admin.ChangeWebhookDelete.(*trip2ggql.DeleteChangeWebhookAdminAdminMutationChangeWebhookDeleteErrorPayload); ok {
		return fmt.Errorf("changeWebhookDelete: %s", ep.Message)
	}
	return nil
}

// update re-points an existing owned change-webhook at url (hash-registry
// takeover). Only id+url are set, so trip2g updates the delivery target and
// leaves the rest of the webhook spec untouched.
func (r *Reconciler) update(ctx context.Context, id int64, url string) error {
	resp, err := trip2ggql.UpdateChangeWebhook(ctx, r.gql, trip2ggql.ChangeWebhookUpdateInput{Id: id, Url: url})
	if err != nil {
		return err
	}
	if ep, ok := resp.Admin.ChangeWebhookUpdate.(*trip2ggql.UpdateChangeWebhookAdminAdminMutationChangeWebhookUpdateErrorPayload); ok {
		return fmt.Errorf("changeWebhookUpdate: %s", ep.Message)
	}
	return nil
}

func (r *Reconciler) listOwnedCron(ctx context.Context) (map[string]int64, error) {
	resp, err := trip2ggql.ListCronWebhooks(ctx, r.gql)
	if err != nil {
		return nil, err
	}
	prefix := "fleetcron:" + r.cfg.FleetID + ":"
	owned := map[string]int64{}
	for _, n := range resp.Admin.AllCronWebhooks.Nodes {
		if strings.HasPrefix(n.Description, prefix) {
			owned[n.Description] = n.Id
		}
	}
	return owned, nil
}

func (r *Reconciler) createCron(ctx context.Context, role Role) error {
	input := trip2ggql.CreateCronWebhookInput{
		Url:             cronDeliveryURL(r.cfg.CallbackURL, r.cfg.FleetID, role.NotePath),
		CronSchedule:    role.CronSchedule,
		ReadPatterns:    orEmpty(role.ReadPatterns),
		WritePatterns:   orEmpty(role.WritePatterns),
		AttachNotes:     orEmpty(role.AttachNotes),
		ConcurrencyMode: orDefault(role.Concurrency, "allow_overlap"),
		MaxDepth:        int64(role.MaxDepth),
		TimeoutSeconds:  int64(role.EffectiveTimeoutSeconds()),
		PassApiKey:      true,
		Enabled:         true,
		Description:     cronMarkerFor(r.cfg.FleetID, role),
		Secret:          deriveSecret(r.cfg.FleetSecret, r.cfg.FleetID, role.NotePath, cronSpecVer(role)),
	}
	resp, err := trip2ggql.CreateCronWebhook(ctx, r.gql, input)
	if err != nil {
		return err
	}
	if ep, ok := resp.Admin.CreateCronWebhook.(*trip2ggql.CreateCronWebhookAdminAdminMutationCreateCronWebhookErrorPayload); ok {
		return fmt.Errorf("createCronWebhook: %s", ep.Message)
	}
	return nil
}

func (r *Reconciler) deleteCron(ctx context.Context, id int64) error {
	resp, err := trip2ggql.DeleteCronWebhook(ctx, r.gql, trip2ggql.DeleteCronWebhookInput{Id: id})
	if err != nil {
		return err
	}
	if ep, ok := resp.Admin.DeleteCronWebhook.(*trip2ggql.DeleteCronWebhookAdminAdminMutationDeleteCronWebhookErrorPayload); ok {
		return fmt.Errorf("deleteCronWebhook: %s", ep.Message)
	}
	return nil
}

// cronMarkerFor is the cron reconcile dedup key stored in the cron-webhook description.
// Uses "fleetcron:" prefix to distinguish from change-webhook markers ("fleet:").
func cronMarkerFor(fleetID string, role Role) string {
	return "fleetcron:" + fleetID + ":" + role.NotePath + "#" + cronSpecVer(role)
}

// cronSpecVer is a short content hash of the cron-webhook-relevant role fields.
// Bumping any field rotates the marker (delete+recreate).
func cronSpecVer(role Role) string {
	h := sha256.Sum256([]byte(strings.Join([]string{
		role.CronSchedule,
		strings.Join(role.ReadPatterns, ","),
		strings.Join(role.WritePatterns, ","),
		strings.Join(role.AttachNotes, ","),
		role.Concurrency,
		strconv.Itoa(role.MaxDepth),
		strconv.Itoa(role.EffectiveTimeoutSeconds()),
	}, "|")))
	return base64.RawURLEncoding.EncodeToString(h[:6])
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
		strconv.Itoa(role.MaxDepth),
		role.Concurrency,
		// Fold the effective timeout in so changing timeout_seconds rotates the
		// marker and the webhook is recreated with the new timeoutSeconds.
		strconv.Itoa(role.EffectiveTimeoutSeconds()),
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
