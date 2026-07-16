package tasks

import (
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// MailDispatcher implements the emailSender interface by enqueuing
// email tasks to an asynq queue instead of sending directly.
type MailDispatcher struct {
	client  *asynq.Client
	enabled bool
}

// NewMailDispatcher creates a dispatcher that enqueues email tasks.
// Pass nil client to create a disabled dispatcher.
func NewMailDispatcher(client *asynq.Client, enabled bool) *MailDispatcher {
	return &MailDispatcher{client: client, enabled: enabled && client != nil}
}

func (d *MailDispatcher) Enabled() bool {
	return d.enabled
}

func (d *MailDispatcher) SendVerificationCode(to, code string) error {
	if !d.enabled {
		return fmt.Errorf("mail dispatcher disabled")
	}
	payload, err := NewEmailVerificationTask(to, code)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	task := asynq.NewTask(TypeEmailVerification, payload,
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
	)
	_, err = d.client.Enqueue(task)
	if err != nil {
		return fmt.Errorf("enqueue verification email: %w", err)
	}
	return nil
}

func (d *MailDispatcher) SendPasswordResetCode(to, code string) error {
	if !d.enabled {
		return fmt.Errorf("mail dispatcher disabled")
	}
	payload, err := NewPasswordResetTask(to, code)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	task := asynq.NewTask(TypePasswordReset, payload,
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
	)
	_, err = d.client.Enqueue(task)
	if err != nil {
		return fmt.Errorf("enqueue password reset email: %w", err)
	}
	return nil
}
