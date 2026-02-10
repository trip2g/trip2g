package delivercronwebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/ptr"
	"trip2g/internal/shortapitoken"
	"trip2g/internal/webhookutil"

	"github.com/valyala/fasthttp"
)

// ResponseSchema is a server constant included in every cron webhook payload.
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
	UpdateCronWebhookDeliveryResult(ctx context.Context, arg db.UpdateCronWebhookDeliveryResultParams) error
	InsertWebhookDeliveryLog(ctx context.Context, arg db.InsertWebhookDeliveryLogParams) error
	InsertNote(ctx context.Context, note model.RawNote) (int64, error)
	EnqueueDeliverCronWebhook(ctx context.Context, params DeliverCronParams) error
	ShortAPITokenSecret() string
	WebhookHTTPClient() *fasthttp.Client
	Logger() logger.Logger
}

// cronWebhookPayload is the JSON body sent to the cron webhook endpoint.
type cronWebhookPayload struct {
	webhookutil.BasePayload
	Instruction    string          `json:"instruction"`
	ResponseSchema json.RawMessage `json:"response_schema"`
	APIToken       string          `json:"api_token,omitempty"`
	PreviousError  string          `json:"previous_error,omitempty"`
}

func Resolve(ctx context.Context, env Env, params DeliverCronParams) error {
	log := env.Logger()

	// Load cron webhook configuration.
	wh, err := env.CronWebhookByID(ctx, params.CronWebhookID)
	if err != nil {
		return fmt.Errorf("failed to load cron webhook %d: %w", params.CronWebhookID, err)
	}

	// Build payload.
	payload := cronWebhookPayload{
		BasePayload:    webhookutil.NewBasePayload(params.DeliveryID, params.Attempt),
		Instruction:    wh.Instruction,
		ResponseSchema: ResponseSchema,
		PreviousError:  params.PreviousError,
	}

	// Generate short API token if pass_api_key is enabled.
	if wh.PassApiKey {
		readPatterns, rpErr := parseJSONStringArray(wh.ReadPatterns)
		if rpErr != nil {
			log.Error("failed to parse read_patterns", "cron_webhook_id", wh.ID, "error", rpErr)
			readPatterns = []string{"*"}
		}

		writePatterns, wpErr := parseJSONStringArray(wh.WritePatterns)
		if wpErr != nil {
			log.Error("failed to parse write_patterns", "cron_webhook_id", wh.ID, "error", wpErr)
			writePatterns = []string{}
		}

		ttl := time.Duration(wh.TimeoutSeconds) * time.Second
		if ttl < 60*time.Minute {
			ttl = 60 * time.Minute
		}

		token, signErr := shortapitoken.Sign(shortapitoken.Data{
			Depth:         1, // Cron webhooks always start at depth 1.
			ReadPatterns:  readPatterns,
			WritePatterns: writePatterns,
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
	if result.Err != nil || result.StatusCode >= 500 {
		errMsg := "server error"
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
				log.Error("failed to enqueue cron webhook retry", "delivery_id", params.DeliveryID, "error", retryErr)
			}
			return nil
		}

		// Mark as failed.
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

	// Parse agent response for changes.
	var applyErr error
	if result.StatusCode >= 200 && result.StatusCode < 300 && result.StatusCode != 202 {
		agentResp, parseErr := webhookutil.ParseAgentResponse(result.Body)
		if parseErr != nil {
			applyErr = parseErr
		} else if agentResp != nil && len(agentResp.Changes) > 0 {
			for _, change := range agentResp.Changes {
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
	}

	// Mark as success.
	updateErr := env.UpdateCronWebhookDeliveryResult(ctx, db.UpdateCronWebhookDeliveryResultParams{
		Status:         "success",
		ResponseStatus: ptr.To(int64(result.StatusCode)),
		DurationMs:     ptr.To(result.DurationMs),
		ID:             params.DeliveryID,
	})
	if updateErr != nil {
		log.Error("failed to update cron delivery result", "delivery_id", params.DeliveryID, "error", updateErr)
	}

	return nil
}

// parseJSONStringArray parses a JSON string array like '["blog/**","docs/*"]'.
func parseJSONStringArray(raw string) ([]string, error) {
	var result []string

	err := json.Unmarshal([]byte(raw), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON string array: %w", err)
	}

	return result, nil
}
