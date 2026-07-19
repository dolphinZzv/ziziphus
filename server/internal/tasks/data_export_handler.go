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

// DataExportHandler processes data-export tasks by collecting user data
// and sending it to the user's email asynchronously.
type DataExportHandler struct {
	userRepo    userExportRepo
	msgRepo     msgExportRepo
	sessMgr     sessionExportRepo
	mailer      *auth.Mailer
}

type userExportRepo interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
}

type msgExportRepo interface {
	GetMessagesBySender(ctx context.Context, senderID string, limit, offset int) ([]*model.Message, error)
}

type sessionExportRepo interface {
	GetUserSessionIDs(ctx context.Context, userID string) []string
}

func NewDataExportHandler(userRepo userExportRepo, msgRepo msgExportRepo, sessMgr sessionExportRepo, mailer *auth.Mailer) *DataExportHandler {
	return &DataExportHandler{
		userRepo: userRepo,
		msgRepo:  msgRepo,
		sessMgr:  sessMgr,
		mailer:   mailer,
	}
}

func (h *DataExportHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeDataExport, h.ProcessTask)
}

func (h *DataExportHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload DataExportPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal data export task: %w", err)
	}

	logger.Info("processing data export",
		"user_id", payload.UserID,
		"lang", payload.Lang,
		"email", payload.Email,
	)

	// Collect user profile
	user, err := h.userRepo.GetByID(ctx, payload.UserID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	user.Password = ""

	// Collect all messages by this user
	var allMessages []*model.Message
	offset := 0
	limit := 200
	for {
		msgs, err := h.msgRepo.GetMessagesBySender(ctx, payload.UserID, limit, offset)
		if err != nil {
			logger.Error("export messages failed", "user_id", payload.UserID, "error", err)
			break
		}
		if len(msgs) == 0 {
			break
		}
		allMessages = append(allMessages, msgs...)
		offset += limit
	}

	// Get session IDs
	sessionIDs := h.sessMgr.GetUserSessionIDs(ctx, payload.UserID)

	// Build export data
	exportData := map[string]any{
		"user":        user,
		"messages":    allMessages,
		"session_ids": sessionIDs,
	}
	raw, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal export data: %w", err)
	}

	// Send email
	if err := h.mailer.SendDataExportLang(payload.Email, string(raw), payload.Lang); err != nil {
		return fmt.Errorf("send data export email: %w", err)
	}

	logger.Info("data export sent",
		"user_id", payload.UserID,
		"email", payload.Email,
		"messages", len(allMessages),
	)
	return nil
}
