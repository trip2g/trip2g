package delivercronwebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/jsonneteval"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/ptr"
	"trip2g/internal/shortapitoken"
	"trip2g/internal/webhookutil"

	"github.com/valyala/fasthttp"
)

//nolint:gochecknoglobals // ResponseSchema is a server constant.
var ResponseSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"status": {"type": "string"},
		"message": {"type": "string"},
		"changes": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"path": {"type": "string"},
					"content": {"type": "string"},
					"expected_hash": {"type": "string"}
				}
			}
		}
	}
}`)

type Env interface {
	CronWebhookByID(ctx context.Context, id int64) (db.CronWebhook, error)
	MarkCronWebhookDeliveryRunning(ctx context.Context, id int64) error
	UpdateCronWebhookDeliveryResult(ctx context.Context, arg db.UpdateCronWebhookDeliveryResultParams) error
	InsertWebhookDeliveryLog(ctx context.Context, arg db.InsertWebhookDeliveryLogParams) error
	InsertNote(ctx context.Context, note model.RawNote) (int64, error)
	LatestNoteViews() *model.NoteViews
	EnqueueDeliverCronWebhook(ctx context.Context, params DeliverCronParams) error
	ShortAPITokenSecret() string
	WebhookHTTPClient() *fasthttp.Client
	Logger() logger.Logger
	GetSecretValues(ctx context.Context, like string) (map[string]string, error)
}

// cronWebhookPayload is the JSON body sent to the cron webhook endpoint.
type cronWebhookPayload struct {
	webhookutil.BasePayload
	Instruction    string                     `json:"instruction"`
	ResponseSchema json.RawMessage            `json:"response_schema"`
	AttachedNotes  []webhookutil.AttachedNote `json:"attached_notes,omitempty"`
	APIToken       string                     `json:"api_token,omitempty"`
	Secrets        map[string]string          `json:"secrets,omitempty"`
	PreviousError  string                     `json:"previous_error,omitempty"`
}

// tokenTTLMargin is the small grace window added to the delivery timeout for
// the scoped write-back token. Replaces the former 60-minute floor.
const tokenTTLMargin = 30 * time.Second

//nolint:gocognit,gocyclo,cyclop,funlen // cron delivery resolver handles full lifecycle: attach-gate, fan-out, write-back, secrets
func Resolve(ctx context.Context, env Env, params DeliverCronParams) error {
	log := env.Logger()

	// Load cron webhook configuration.
	wh, err := env.CronWebhookByID(ctx, params.CronWebhookID)
	if err != nil {
		return fmt.Errorf("failed to load cron webhook %d: %w", params.CronWebhookID, err)
	}

	if params.Attempt <= 1 {
		if mErr := env.MarkCronWebhookDeliveryRunning(ctx, params.DeliveryID); mErr != nil {
			log.Error("failed to mark cron delivery running", "delivery_id", params.DeliveryID, "error", mErr)
		}
	}

	p, _ := model.CronWebhookSecretPrefix(params.CronWebhookID)
	secrets := loadCronSecrets(ctx, env, log, p.String())

	// Materialize attach_notes context and apply presence gate.
	var attachedNotes []webhookutil.AttachedNote
	if wh.AttachNotes != "" && wh.AttachNotes != "[]" { //nolint:nestif // attach-gate requires parse, nil-check, and gate evaluation
		attach, aerr := webhookutil.ParseJSONStringArray(wh.AttachNotes)
		if aerr != nil {
			log.Error("failed to parse attach_notes", "cron_webhook_id", wh.ID, "error", aerr)
		} else {
			nvs := env.LatestNoteViews()
			if !webhookutil.AttachGateSatisfied(attach, nvs) {
				log.Info("cron delivery skipped: attach_notes gate not satisfied",
					"cron_webhook_id", wh.ID, "delivery_id", params.DeliveryID)
				updateErr := env.UpdateCronWebhookDeliveryResult(ctx, db.UpdateCronWebhookDeliveryResultParams{
					Status: "success",
					ID:     params.DeliveryID,
				})
				if updateErr != nil {
					log.Error("failed to mark skipped cron delivery", "delivery_id", params.DeliveryID, "error", updateErr)
				}
				return nil
			}
			attachedNotes = webhookutil.MaterializeAttachedNotes(attach, nvs)
		}
	}

	// Build payload.
	payload := cronWebhookPayload{
		BasePayload:    webhookutil.NewBasePayload(params.DeliveryID, params.Attempt),
		Instruction:    wh.Instruction,
		ResponseSchema: ResponseSchema,
		AttachedNotes:  attachedNotes,
		Secrets:        secrets,
		PreviousError:  params.PreviousError,
	}

	// Parse write patterns for validating agent response changes.
	writePatterns, wpErr := webhookutil.ParseJSONStringArray(wh.WritePatterns)
	if wpErr != nil {
		log.Error("failed to parse write_patterns", "cron_webhook_id", wh.ID, "error", wpErr)
		writePatterns = []string{}
	}

	// Generate short API token if pass_api_key is enabled.
	if wh.PassApiKey {
		readPatterns, rpErr := webhookutil.ParseJSONStringArray(wh.ReadPatterns)
		if rpErr != nil {
			log.Error("failed to parse read_patterns", "cron_webhook_id", wh.ID, "error", rpErr)
			readPatterns = []string{} // fail closed: malformed scope grants nothing
		}

		ttl := time.Duration(wh.TimeoutSeconds)*time.Second + tokenTTLMargin

		token, signErr := shortapitoken.Sign(shortapitoken.Data{
			Depth:         1, // Cron webhooks always start at depth 1.
			ReadPatterns:  readPatterns,
			WritePatterns: writePatterns,
			DeliveryKind:  "cron",
			DeliveryID:    params.DeliveryID,
		}, env.ShortAPITokenSecret(), ttl)
		if signErr != nil {
			log.Error("failed to sign short API token", "cron_webhook_id", wh.ID, "error", signErr)
		} else {
			payload.APIToken = token
		}
	}

	// Marshal payload to JSON.
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal cron webhook payload: %w", err)
	}

	// Apply outbound transform (if configured) strictly between marshal and
	// sign, so both the HMAC and the actual sent bytes are the transformed result.
	transformed := wh.TransformJsonnet != ""
	if transformed {
		out, terr := jsonneteval.EvalJSON(wh.TransformJsonnet, transformExtVars(payloadBytes))
		if terr != nil {
			handleCronDeliveryError(ctx, env, params,
				webhookutil.DeliveryResult{Err: fmt.Errorf("transform_jsonnet: %w", terr)}, wh)
			return nil
		}
		payloadBytes = out
	}

	// Sign payload with HMAC.
	signature := webhookutil.SignHMAC(payloadBytes, wh.Secret)
	timestamp := strconv.FormatInt(payload.Timestamp, 10)

	// Build headers.
	headers := map[string]string{
		"X-Webhook-ID":        strconv.FormatInt(params.DeliveryID, 10),
		"X-Webhook-Timestamp": timestamp,
		"X-Webhook-Signature": signature,
		"X-Webhook-Attempt":   strconv.Itoa(params.Attempt),
	}

	// Send HTTP request.
	timeout := time.Duration(wh.TimeoutSeconds) * time.Second
	result := webhookutil.Deliver(env.WebhookHTTPClient(), wh.Url, payloadBytes, headers, timeout)

	// Save delivery log.
	// F9(d): when transform_jsonnet is active, payloadBytes IS the transformed
	// output (api_token/secrets never appear there — transformExtVars excludes
	// them). Log those bytes directly so the logged body matches what was sent.
	// Without a transform, log a redacted copy of the pre-marshal struct with
	// APIToken and Secrets zeroed out.
	var requestBodyStr string
	if transformed {
		requestBodyStr = string(payloadBytes)
	} else {
		redacted := payload
		redacted.APIToken = ""
		redacted.Secrets = nil
		redactedBytes, redErr := json.Marshal(redacted)
		if redErr != nil {
			redactedBytes = []byte("{}")
		}
		requestBodyStr = string(redactedBytes)
	}
	logParams := db.InsertWebhookDeliveryLogParams{
		DeliveryID:  params.DeliveryID,
		Kind:        "cron",
		RequestBody: &requestBodyStr,
	}
	if result.Body != nil {
		responseBodyStr := string(result.Body)
		logParams.ResponseBody = &responseBodyStr
	}
	if result.Err != nil {
		errMsg := result.Err.Error()
		logParams.ErrorMessage = &errMsg
	}

	logErr := env.InsertWebhookDeliveryLog(ctx, logParams)
	if logErr != nil {
		log.Error("failed to insert cron webhook delivery log", "delivery_id", params.DeliveryID, "error", logErr)
	}

	// Handle HTTP error or server error.
	//nolint:nilerr // error handled via handleCronDeliveryError, not returned.
	if result.Err != nil || result.StatusCode >= 300 {
		handleCronDeliveryError(ctx, env, params, result, wh)
		return nil
	}

	// Parse agent response for changes.
	var applyErr error
	if result.StatusCode >= 200 && result.StatusCode < 300 && result.StatusCode != http.StatusAccepted {
		applyErr = applyCronAgentChanges(ctx, env, result, writePatterns)
	}

	// Handle agent response errors with retry.
	if applyErr != nil {
		if int64(params.Attempt) < wh.MaxRetries {
			retryErr := env.EnqueueDeliverCronWebhook(ctx, DeliverCronParams{
				DeliveryID:    params.DeliveryID,
				CronWebhookID: params.CronWebhookID,
				Attempt:       params.Attempt + 1,
				PreviousError: applyErr.Error(),
			})
			if retryErr != nil {
				log.Error("failed to enqueue cron webhook retry", "delivery_id", params.DeliveryID, "error", retryErr)
			}
			return nil
		}

		log.Warn("agent response error, no retries left",
			"delivery_id", params.DeliveryID,
			"error", applyErr,
		)

		// Mark as failed when agent response error with no retries left.
		updateErr := env.UpdateCronWebhookDeliveryResult(ctx, db.UpdateCronWebhookDeliveryResultParams{
			Status:         "failed",
			ResponseStatus: ptr.To(int64(result.StatusCode)),
			DurationMs:     ptr.To(result.DurationMs),
			ID:             params.DeliveryID,
		})
		if updateErr != nil {
			log.Error("failed to update cron delivery result", "delivery_id", params.DeliveryID, "error", updateErr)
		}
		return nil
	}

	// Parse fleet-reported spend (tokens/steps) from the response body.
	var tokensUsed, steps *int64
	if resp, perr := webhookutil.ParseAgentResponse(result.Body); perr == nil && resp != nil {
		if resp.TokensUsed > 0 {
			tokensUsed = ptr.To(int64(resp.TokensUsed))
		}
		if resp.Steps > 0 {
			steps = ptr.To(int64(resp.Steps))
		}
	}

	// Mark as success.
	updateErr := env.UpdateCronWebhookDeliveryResult(ctx, db.UpdateCronWebhookDeliveryResultParams{
		Status:         "success",
		ResponseStatus: ptr.To(int64(result.StatusCode)),
		DurationMs:     ptr.To(result.DurationMs),
		TokensUsed:     tokensUsed,
		Steps:          steps,
		ID:             params.DeliveryID,
	})
	if updateErr != nil {
		log.Error("failed to update cron delivery result", "delivery_id", params.DeliveryID, "error", updateErr)
	}

	return nil
}

// loadCronSecrets fetches decrypted secrets for the given prefix, returning a map keyed by bare name.
func loadCronSecrets(ctx context.Context, env Env, log logger.Logger, prefix string) map[string]string {
	all, err := env.GetSecretValues(ctx, prefix+"%")
	if err != nil {
		log.Error("failed to load cron webhook secrets", "prefix", prefix, "error", err)
		return nil
	}
	if len(all) == 0 {
		return nil
	}
	result := make(map[string]string, len(all))
	for k, v := range all {
		result[strings.TrimPrefix(k, prefix)] = v
	}
	return result
}

// handleCronDeliveryError handles HTTP errors and retries.
func handleCronDeliveryError(ctx context.Context, env Env, params DeliverCronParams, result webhookutil.DeliveryResult, wh db.CronWebhook) {
	var errMsg string
	if result.Err != nil {
		errMsg = result.Err.Error()
	} else {
		errMsg = fmt.Sprintf("HTTP %d", result.StatusCode)
	}

	if int64(params.Attempt) < wh.MaxRetries {
		retryErr := env.EnqueueDeliverCronWebhook(ctx, DeliverCronParams{
			DeliveryID:    params.DeliveryID,
			CronWebhookID: params.CronWebhookID,
			Attempt:       params.Attempt + 1,
			PreviousError: errMsg,
		})
		if retryErr != nil {
			env.Logger().Error("failed to enqueue cron webhook retry", "delivery_id", params.DeliveryID, "error", retryErr)
		}
		return
	}

	// Mark as failed.
	updateErr := env.UpdateCronWebhookDeliveryResult(ctx, db.UpdateCronWebhookDeliveryResultParams{
		Status:         "failed",
		ResponseStatus: ptr.To(int64(result.StatusCode)),
		DurationMs:     ptr.To(result.DurationMs),
		ID:             params.DeliveryID,
	})
	if updateErr != nil {
		env.Logger().Error("failed to update cron delivery result", "delivery_id", params.DeliveryID, "error", updateErr)
	}
}

// transformExtVars exposes non-secret payload fields to the jsonnet transform.
// api_token and secrets are never exposed.
func transformExtVars(payloadBytes []byte) map[string]string {
	var p map[string]json.RawMessage
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return map[string]string{}
	}
	ev := make(map[string]string, 3)
	for _, k := range []string{"changes", "attached_notes", "meta"} {
		if v, ok := p[k]; ok {
			key := k
			if k == "changes" {
				key = "change"
			}
			ev[key] = string(v)
		}
	}
	return ev
}

// applyCronAgentChanges parses and applies agent response changes.
func applyCronAgentChanges(ctx context.Context, env Env, result webhookutil.DeliveryResult, writePatterns []string) error {
	agentResp, parseErr := webhookutil.ParseAgentResponse(result.Body)
	if parseErr != nil {
		return parseErr
	}
	if agentResp == nil || len(agentResp.Changes) == 0 {
		return nil
	}

	// Read the current note views once; needed for patch operations.
	nvs := env.LatestNoteViews()

	for _, change := range agentResp.Changes {
		// Deny-all when write_patterns is empty: a cron webhook delivery is always a
		// scoped context, so an empty list means "no writes permitted" rather
		// than "allow all". Also deny on no-match when non-empty.
		if len(writePatterns) == 0 || !webhookutil.MatchesAny(change.Path, writePatterns) {
			return fmt.Errorf("path %q not allowed by write_patterns", change.Path)
		}

		var content string
		if change.Kind == "patch" { //nolint:nestif // patch requires sequential null-checks before string ops
			// Apply find/replace against the note's current content.
			// Matches updateNotes Patch semantics: find must be present exactly once.
			if nvs == nil {
				return fmt.Errorf("note not found for patch: %s", change.Path)
			}
			nv := nvs.PathMap[change.Path]
			if nv == nil {
				return fmt.Errorf("note not found for patch: %s", change.Path)
			}
			current := string(nv.Content)
			idx := strings.Index(current, change.Find)
			if idx == -1 {
				return fmt.Errorf("patch find string not found in %s", change.Path)
			}
			// Reject ambiguous finds (multiple occurrences) to match updateNotes Patch semantics.
			if strings.Contains(current[idx+len(change.Find):], change.Find) {
				return fmt.Errorf("patch find string is ambiguous (multiple occurrences) in %s", change.Path)
			}
			content = current[:idx] + change.Replace + current[idx+len(change.Find):]
		} else {
			// Kind=="" or "upsert": upsert with provided content.
			content = change.Content
		}

		_, insertErr := env.InsertNote(ctx, model.RawNote{
			Path:    change.Path,
			Content: content,
		})
		if insertErr != nil {
			return fmt.Errorf("failed to apply change for %s: %w", change.Path, insertErr)
		}
	}

	return nil
}
