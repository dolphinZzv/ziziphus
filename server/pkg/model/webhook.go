package model

type ConvWebhook struct {
	ID            int64           `json:"id"`
	ConvID        string          `json:"conv_id"`
	Name          string          `json:"name"`
	APIKeyPlain   string          `json:"api_key,omitempty"`
	APIKeyHash    string          `json:"-"`
	CallbackURL   string          `json:"callback_url,omitempty"`
	Headers       []WebhookHeader `json:"headers,omitempty"`
	CIDRWhitelist []string        `json:"cidr_whitelist,omitempty"`
	CreatedBy     string          `json:"created_by"`
	CreatedAt     int64           `json:"created_at"`
}

type WebhookHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
