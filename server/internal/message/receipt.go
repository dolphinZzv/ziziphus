package message

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
	"siciv.space/agent/panda_ai/pkg/protocol"
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
			logger.Debug("mark read: msg not found", "msg_id", msgID)
			return nil
		}
		return err
	}

	if msg.ConvID != convID {
		logger.Debug("mark read: msg convID mismatch", "msg_id", msgID, "msg_conv_id", msg.ConvID, "conv_id", convID)
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
			c.SendFrame(frame)
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
		h.receipt.Upsert(ctx, rc)
	}

	logger.Debug("read notify sent", "conv_id", convID, "sender_id", msg.SenderID)
	return nil
}
