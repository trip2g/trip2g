package deliverchangewebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"trip2g/internal/case/handlenotewebhooks"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/ptr"
	"trip2g/internal/shortapitoken"
	"trip2g/internal/webhookutil"

	"github.com/valyala/fasthttp"
)

type Env interface {
	WebhookByID(ctx context.Context, id int64) (db.ChangeWebhook, error)
	UpdateWebhookDeliveryResult(ctx context.Context, arg db.UpdateWebhookDeliveryResultParams) error
	InsertWebhookDeliveryLog(ctx context.Context, arg db.InsertWebhookDeliveryLogParams) error
	InsertNote(ctx context.Context, note model.RawNote) (int64, error)
	EnqueueDeliverChangeWebhook(ctx context.Context, params handlenotewebhooks.DeliverChangeWebhookParams) error
	ShortAPITokenSecret() string
	WebhookHTTPClient() *fasthttp.Client
	Logger() logger.Logger
}

// changeWebhookPayload is the JSON body sent to the webhook endpoint.
type changeWebhookPayload struct {
	webhookutil.BasePayload
	Depth         int                             `json:"depth"`
	Instruction   string                          `json:"instruction"`
	Changes       []handlenotewebhooks.ChangeInfo `json:"changes"`
	APIToken      string                          `json:"api_token,omitempty"`
	PreviousError string                          `json:"previous_error,omitempty"`
}

func Resolve(ctx context.Context, env Env, params handlenotewebhooks.DeliverChangeWebhookParams) error {
	log := env.Logger()

	// Load webhook configuration.
	wh, err := env.WebhookByID(ctx, params.WebhookID)
	if err != nil {
		return fmt.Errorf("failed to load webhook %d: %w", params.WebhookID, err)
	}

	// Build payload.
	payload := changeWebhookPayload{
		BasePayload:   webhookutil.NewBasePayload(params.DeliveryID, params.Attempt),
		Depth:         params.Depth,
		Instruction:   wh.Instruction,
		Changes:       params.Changes,
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
			readPatterns = []string{"*"}
		}

		ttl := time.Duration(wh.TimeoutSeconds) * time.Second
		if ttl < 60*time.Minute {
			ttl = 60 * time.Minute
		}

		token, signErr := shortapitoken.Sign(shortapitoken.Data{
			Depth:         params.Depth + 1,
			ReadPatterns:  readPatterns,
			WritePatterns: writePatterns,
		}, env.ShortAPITokenSecret(), ttl)
		if signErr != nil {
			log.Error("failed to sign short API token", "webhook_id", wh.ID, "error", signErr)
		} else {
			payload.APIToken = token
		}
	}

	// Marshal payload to JSON.
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
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
	requestBodyStr := string(payloadBytes)
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
	if result.Err != nil || result.StatusCode >= 300 {
		var errMsg string
		if result.Err != nil {
			errMsg = result.Err.Error()
		} else {
			errMsg = fmt.Sprintf("HTTP %d", result.StatusCode)
		}

		if int64(params.Attempt) < wh.MaxRetries {
			// Enqueue retry.
			retryErr := env.EnqueueDeliverChangeWebhook(ctx, handlenotewebhooks.DeliverChangeWebhookParams{
				DeliveryID:    params.DeliveryID,
				WebhookID:     params.WebhookID,
				Attempt:       params.Attempt + 1,
				Depth:         params.Depth,
				Changes:       params.Changes,
				PreviousError: errMsg,
			})
			if retryErr != nil {
				log.Error("failed to enqueue webhook retry", "delivery_id", params.DeliveryID, "error", retryErr)
			}
			return nil
		}

		// Mark as failed.
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

	// Parse agent response for changes.
	var applyErr error
	if result.StatusCode >= 200 && result.StatusCode < 300 && result.StatusCode != 202 {
		agentResp, parseErr := webhookutil.ParseAgentResponse(result.Body)
		if parseErr != nil {
			applyErr = parseErr
		} else if agentResp != nil && len(agentResp.Changes) > 0 {
			// Apply agent changes via InsertNote.
			for _, change := range agentResp.Changes {
				// Validate path against write patterns.
				if len(writePatterns) > 0 && !webhookutil.MatchesAny(change.Path, writePatterns) {
					applyErr = fmt.Errorf("path %q not allowed by write_patterns", change.Path)
					break
				}

				_, insertErr := env.InsertNote(ctx, model.RawNote{
					Path:    change.Path,
					Content: change.Content,
				})
				if insertErr != nil {
					applyErr = fmt.Errorf("failed to apply change for %s: %w", change.Path, insertErr)
					break
				}
			}
		}
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

	// Mark as success.
	updateErr := env.UpdateWebhookDeliveryResult(ctx, db.UpdateWebhookDeliveryResultParams{
		Status:         "success",
		ResponseStatus: ptr.To(int64(result.StatusCode)),
		DurationMs:     ptr.To(result.DurationMs),
		ID:             params.DeliveryID,
	})
	if updateErr != nil {
		log.Error("failed to update delivery result", "delivery_id", params.DeliveryID, "error", updateErr)
	}

	return nil
}
