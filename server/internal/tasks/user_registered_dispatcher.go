package tasks

import (
	"fmt"

	"github.com/hibiken/asynq"
)

// UserRegisteredDispatcher enqueues a user_registered event when a new user signs up.
// Downstream handlers (auto-join group, welcome email, etc.) process the event.
type UserRegisteredDispatcher struct {
	client *asynq.Client
}

func NewUserRegisteredDispatcher(client *asynq.Client) *UserRegisteredDispatcher {
	return &UserRegisteredDispatcher{client: client}
}

func (d *UserRegisteredDispatcher) Enqueue(userID, lang, email string) error {
	payload, err := NewUserRegisteredTask(userID, lang, email)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	task := asynq.NewTask(TypeUserRegistered, payload, asynq.MaxRetry(3))
	if _, err := d.client.Enqueue(task); err != nil {
		return fmt.Errorf("enqueue user registered: %w", err)
	}
	return nil
}
