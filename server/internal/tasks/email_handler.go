package tasks

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel/trace"
	"ziziphus/internal/auth"
	"ziziphus/pkg/logger"
)

// MailHandler processes email tasks from the asynq queue
// by calling the real SMTP mailer.
type MailHandler struct {
	mailer *auth.Mailer
}

func NewMailHandler(mailer *auth.Mailer) *MailHandler {
	return &MailHandler{mailer: mailer}
}

func (h *MailHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()

	switch task.Type() {
	case TypeEmailVerification:
		logger.Debug("processing email verification", "trace_id", traceID)
		return h.handleVerification(task)
	case TypePasswordReset:
		logger.Debug("processing password reset email", "trace_id", traceID)
		return h.handlePasswordReset(task)
	default:
		logger.Warn("unknown task type", "type", task.Type())
		return nil
	}
}

// RegisterHandlers registers email task handlers on the asynq mux.
func (h *MailHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEmailVerification, h.ProcessTask)
	mux.HandleFunc(TypePasswordReset, h.ProcessTask)
}

func (h *MailHandler) handleVerification(task *asynq.Task) error {
	var payload EmailVerificationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}
	logger.Info("sending verification email",
		"to", payload.To,
	)
	return h.mailer.SendVerificationCode(payload.To, payload.Code)
}

func (h *MailHandler) handlePasswordReset(task *asynq.Task) error {
	var payload PasswordResetPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}
	logger.Info("sending password reset email",
		"to", payload.To,
	)
	return h.mailer.SendPasswordResetCode(payload.To, payload.Code)
}
