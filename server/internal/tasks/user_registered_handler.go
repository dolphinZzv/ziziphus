package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"ziziphus/internal/auth"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
)

// UserRegisteredHandler processes the user_registered event.
// It handles all post-registration side effects such as auto-joining groups
// and sending welcome emails.
type UserRegisteredHandler struct {
	convMgr  userRegConvMgr
	mailer   *auth.Mailer
	autoJoin string // group ID to auto-join (empty = disabled)
}

type userRegConvMgr interface {
	Get(ctx context.Context, convID string) (*model.Conversation, error)
	AddMember(ctx context.Context, convID, userID, operatorID string) error
}

func NewUserRegisteredHandler(convMgr userRegConvMgr, mailer *auth.Mailer, autoJoinGroup string) *UserRegisteredHandler {
	return &UserRegisteredHandler{convMgr: convMgr, mailer: mailer, autoJoin: autoJoinGroup}
}

func (h *UserRegisteredHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeUserRegistered, h.ProcessTask)
}

func (h *UserRegisteredHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload UserRegisteredPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal user registered task: %w", err)
	}

	logger.Debug("processing user registered event",
		"user_id", payload.UserID, "lang", payload.Lang)

	// Auto-join group (if configured)
	if h.autoJoin != "" {
		if err := h.autoJoinGroup(ctx, payload.UserID); err != nil {
			return err
		}
	}

	// Welcome email
	if payload.Email != "" && h.mailer != nil {
		if err := h.mailer.SendWelcomeEmailLang(payload.Email, payload.Lang); err != nil {
			logger.Warn("user registered: welcome email failed",
				"user", payload.UserID, "email", payload.Email, "error", err)
			return fmt.Errorf("send welcome email: %w", err)
		}
		logger.Info("user registered: welcome email sent",
			"user", payload.UserID, "email", payload.Email)
	}

	return nil
}

func (h *UserRegisteredHandler) autoJoinGroup(ctx context.Context, userID string) error {
	conv, err := h.convMgr.Get(ctx, h.autoJoin)
	if err != nil {
		logger.Warn("user registered: auto-join group not found",
			"group", h.autoJoin, "user", userID, "error", err)
		return fmt.Errorf("get group %s: %w", h.autoJoin, err)
	}

	if err := h.convMgr.AddMember(ctx, h.autoJoin, userID, conv.OwnerID); err != nil {
		logger.Warn("user registered: auto-join group failed",
			"group", h.autoJoin, "user", userID, "error", err)
		return fmt.Errorf("add member: %w", err)
	}

	logger.Info("user registered: auto-joined group",
		"group", h.autoJoin, "user", userID)
	return nil
}
