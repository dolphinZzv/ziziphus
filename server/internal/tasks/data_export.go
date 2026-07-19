package tasks

import "encoding/json"

const TypeDataExport = "data_export"

type DataExportPayload struct {
	UserID   string `json:"user_id"`
	Lang     string `json:"lang"`
	Email    string `json:"email"`
}

// NewDataExportTask creates a serialisable data export task payload.
func NewDataExportTask(userID, lang, email string) ([]byte, error) {
	return json.Marshal(DataExportPayload{UserID: userID, Lang: lang, Email: email})
}
