package message

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
	"ziziphus/pkg/protocol"
)

type ReceiptHandler struct {
	msgRepo  receiptMsgRepo
	seqCache receiptSeqCache
	convRepo receiptConvRepo
	gateway  receiptConnRegistry
	receipt  receiptWriter
}

type receiptMsgRepo interface {
	Get(ctx context.Context, msgID int64) (*model.Message, error)
}

type receiptSeqCache interface {
	SetUserSeq(ctx context.Context, userID, convID string, seq int64) error
	GetUserSeq(ctx context.Context, userID, convID string) (int64, error)
	GetAndIncrementConvSeq(ctx context.Context, convID string) (int64, error)
	GetConvSeq(ctx context.Context, convID string) (int64, error)
}

type receiptConvRepo interface {
	Get(ctx context.Context, convID string) (*model.Conversation, error)
}

type receiptConnRegistry interface {
	GetByUserID(ctx context.Context, userID string) []any
}

func NewReceiptHandler(msgRepo receiptMsgRepo, seqCache receiptSeqCache, convRepo receiptConvRepo, gateway receiptConnRegistry, receipt receiptWriter) *ReceiptHandler {
	return &ReceiptHandler{
		msgRepo:  msgRepo,
		seqCache: seqCache,
		convRepo: convRepo,
		gateway:  gateway,
		receipt:  receipt,
	}
}

func (h *ReceiptHandler) MarkRead(ctx context.Context, userID, convID string, msgID int64) error {
	timestamp := time.Now().UnixMilli()

	msg, err := h.msgRepo.Get(ctx, msgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Message not found — try to clear unread by using the conversation's
			// current convSeq directly. This handles stale msg_ids from the frontend.
			if convSeq, getErr := h.seqCache.GetConvSeq(ctx, convID); getErr == nil && convSeq > 0 {
				if setErr := h.seqCache.SetUserSeq(ctx, userID, convID, convSeq); setErr != nil {
					logger.Error("mark read: set user seq after missing msg failed", "error", setErr)
				}
			}
			logger.Debug("mark read: msg not found, used convSeq", "msg_id", msgID, "conv_id", convID)
			return nil
		}
		return err
	}

	if msg.ConvID != convID {
		// convID mismatch — still try to clear unread
		if convSeq, getErr := h.seqCache.GetConvSeq(ctx, convID); getErr == nil && convSeq > 0 {
			if setErr := h.seqCache.SetUserSeq(ctx, userID, convID, convSeq); setErr != nil {
				logger.Error("mark read: set user seq after convID mismatch failed", "error", setErr)
			}
		}
		logger.Debug("mark read: msg convID mismatch, used convSeq", "msg_id", msgID, "msg_conv_id", msg.ConvID, "conv_id", convID)
		return nil
	}

	// Use convSeq for unread count calculation, not msgID
	if err := h.seqCache.SetUserSeq(ctx, userID, convID, msg.ConvSeq); err != nil {
		return err
	}

	if msg.SenderID == userID {
		return nil
	}

	payload := protocol.MsgReadNotifyPayload{
		ConvID:    convID,
		UserID:    userID,
		MsgID:     msgID,
		Timestamp: timestamp,
	}
	data, _ := json.Marshal(payload)
	frame := protocol.Frame{
		Type:    protocol.MsgReadNotify,
		ID:      "",
		Payload: data,
	}

	// send to sender's connections
	conns := h.gateway.GetByUserID(ctx, msg.SenderID)
	for _, conn := range conns {
		if c, ok := conn.(interface{ SendFrame(protocol.Frame) error }); ok {
			_ = c.SendFrame(frame)
		}
	}

	// write receipt
	rc := &model.Receipt{
		MsgID:     msgID,
		UserID:    userID,
		Status:    model.ReceiptRead,
		Timestamp: timestamp,
	}
	if h.receipt != nil {
		_ = h.receipt.Upsert(ctx, rc)
	}

	logger.Debug("read notify sent", "conv_id", convID, "sender_id", msg.SenderID)
	return nil
}
