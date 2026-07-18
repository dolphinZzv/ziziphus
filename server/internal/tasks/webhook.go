package tasks

import (
	"encoding/json"

	"ziziphus/pkg/model"
)

const TypeWebhookForward = "webhook:forward"

type WebhookForwardPayload struct {
	CallbackURL string                `json:"callback_url"`
	APIKeyHash  string                `json:"api_key_hash"`
	Headers     []model.WebhookHeader `json:"headers"`
	AppName     string                `json:"app_name"`
	WhID        int64                 `json:"wh_id"`
	ConvID      string                `json:"conv_id"`
	MsgID       int64                 `json:"msg_id"`
	SenderID    string                `json:"sender_id"`
	SenderName  string                `json:"sender_name"`
	ContentType int                   `json:"content_type"`
	Body        string                `json:"body"`
	ReplyTo     int64                 `json:"reply_to"`
	Timestamp   int64                 `json:"timestamp"`
}

func NewWebhookForwardTask(wh *model.ConvWebhook, appName string, msg *model.Message) (*WebhookForwardPayload, error) {
	return &WebhookForwardPayload{
		CallbackURL: wh.CallbackURL,
		APIKeyHash:  wh.APIKeyHash,
		Headers:     wh.Headers,
		AppName:     appName,
		WhID:        wh.ID,
		ConvID:      msg.ConvID,
		MsgID:       msg.MsgID,
		SenderID:    msg.SenderID,
		SenderName:  msg.SenderName,
		ContentType: int(msg.ContentType),
		Body:        msg.Body,
		ReplyTo:     msg.ReplyTo,
		Timestamp:   msg.Timestamp,
	}, nil
}

func (p *WebhookForwardPayload) Marshal() ([]byte, error) {
	return json.Marshal(p)
}
