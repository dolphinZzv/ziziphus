package message

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"ziziphus/internal/metrics"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
	"ziziphus/pkg/protocol"
)

type Ingest struct {
	store        messageStore
	router       *Router
	pusher       *Pusher
	rateLimit    *RateLimiter
	idGen        idGenerator
	seqCache     seqCache
	convMgr      convManager
	contactReqDB contactRequestDB
	contactRepo  contactCreator
	userDB       userGetter
	whDB         whForwarder
}

type userGetter interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
}

type whForwarder interface {
	ListByConvID(ctx context.Context, convID string) ([]*model.ConvWebhook, error)
}

type contactCreator interface {
	AddContact(ctx context.Context, userID, contactID string) error
}

type messageStore interface {
	Insert(ctx context.Context, msg *model.Message) error
	Get(ctx context.Context, msgID int64) (*model.Message, error)
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

type contactRequestDB interface {
	GetByFormMsgID(ctx context.Context, formMsgID int64) (*model.ContactRequest, error)
	GetByID(ctx context.Context, id int64) (*model.ContactRequest, error)
	Insert(ctx context.Context, req *model.ContactRequest) (int64, error)
	UpdateStatus(ctx context.Context, id int64, status model.ContactRequestStatus) error
	UpdateStatusTx(ctx context.Context, tx pgx.Tx, id int64, status model.ContactRequestStatus) error
	LockByIDTx(ctx context.Context, tx pgx.Tx, id int64) (*model.ContactRequest, error)
	UpdateFormMsgID(ctx context.Context, id, formMsgID int64) error
	Delete(ctx context.Context, id int64) error
}

type convManager interface {
	GetOrCreateP2P(ctx context.Context, userA, userB string) (*model.Conversation, error)
	GetOrCreateSystemConv(ctx context.Context, userID string) (*model.Conversation, error)
	Get(ctx context.Context, convID string) (*model.Conversation, error)
	IsMember(ctx context.Context, convID, userID string) (bool, error)
}

func NewIngest(store messageStore, router *Router, pusher *Pusher, rateLimit *RateLimiter, idGen idGenerator, seqCache seqCache, convMgr convManager, contactReqDB contactRequestDB, contactRepo contactCreator, userDB userGetter, whDB whForwarder) *Ingest {
	return &Ingest{
		store:        store,
		router:       router,
		pusher:       pusher,
		rateLimit:    rateLimit,
		idGen:        idGen,
		seqCache:     seqCache,
		convMgr:      convMgr,
		contactReqDB: contactReqDB,
		contactRepo:  contactRepo,
		userDB:       userDB,
		whDB:         whDB,
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

	// 1.5. FormResponse — handle specially before the normal message flow.
	if model.ContentType(payload.ContentType) == model.ContentFormResponse {
		return in.handleFormResponse(ctx, senderID, sessionID, payload)
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

	// Verify sender is a member of the conversation
	isMember, err := in.convMgr.IsMember(ctx, payload.ConvID, senderID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, model.ErrNotInConv
	}

	// 4. assign IDs
	msgID := in.idGen.NextID()
	convSeq, err := in.seqCache.GetAndIncrementConvSeq(ctx, payload.ConvID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()

	senderName := senderID
	if user, err := in.userDB.GetByID(ctx, senderID); err == nil && user != nil {
		senderName = user.Name
	}

	msg := &model.Message{
		MsgID:           msgID,
		ConvID:          payload.ConvID,
		SenderID:        senderID,
		SenderName:      senderName,
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

	// 7. route + push
	targets := in.router.Route(ctx, msg)
	if len(targets) > 0 {
		in.pusher.Push(context.Background(), msg, targets)
	}

	// 8. webhook forwarding (async, group only)
	if in.whDB != nil {
		go in.forwardToWebhooks(context.Background(), msg)
	}

	return &protocol.MsgSendAckPayload{
		MsgID:     msgID,
		Timestamp: now,
		ClientSeq: payload.ClientSeq,
		Status:    int(model.MsgSent),
	}, nil
}

// SendSystemMessage creates, persists, and pushes a system message (content_type=5).
// senderID is the user who triggered the system message; if non-empty, their
// user_seq is set so the system message doesn't count as unread for them.
func (in *Ingest) SendSystemMessage(ctx context.Context, convID, body string, senderID ...string) (*model.Message, error) {
	msgID := in.idGen.NextID()
	convSeq, err := in.seqCache.GetAndIncrementConvSeq(ctx, convID)
	if err != nil {
		return nil, err
	}
	msg := &model.Message{
		MsgID:           msgID,
		ConvID:          convID,
		SenderID:        "",
		SenderSessionID: strconv.FormatInt(msgID, 10),
		ContentType:     model.ContentSystem,
		Body:            body,
		Timestamp:       time.Now().UnixMilli(),
		ConvSeq:         convSeq,
		Status:          model.MsgSent,
	}
	if err := in.store.Insert(ctx, msg); err != nil {
		return nil, err
	}
	in.seqCache.SetRecentMsg(ctx, convID, msgID, float64(msgID))

	// Set the sender's user_seq so the system message doesn't
	// count as unread for the user who triggered it.
	if len(senderID) > 0 && senderID[0] != "" {
		in.seqCache.SetUserSeq(ctx, senderID[0], convID, convSeq)
	}

	targets := in.router.Route(ctx, msg)
	if len(targets) > 0 {
		in.pusher.Push(ctx, msg, targets)
	}
	return msg, nil
}

// SendFormMessage creates, persists, and pushes a form message (content_type=10).
// It is a general-purpose infrastructure for sending form definitions to a conversation.
func (in *Ingest) SendFormMessage(ctx context.Context, convID string, body *model.FormDefinitionBody) (*model.Message, error) {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal form body: %w", err)
	}

	msgID := in.idGen.NextID()
	convSeq, err := in.seqCache.GetAndIncrementConvSeq(ctx, convID)
	if err != nil {
		return nil, err
	}
	msg := &model.Message{
		MsgID:           msgID,
		ConvID:          convID,
		SenderID:        "",
		SenderSessionID: strconv.FormatInt(msgID, 10),
		ContentType:     model.ContentForm,
		Body:            string(bodyJSON),
		Timestamp:       time.Now().UnixMilli(),
		ConvSeq:         convSeq,
		Status:          model.MsgSent,
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

// handleFormResponse processes a FormResponse message (content_type=11).
func (in *Ingest) handleFormResponse(ctx context.Context, senderID string, sessionID string, payload protocol.MsgSendPayload) (*protocol.MsgSendAckPayload, error) {
	var resp model.FormResponseBody
	if err := json.Unmarshal([]byte(payload.Body), &resp); err != nil {
		return nil, model.ErrBadMsgContent
	}

	// Dedup: check if this client_seq was already processed.
	existing, err := in.store.GetByClientSeq(ctx, senderID, sessionID, payload.ClientSeq)
	if err == nil && existing != nil {
		return &protocol.MsgSendAckPayload{
			MsgID:     existing.MsgID,
			Timestamp: existing.Timestamp,
			ClientSeq: payload.ClientSeq,
			Status:    int(model.MsgSent),
		}, nil
	}

	// Look up the contact request by form_msg_id (most reliable).
	req, err := in.contactReqDB.GetByFormMsgID(ctx, resp.FormMsgID)
	if err != nil {
		return nil, err
	}
	if req == nil {
		// Fallback: try by request_id.
		req, err = in.contactReqDB.GetByID(ctx, resp.RequestID)
		if err != nil {
			return nil, err
		}
	}
	if req == nil {
		return nil, model.ErrContactRequestNotFound
	}

	// Validate permission: sender must be the target of this request.
	if req.ToUserID != senderID {
		return nil, model.ErrNotInConv
	}

	// Validate conversation: must be the sender's own system conversation.
	expectedConvID := model.MakeSystemConvID(senderID)
	if payload.ConvID != expectedConvID {
		return nil, model.ErrNotInConv
	}

	// Validate status: must still be pending (or already approved for idempotent replay).
	if req.Status == model.ContactRequestApproved {
		// Already approved — this is an idempotent replay from the client.
		// Ensure contacts and P2P exist (may be missing from an earlier deploy).
		if resp.Action == "approve" {
			in.ensureContacts(ctx, req.FromUserID, req.ToUserID)
		}
		// Return ack for the already-persisted FormResponse message.
		existing, err := in.store.GetByClientSeq(ctx, senderID, sessionID, payload.ClientSeq)
		if err == nil && existing != nil {
			return &protocol.MsgSendAckPayload{
				MsgID:     existing.MsgID,
				Timestamp: existing.Timestamp,
				ClientSeq: payload.ClientSeq,
				Status:    int(model.MsgSent),
			}, nil
		}
		// Fallback: the FormResponse was already persisted but we return ok anyway.
		return &protocol.MsgSendAckPayload{
			MsgID:     0,
			Timestamp: time.Now().UnixMilli(),
			ClientSeq: payload.ClientSeq,
			Status:    int(model.MsgSent),
		}, nil
	}
	if req.Status != model.ContactRequestPending {
		return nil, model.ErrContactRequestAlreadyHandled
	}

	// Determine new status based on action.
	var newStatus model.ContactRequestStatus
	switch resp.Action {
	case "approve":
		newStatus = model.ContactRequestApproved
	case "reject":
		newStatus = model.ContactRequestRejected
	default:
		return nil, model.ErrBadMsgContent
	}

	// Persist the FormResponse message.
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
		ContentType:     model.ContentFormResponse,
		Body:            payload.Body,
		ReplyTo:         resp.FormMsgID,
		Timestamp:       now,
		ClientSeq:       payload.ClientSeq,
		ConvSeq:         convSeq,
		Status:          model.MsgSent,
	}
	if err := in.store.Insert(ctx, msg); err != nil {
		return nil, err
	}
	metrics.MessagesSentTotal.Inc()
	in.seqCache.SetUserSeq(ctx, senderID, payload.ConvID, convSeq)
	in.seqCache.SetRecentMsg(ctx, payload.ConvID, msgID, float64(msgID))

	// Update contact request status.
	if err := in.contactReqDB.UpdateStatus(ctx, req.ID, newStatus); err != nil {
		// Log but don't fail.
	}

	responderName := senderID
	if resp.ResponderName != "" {
		responderName = resp.ResponderName
	}
	initiatorName := req.FromUserID
	if resp.FormMsgID > 0 {
		if formMsg, err := in.store.Get(ctx, resp.FormMsgID); err == nil && formMsg != nil {
			var formBody model.FormDefinitionBody
			if json.Unmarshal([]byte(formMsg.Body), &formBody) == nil && formBody.FromUserName != "" {
				initiatorName = formBody.FromUserName
			}
		}
	}

	sysB := model.MakeSystemConvID(req.ToUserID)   // B's system conv
	sysA := model.MakeSystemConvID(req.FromUserID) // A's system conv

	if newStatus == model.ContactRequestApproved {
		in.ensureContacts(ctx, req.FromUserID, req.ToUserID)
		in.SendSystemMessage(ctx, sysA, fmt.Sprintf("%s 已通过你的好友申请", responderName))
		in.SendSystemMessage(ctx, sysB, fmt.Sprintf("你已通过 %s 的好友申请", initiatorName))
	} else {
		in.SendSystemMessage(ctx, sysA, fmt.Sprintf("%s 已拒绝你的好友申请", responderName))
		in.SendSystemMessage(ctx, sysB, fmt.Sprintf("你已拒绝 %s 的好友申请", initiatorName))
	}

	// Push the FormResponse to the sender's own system conversation.
	targets := in.router.Route(ctx, msg)
	if len(targets) > 0 {
		in.pusher.Push(ctx, msg, targets)
	}

	return &protocol.MsgSendAckPayload{
		MsgID:     msgID,
		Timestamp: now,
		ClientSeq: payload.ClientSeq,
		Status:    int(model.MsgSent),
	}, nil
}

// ensureContacts creates bidirectional contacts and ensures the P2P conversation exists.
// It is idempotent — safe to call multiple times.
func (in *Ingest) ensureContacts(ctx context.Context, userA, userB string) {
	if in.contactRepo != nil {
		in.contactRepo.AddContact(ctx, userA, userB)
		in.contactRepo.AddContact(ctx, userB, userA)
	}
	if in.convMgr != nil {
		in.convMgr.GetOrCreateP2P(ctx, userA, userB)
	}
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


// forwardToWebhooks forwards a message to configured webhook callbacks.
func (in *Ingest) forwardToWebhooks(ctx context.Context, msg *model.Message) {
	conv, err := in.convMgr.Get(ctx, msg.ConvID)
	if err != nil || conv.Type != model.ConvGroup {
		return
	}

	whList, err := in.whDB.ListByConvID(ctx, msg.ConvID)
	if err != nil || len(whList) == 0 {
		return
	}

	mentioned := extractMentions(msg.Body)

	for _, wh := range whList {
		if wh.CallbackURL == "" {
			continue
		}
		if len(mentioned) > 0 && !mentioned[wh.Name] {
			continue
		}
		go in.sendWebhookWithRetry(ctx, wh, msg)
	}
}

func (in *Ingest) sendWebhookWithRetry(ctx context.Context, wh *model.ConvWebhook, msg *model.Message) {
	payload := buildWebhookPayload(wh.ID, msg)
	body, _ := json.Marshal(payload)

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(1<<uint(attempt+1)-1) * time.Second
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", wh.CallbackURL, bytes.NewReader(body))
		if err != nil {
			logger.Warn("webhook forward request creation failed", "wh_id", wh.ID, "error", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Ziziphus-Webhook/1.0")
		req.Header.Set("X-Signature", computeSignature([]byte(wh.APIKeyHash), body))
		for _, h := range wh.Headers {
			req.Header.Set(h.Key, h.Value)
		}

		resp, doErr := http.DefaultClient.Do(req)
		if doErr == nil && resp.StatusCode < 500 {
			resp.Body.Close()
			in.logWhAudit(ctx, wh.ID, msg.MsgID, "forward", "")
			return
		}
		if doErr != nil {
			logger.Warn("webhook forward attempt failed",
				"wh_id", wh.ID, "attempt", attempt, "error", doErr)
		} else {
			resp.Body.Close()
		}
	}
	in.logWhAudit(ctx, wh.ID, msg.MsgID, "forward_fail",
		"max retries exhausted")
}

func (in *Ingest) logWhAudit(ctx context.Context, whID int64, msgID int64, action, reason string) {
	logger.Info("webhook forward", "webhook_id", whID, "msg_id", msgID, "action", action, "reason", reason)
}

func buildWebhookPayload(whID int64, msg *model.Message) map[string]any {
	return map[string]any{
		"event":    "message.created",
		"webhook_id": whID,
		"conv_id":  msg.ConvID,
		"message": map[string]any{
			"msg_id":       msg.MsgID,
			"sender_id":    msg.SenderID,
			"sender_name":  msg.SenderName,
			"content_type": int(msg.ContentType),
			"body":         msg.Body,
			"reply_to":     msg.ReplyTo,
			"timestamp":    msg.Timestamp,
		},
	}
}

func computeSignature(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func extractMentions(body string) map[string]bool {
	result := make(map[string]bool)
	words := strings.Fields(body)
	for _, w := range words {
		if strings.HasPrefix(w, "@") {
			name := strings.TrimLeft(w, "@,.!?;:")
			if name != "" {
				result[name] = true
			}
		}
	}
	return result
}

