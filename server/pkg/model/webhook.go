package model

type ConvWebhook struct {
	ID            int64           `json:"id"`
	ConvID        string          `json:"conv_id"`
	Name          string          `json:"name"`
	Token         string          `json:"token,omitempty"`
	APIKeyPlain   string          `json:"api_key,omitempty"`
	APIKeyHash    string          `json:"-"`
	CallbackURL   string          `json:"callback_url,omitempty"`
	Headers       []WebhookHeader `json:"headers,omitempty"`
	CIDRWhitelist []string        `json:"cidr_whitelist,omitempty"`
	RequireAudit  bool            `json:"require_audit"`
	CreatedBy     string          `json:"created_by"`
	CreatedAt     int64           `json:"created_at"`
}

// APIKeyPlaintext is used only when returning the key to the creator at creation time.
// It is NOT stored in the database — only the bcrypt hash (APIKeyHash) is persisted.
type ConvWebhookWithKey struct {
	*ConvWebhook
	APIKey string `json:"api_key"`
	Token  string `json:"token"`
}

type WebhookHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type WebhookAuditLog struct {
	ID        int64  `json:"id"`
	WebhookID int64  `json:"webhook_id"`
	ConvID    string `json:"conv_id"`
	MsgID     int64  `json:"msg_id"`
	Action    string `json:"action"`
	ActorID   string `json:"actor_id"`
	Reason    string `json:"reason,omitempty"`
	CallerIP  string `json:"caller_ip,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

type WebhookMessage struct {
	MsgID       int64  `json:"msg_id"`
	WebhookID   int64  `json:"webhook_id"`
	ConvID      string `json:"conv_id"`
	AuditStatus string `json:"audit_status"`
	SourceIP    string `json:"source_ip"`
	CreatedAt   int64  `json:"created_at"`
}
