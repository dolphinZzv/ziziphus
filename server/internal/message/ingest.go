package message

import (
	"context"
	"strings"
	"time"

	"github.com/dolphinz/im-server/internal/metrics"
	"github.com/dolphinz/im-server/pkg/model"
	"github.com/dolphinz/im-server/pkg/protocol"
)

type Ingest struct {
	store     messageStore
	router    *Router
	pusher    *Pusher
	rateLimit *RateLimiter
	idGen     idGenerator
	seqCache  seqCache
	convMgr   convManager
}

type messageStore interface {
	Insert(ctx context.Context, msg *model.Message) error
	GetByClientSeq(ctx context.Context, senderID, sessionID string, clientSeq int64) (*model.Message, error)
}

type idGenerator interface {
	NextID() int64
}

type seqCache interface {
	GetAndIncrementConvSeq(ctx context.Context, convID string) (int64, error)
	SetUserSeq(ctx context.Context, userID, convID string, seq int64) error
	SetRecentMsg(ctx context.Context, convID string, msgID int64, score float64) error
}

type convManager interface {
	GetOrCreateP2P(ctx context.Context, userA, userB string) (*model.Conversation, error)
	Get(ctx context.Context, convID string) (*model.Conversation, error)
	IsMember(ctx context.Context, convID, userID string) (bool, error)
}

func NewIngest(store messageStore, router *Router, pusher *Pusher, rateLimit *RateLimiter, idGen idGenerator, seqCache seqCache, convMgr convManager) *Ingest {
	return &Ingest{
		store:     store,
		router:    router,
		pusher:    pusher,
		rateLimit: rateLimit,
		idGen:     idGen,
		seqCache:  seqCache,
		convMgr:   convMgr,
	}
}

func (in *Ingest) Ingest(ctx context.Context, senderID, sessionID string, payload protocol.MsgSendPayload) (*protocol.MsgSendAckPayload, error) {
	// 1. rate limit
	if err := in.rateLimit.Check(ctx, senderID); err != nil {
		return nil, err
	}
	if err := in.rateLimit.CheckBodySize(payload.Body); err != nil {
		return nil, err
	}

	// 2. dedup
	existing, err := in.store.GetByClientSeq(ctx, senderID, sessionID, payload.ClientSeq)
	if err == nil && existing != nil {
		return &protocol.MsgSendAckPayload{
			MsgID:     existing.MsgID,
			Timestamp: existing.Timestamp,
			ClientSeq: payload.ClientSeq,
			Status:    int(model.MsgSent),
		}, nil
	}

	// 3. auto-create P2P conversation if not exists
	if _, err := in.convMgr.Get(ctx, payload.ConvID); err != nil {
		if model.IsP2PConvID(payload.ConvID) {
			otherID := parseP2PCounterpart(payload.ConvID, senderID)
			if otherID == "" {
				return nil, model.ErrConvNotFound
			}
			if _, err = in.convMgr.GetOrCreateP2P(ctx, senderID, otherID); err != nil {
				return nil, err
			}
		} else {
			return nil, model.ErrConvNotFound
		}
	}

	// 4. assign IDs
	msgID := in.idGen.NextID()
	convSeq, err := in.seqCache.GetAndIncrementConvSeq(ctx, payload.ConvID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()

	msg := &model.Message{
		MsgID:           msgID,
		ConvID:          payload.ConvID,
		SenderID:        senderID,
		SenderSessionID: sessionID,
		ContentType:     model.ContentType(payload.ContentType),
		Body:            payload.Body,
		Mention:         payload.Mention,
		ReplyTo:         payload.ReplyTo,
		Timestamp:       now,
		ClientSeq:       payload.ClientSeq,
		ConvSeq:         convSeq,
		Status:          model.MsgSent,
	}

	// 5. persist
	if err := in.store.Insert(ctx, msg); err != nil {
		return nil, err
	}
	metrics.MessagesSentTotal.Inc()

	// 6. set sender's user_seq so self-messages don't count as unread
	in.seqCache.SetUserSeq(ctx, senderID, payload.ConvID, convSeq)

	// 6. cache recent
	in.seqCache.SetRecentMsg(ctx, payload.ConvID, msgID, float64(msgID))

	// 7. route + push (async — ack must arrive before push to prevent client timeout)
	targets := in.router.Route(ctx, msg)
	if len(targets) > 0 {
		go in.pusher.Push(context.Background(), msg, targets)
	}

	return &protocol.MsgSendAckPayload{
		MsgID:     msgID,
		Timestamp: now,
		ClientSeq: payload.ClientSeq,
		Status:    int(model.MsgSent),
	}, nil
}

// SendSystemMessage creates, persists, and pushes a system message (content_type=5).
func (in *Ingest) SendSystemMessage(ctx context.Context, convID, body string) (*model.Message, error) {
	msgID := in.idGen.NextID()
	convSeq, err := in.seqCache.GetAndIncrementConvSeq(ctx, convID)
	if err != nil {
		return nil, err
	}
	msg := &model.Message{
		MsgID:       msgID,
		ConvID:      convID,
		SenderID:    "",
		ContentType: model.ContentSystem,
		Body:        body,
		Timestamp:   time.Now().UnixMilli(),
		ConvSeq:     convSeq,
		Status:      model.MsgSent,
	}
	if err := in.store.Insert(ctx, msg); err != nil {
		return nil, err
	}
	in.seqCache.SetRecentMsg(ctx, convID, msgID, float64(msgID))

	targets := in.router.Route(ctx, msg)
	if len(targets) > 0 {
		in.pusher.Push(ctx, msg, targets)
	}
	return msg, nil
}

// parseP2PCounterpart extracts the other user's ID from a P2P convID.
// convID format: "sorted_user_a:sorted_user_b"
func parseP2PCounterpart(convID, senderID string) string {
	parts := strings.SplitN(convID, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	if parts[0] == senderID {
		return parts[1]
	}
	return parts[0]
}
