package deliverchangewebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/db"
	"trip2g/internal/jsonneteval"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/ptr"
	"trip2g/internal/shortapitoken"
	"trip2g/internal/webhookutil"

	"github.com/valyala/fasthttp"
)

type Env interface {
	WebhookByID(ctx context.Context, id int64) (db.ChangeWebhook, error)
	MarkWebhookDeliveryRunning(ctx context.Context, id int64) error
	UpdateWebhookDeliveryResult(ctx context.Context, arg db.UpdateWebhookDeliveryResultParams) error
	InsertWebhookDeliveryLog(ctx context.Context, arg db.InsertWebhookDeliveryLogParams) error
	InsertNote(ctx context.Context, note model.RawNote) (int64, error)
	PrepareLatestNotes(ctx context.Context, partial bool) (*model.NoteViews, error)
	LatestNoteViews() *model.NoteViews
	EnqueueDeliverChangeWebhook(ctx context.Context, params handlenotewebhooks.DeliverChangeWebhookParams) error
	ShortAPITokenSecret() string
	WebhookHTTPClient() *fasthttp.Client
	Logger() logger.Logger
	GetSecretValues(ctx context.Context, like string) (map[string]string, error)
}

// changeWebhookPayload is the JSON body sent to the webhook endpoint.
type changeWebhookPayload struct {
	webhookutil.BasePayload
	Depth         int                             `json:"depth"`
	Instruction   string                          `json:"instruction"`
	Changes       []handlenotewebhooks.ChangeInfo `json:"changes"`
	AttachedNotes []webhookutil.AttachedNote      `json:"attached_notes,omitempty"`
	APIToken      string                          `json:"api_token,omitempty"`
	Secrets       map[string]string               `json:"secrets,omitempty"`
	PreviousError string                          `json:"previous_error,omitempty"`
}

// tokenTTLMargin is the small grace window added to the delivery timeout for
// the scoped write-back token. Replaces the former 60-minute floor.
const tokenTTLMargin = 30 * time.Second

//nolint:gocognit,gocyclo,cyclop,funlen // delivery resolver handles full webhook lifecycle: auth, attach-gate, fan-out, write-back
func Resolve(ctx context.Context, env Env, params handlenotewebhooks.DeliverChangeWebhookParams) error {
	log := env.Logger()

	// Load webhook configuration.
	wh, err := env.WebhookByID(ctx, params.WebhookID)
	if err != nil {
		return fmt.Errorf("failed to load webhook %d: %w", params.WebhookID, err)
	}

	if params.Attempt <= 1 {
		if mErr := env.MarkWebhookDeliveryRunning(ctx, params.DeliveryID); mErr != nil {
			log.Error("failed to mark delivery running", "delivery_id", params.DeliveryID, "error", mErr)
		}
	}

	p, _ := model.ChangeWebhookSecretPrefix(wh.ID)
	secrets := loadSecrets(ctx, env, log, p.String())

	// Build payload.
	payload := changeWebhookPayload{
		BasePayload:   webhookutil.NewBasePayload(params.DeliveryID, params.Attempt),
		Depth:         params.Depth,
		Instruction:   wh.Instruction,
		Changes:       params.Changes,
		Secrets:       secrets,
		PreviousError: params.PreviousError,
	}

	// Parse write patterns for validating agent response changes.
	writePatterns, wpErr := webhookutil.ParseJSONStringArray(wh.WritePatterns)
	if wpErr != nil {
		log.Error("failed to parse write_patterns", "webhook_id", wh.ID, "error", wpErr)
		writePatterns = []string{}
	}

	// Generate short API token if pass_api_key is enabled.
	if wh.PassApiKey {
		readPatterns, rpErr := webhookutil.ParseJSONStringArray(wh.ReadPatterns)
		if rpErr != nil {
			log.Error("failed to parse read_patterns", "webhook_id", wh.ID, "error", rpErr)
			readPatterns = []string{"**"}
		}

		ttl := time.Duration(wh.TimeoutSeconds)*time.Second + tokenTTLMargin

		token, signErr := shortapitoken.Sign(shortapitoken.Data{
			Depth:         params.Depth + 1,
			ReadPatterns:  readPatterns,
			WritePatterns: writePatterns,
			DeliveryKind:  "change",
			DeliveryID:    params.DeliveryID,
		}, env.ShortAPITokenSecret(), ttl)
		if signErr != nil {
			log.Error("failed to sign short API token", "webhook_id", wh.ID, "error", signErr)
		} else {
			payload.APIToken = token
		}
	}

	// Materialize attach_notes context notes into the payload.
	if wh.AttachNotes != "" && wh.AttachNotes != "[]" {
		if attach, aerr := webhookutil.ParseJSONStringArray(wh.AttachNotes); aerr == nil {
			payload.AttachedNotes = webhookutil.MaterializeAttachedNotes(attach, env.LatestNoteViews())
		} else {
			log.Error("failed to parse attach_notes", "webhook_id", wh.ID, "error", aerr)
		}
	}

	// Marshal payload to JSON.
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Apply outbound transform (if configured) strictly between marshal and sign,
	// so both the HMAC and the logged request_body cover the transformed result.
	transformed := wh.TransformJsonnet != ""
	if transformed {
		out, terr := jsonneteval.EvalJSON(wh.TransformJsonnet, transformExtVars(payloadBytes))
		if terr != nil {
			// Never send a half-built request.
			handleDeliveryError(ctx, env, params,
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
		Kind:        "change",
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
		log.Error("failed to insert webhook delivery log", "delivery_id", params.DeliveryID, "error", logErr)
	}

	// Handle HTTP error or server error.
	//nolint:nilerr // error handled via handleDeliveryError, not returned.
	if result.Err != nil || result.StatusCode >= 300 {
		handleDeliveryError(ctx, env, params, result, wh)
		return nil
	}

	// Parse agent response for changes.
	var applyErr error
	if result.StatusCode >= 200 && result.StatusCode < 300 && result.StatusCode != http.StatusAccepted {
		applyErr = applyAgentChanges(ctx, env, result, writePatterns)
	}

	// Handle agent response errors with retry.
	if applyErr != nil {
		if int64(params.Attempt) < wh.MaxRetries {
			retryErr := env.EnqueueDeliverChangeWebhook(ctx, handlenotewebhooks.DeliverChangeWebhookParams{
				DeliveryID:    params.DeliveryID,
				WebhookID:     params.WebhookID,
				Attempt:       params.Attempt + 1,
				Depth:         params.Depth,
				Changes:       params.Changes,
				PreviousError: applyErr.Error(),
			})
			if retryErr != nil {
				log.Error("failed to enqueue webhook retry", "delivery_id", params.DeliveryID, "error", retryErr)
			}
			return nil
		}

		log.Warn("agent response error, no retries left",
			"delivery_id", params.DeliveryID,
			"error", applyErr,
		)

		// Mark as failed when agent response error with no retries left.
		updateErr := env.UpdateWebhookDeliveryResult(ctx, db.UpdateWebhookDeliveryResultParams{
			Status:         "failed",
			ResponseStatus: ptr.To(int64(result.StatusCode)),
			DurationMs:     ptr.To(result.DurationMs),
			ID:             params.DeliveryID,
		})
		if updateErr != nil {
			log.Error("failed to update delivery result", "delivery_id", params.DeliveryID, "error", updateErr)
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
	updateErr := env.UpdateWebhookDeliveryResult(ctx, db.UpdateWebhookDeliveryResultParams{
		Status:         "success",
		ResponseStatus: ptr.To(int64(result.StatusCode)),
		DurationMs:     ptr.To(result.DurationMs),
		TokensUsed:     tokensUsed,
		Steps:          steps,
		ID:             params.DeliveryID,
	})
	if updateErr != nil {
		log.Error("failed to update delivery result", "delivery_id", params.DeliveryID, "error", updateErr)
	}

	return nil
}

// loadSecrets fetches decrypted secrets for the given prefix, returning a map keyed by bare name.
func loadSecrets(ctx context.Context, env Env, log logger.Logger, prefix string) map[string]string {
	all, err := env.GetSecretValues(ctx, prefix+"%")
	if err != nil {
		log.Error("failed to load webhook secrets", "prefix", prefix, "error", err)
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

// handleDeliveryError handles HTTP errors and retries.
func handleDeliveryError(
	ctx context.Context,
	env Env,
	params handlenotewebhooks.DeliverChangeWebhookParams,
	result webhookutil.DeliveryResult,
	wh db.ChangeWebhook,
) {
	var errMsg string
	if result.Err != nil {
		errMsg = result.Err.Error()
	} else {
		errMsg = fmt.Sprintf("HTTP %d", result.StatusCode)
	}

	if int64(params.Attempt) < wh.MaxRetries {
		retryErr := env.EnqueueDeliverChangeWebhook(ctx, handlenotewebhooks.DeliverChangeWebhookParams{
			DeliveryID:    params.DeliveryID,
			WebhookID:     params.WebhookID,
			Attempt:       params.Attempt + 1,
			Depth:         params.Depth,
			Changes:       params.Changes,
			PreviousError: errMsg,
		})
		if retryErr != nil {
			env.Logger().Error("failed to enqueue webhook retry", "delivery_id", params.DeliveryID, "error", retryErr)
		}
		return
	}

	// Mark as failed.
	updateErr := env.UpdateWebhookDeliveryResult(ctx, db.UpdateWebhookDeliveryResultParams{
		Status:         "failed",
		ResponseStatus: ptr.To(int64(result.StatusCode)),
		DurationMs:     ptr.To(result.DurationMs),
		ID:             params.DeliveryID,
	})
	if updateErr != nil {
		env.Logger().Error("failed to update delivery result", "delivery_id", params.DeliveryID, "error", updateErr)
	}
}

// transformExtVars exposes the change/attached-notes/meta of the outbound
// payload to the jsonnet transform. The api_token and secrets are intentionally
// NEVER exposed (they must not reach the transform or the logged request_body).
func transformExtVars(payloadBytes []byte) map[string]string {
	var p map[string]json.RawMessage
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return map[string]string{}
	}

	ev := make(map[string]string, 3)
	if v, ok := p["changes"]; ok {
		ev["change"] = string(v)
	}
	if v, ok := p["attached_notes"]; ok {
		ev["attached_notes"] = string(v)
	}
	if v, ok := p["meta"]; ok {
		ev["meta"] = string(v)
	}
	return ev
}

// TransformExtVarsForTest exposes transformExtVars to the external test package.
func TransformExtVarsForTest(payloadBytes []byte) map[string]string {
	return transformExtVars(payloadBytes)
}

// applyAgentChanges parses, validates, and applies agent response changes.
// The whole batch is validated (write patterns, expected_hash CAS, patch
// resolution) before any item is written, so an invalid item rejects the
// batch without partial application.
func applyAgentChanges(ctx context.Context, env Env, result webhookutil.DeliveryResult, writePatterns []string) error {
	agentResp, parseErr := webhookutil.ParseAgentResponse(result.Body)
	if parseErr != nil {
		return parseErr
	}
	if agentResp == nil || len(agentResp.Changes) == 0 {
		return nil
	}

	notes, resolveErr := webhookutil.ResolveAgentChanges(agentResp.Changes, env.LatestNoteViews(), writePatterns)
	if resolveErr != nil {
		return resolveErr
	}

	for _, note := range notes {
		if _, insertErr := env.InsertNote(ctx, note); insertErr != nil {
			return fmt.Errorf("failed to apply change for %s: %w", note.Path, insertErr)
		}
	}

	// Refresh LatestNoteViews cache after applying changes.
	_, prepareErr := env.PrepareLatestNotes(ctx, false)
	if prepareErr != nil {
		return fmt.Errorf("failed to refresh note views: %w", prepareErr)
	}

	return nil
}
