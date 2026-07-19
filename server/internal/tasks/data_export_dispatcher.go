package tasks

import (
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// DataExportDispatcher enqueues data-export tasks for async processing.
type DataExportDispatcher struct {
	client *asynq.Client
}

func NewDataExportDispatcher(client *asynq.Client) *DataExportDispatcher {
	return &DataExportDispatcher{client: client}
}

func (d *DataExportDispatcher) Enqueue(userID, lang, email string) error {
	payload, err := NewDataExportTask(userID, lang, email)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	task := asynq.NewTask(TypeDataExport, payload,
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
	)
	if _, err := d.client.Enqueue(task); err != nil {
		return fmt.Errorf("enqueue data export: %w", err)
	}
	return nil
}
