package model

// FormType identifies the kind of form for client-side rendering.
type FormType string

const (
	FormTypeContactRequest FormType = "contact_request"
)

// FormFieldType enumerates supported input field types.
type FormFieldType string

const (
	FormFieldText     FormFieldType = "text"
	FormFieldTextarea FormFieldType = "textarea"
	FormFieldSelect   FormFieldType = "select"
	FormFieldRadio    FormFieldType = "radio"
	FormFieldCheckbox FormFieldType = "checkbox"
	FormFieldDate     FormFieldType = "date"
	FormFieldTime     FormFieldType = "time"
	FormFieldNumber   FormFieldType = "number"
	FormFieldRating   FormFieldType = "rating"
)

// FormSubmitMode controls whether a user may submit multiple responses.
type FormSubmitMode string

const (
	FormSubmitSingle   FormSubmitMode = "single"
	FormSubmitMultiple FormSubmitMode = "multiple"
)

// FormStatus is a client-side hint stored in the body for initial render.
// It is NOT authoritative — use the contact_requests table for authoritative state.
type FormStatus string

const (
	FormStatusActive FormStatus = "active"
	FormStatusClosed FormStatus = "closed"
)

// FormActionStyle hints at button rendering.
type FormActionStyle string

const (
	FormActionStylePrimary FormActionStyle = "primary"
	FormActionStyleDanger  FormActionStyle = "danger"
	FormActionStyleDefault FormActionStyle = "default"
)

// FormValidation holds optional validation constraints for a field.
type FormValidation struct {
	MinLength int    `json:"min_length"`
	MaxLength int    `json:"max_length"`
	Pattern   string `json:"pattern,omitempty"`
}

// FormField describes a single input field in a form definition.
type FormField struct {
	FieldID      string          `json:"field_id"`
	Type         FormFieldType   `json:"type"`
	Label        string          `json:"label"`
	Required     bool            `json:"required"`
	Options      []string        `json:"options,omitempty"`
	Placeholder  string          `json:"placeholder,omitempty"`
	DefaultValue interface{}     `json:"default_value,omitempty"`
	Validation   *FormValidation `json:"validation,omitempty"`
}

// FormAction describes a button presented in the form bubble.
type FormAction struct {
	Action string          `json:"action"`
	Label  string          `json:"label"`
	Style  FormActionStyle `json:"style"`
}

// FormDefinitionBody is the JSON-serialized body for ContentType=10 messages.
type FormDefinitionBody struct {
	FormID         string         `json:"form_id"`
	Type           FormType       `json:"type"`
	Title          string         `json:"title"`
	Description    string         `json:"description,omitempty"`
	FromUserID     string         `json:"from_user_id,omitempty"`
	FromUserName   string         `json:"from_user_name,omitempty"`
	FromUserAvatar string         `json:"from_user_avatar,omitempty"`
	RequestID      int64          `json:"request_id"`
	Message        string         `json:"message,omitempty"`
	Fields         []FormField    `json:"fields,omitempty"`
	Actions        []FormAction   `json:"actions"`
	SubmitMode     FormSubmitMode `json:"submit_mode,omitempty"`
	Deadline       *int64         `json:"deadline,omitempty"`
	Status         FormStatus     `json:"status"`
	CreatedAt      int64          `json:"created_at"`
}

// FormAnswer is a single field-value pair in a form response.
type FormAnswer struct {
	FieldID string      `json:"field_id"`
	Value   interface{} `json:"value"`
}

// FormResponseBody is the JSON-serialized body for ContentType=11 messages.
type FormResponseBody struct {
	FormMsgID     int64        `json:"form_msg_id"`
	RequestID     int64        `json:"request_id"`
	Action        string       `json:"action"`
	ResponderID   string       `json:"responder_id"`
	ResponderName string       `json:"responder_name"`
	Answers       []FormAnswer `json:"answers,omitempty"`
	SubmittedAt   int64        `json:"submitted_at"`
}
