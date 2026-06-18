package message

import (
	"context"

	"siciv.space/agent/panda_ai/pkg/model"
	"siciv.space/agent/panda_ai/pkg/protocol"
)

type SyncHandler struct {
	msgRepo  syncMsgRepo
	seqCache syncSeqCache
}

type syncMsgRepo interface {
	GetMessagesSinceSeq(ctx context.Context, convID string, lastSeq int64, limit int) ([]*model.Message, error)
}

type syncSeqCache interface {
	SetSessionSeq(ctx context.Context, sessionID, convID string, seq int64) error
}

func NewSyncHandler(msgRepo syncMsgRepo, seqCache syncSeqCache) *SyncHandler {
	return &SyncHandler{
		msgRepo:  msgRepo,
		seqCache: seqCache,
	}
}

func (h *SyncHandler) Handle(ctx context.Context, sessionID string, req protocol.SyncReqPayload) (*protocol.SyncResPayload, error) {
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	messages, err := h.msgRepo.GetMessagesSinceSeq(ctx, req.ConvID, req.LastConvSeq, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	// advance session seq
	if len(messages) > 0 {
		lastSeq := messages[len(messages)-1].ConvSeq
		h.seqCache.SetSessionSeq(ctx, sessionID, req.ConvID, lastSeq)
	}

	syncMsgs := make([]protocol.SyncMessage, 0, len(messages))
	for _, m := range messages {
		syncMsgs = append(syncMsgs, protocol.SyncMessage{
			MsgID:       m.MsgID,
			SenderID:    m.SenderID,
			ContentType: int(m.ContentType),
			Body:        m.Body,
			Timestamp:   m.Timestamp,
			ConvSeq:     m.ConvSeq,
		})
	}

	return &protocol.SyncResPayload{
		ConvID:   req.ConvID,
		Messages: syncMsgs,
		HasMore:  hasMore,
	}, nil
}
