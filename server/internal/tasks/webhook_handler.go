package tasks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hibiken/asynq"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
)

// WebhookHandler processes webhook forwarding tasks from the asynq queue.
type WebhookHandler struct{}

func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{}
}

func (h *WebhookHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeWebhookForward, h.ProcessTask)
}

func (h *WebhookHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload WebhookForwardPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal webhook task: %w", err)
	}

	// Build webhook payload
	body := buildWebhookBody(&payload)
	raw, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", payload.CallbackURL, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", payload.AppName+"-Webhook/1.0")
	req.Header.Set("X-Signature", computeSignature([]byte(payload.APIKeyHash), raw))
	// Propagate the OTel trace context to downstream webhook receivers
	// via the W3C traceparent header so they can continue the distributed trace.
	if payload.TraceID != "" {
		req.Header.Set("traceparent", "00-"+payload.TraceID+"-0000000000000001-01")
	}
	for _, hdr := range payload.Headers {
		req.Header.Set(hdr.Key, hdr.Value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Warn("webhook forward failed", "wh_id", payload.WhID, "msg_id", payload.MsgID, "error", err)
		return fmt.Errorf("webhook POST failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		err = fmt.Errorf("webhook returned %d", resp.StatusCode)
		logger.Warn("webhook forward failed", "wh_id", payload.WhID, "msg_id", payload.MsgID, "status", resp.StatusCode)
		return err
	}

	logger.Info("webhook forwarded", "wh_id", payload.WhID, "msg_id", payload.MsgID, "status", resp.StatusCode)
	return nil
}

func buildWebhookBody(p *WebhookForwardPayload) map[string]any {
	body := map[string]any{
		"event":      "message.created",
		"webhook_id": p.WhID,
		"conv_id":    p.ConvID,
		"message": map[string]any{
			"msg_id":       p.MsgID,
			"sender_id":    p.SenderID,
			"sender_name":  p.SenderName,
			"content_type": p.ContentType,
			"body":         p.Body,
			"reply_to":     p.ReplyTo,
			"timestamp":    p.Timestamp,
		},
	}
	if p.TraceID != "" {
		body["trace_id"] = p.TraceID
	}
	return body
}

func computeSignature(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Ensure model.WebhookHeader is accessible
var _ = model.WebhookHeader{}
