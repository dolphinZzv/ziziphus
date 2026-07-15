package message

import (
	"context"
	"encoding/json"
	"time"

	"ziziphus/internal/metrics"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
	"ziziphus/pkg/protocol"
)

type Pusher struct {
	gateway connRegistry
	receipt receiptWriter
}

type connRegistry interface {
	GetBySessionID(ctx context.Context, sessionID string) any
	GetByUserID(ctx context.Context, userID string) []any
}

type receiptWriter interface {
	Upsert(ctx context.Context, receipt *model.Receipt) error
}

func NewPusher(gateway connRegistry, receipt receiptWriter) *Pusher {
	return &Pusher{
		gateway: gateway,
		receipt: receipt,
	}
}

func (p *Pusher) Push(ctx context.Context, msg *model.Message, targets []RouteTarget) {
	var pushCount int
	defer func() {
		metrics.MessagesPushTotal.Add(float64(pushCount))
	}()

	pushPayload := protocol.MsgPushPayload{
		MsgID:       msg.MsgID,
		ConvID:      msg.ConvID,
		SenderID:    msg.SenderID,
		SenderName:  msg.SenderName,
		ContentType: int(msg.ContentType),
		Body:        msg.Body,
		ReplyTo:     msg.ReplyTo,
		Mention:     msg.Mention,
		Timestamp:   msg.Timestamp,
		ConvSeq:     msg.ConvSeq,
	}
	pushData, _ := json.Marshal(pushPayload)
	frame := protocol.Frame{
		Type:    protocol.MsgPush,
		ID:      "",
		Payload: pushData,
	}

	for _, target := range targets {
		if len(target.SessionIDs) == 0 {
			continue
		}
		for _, sessionID := range target.SessionIDs {
			conn := p.gateway.GetBySessionID(ctx, sessionID)
			if conn == nil {
				continue
			}
			// Send push
			c, ok := conn.(interface{ SendFrame(protocol.Frame) error })
			if !ok {
				continue
			}
			if err := c.SendFrame(frame); err != nil {
				logger.Warn("push failed", "session_id", sessionID, "error", err)
				continue
			}
			pushCount++

			// write delivery receipt
			receipt := &model.Receipt{
				MsgID:     msg.MsgID,
				UserID:    target.UserID,
				SessionID: sessionID,
				Status:    model.ReceiptDelivered,
				Timestamp: time.Now().UnixMilli(),
			}
			if err := p.receipt.Upsert(ctx, receipt); err != nil {
				logger.Warn("receipt upsert failed", "msg_id", msg.MsgID, "error", err)
			}
		}
	}
}
