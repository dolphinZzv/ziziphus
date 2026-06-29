package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"siciv.space/agent/panda_ai/internal/auth"
	"siciv.space/agent/panda_ai/internal/message"
	"siciv.space/agent/panda_ai/internal/storage/db"
	"siciv.space/agent/panda_ai/pkg/i18n"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

// ---------- interfaces ----------

type whConvManager interface {
	Get(ctx context.Context, convID string) (*model.Conversation, error)
	IsMember(ctx context.Context, convID, userID string) (bool, error)
	GetMembers(ctx context.Context, convID string) ([]*model.ConvMember, error)
}

type whUserGetter interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
}

type whMessageStore interface {
	Insert(ctx context.Context, msg *model.Message) error
}

type whPusher interface {
	Push(ctx context.Context, msg *model.Message, targets []message.RouteTarget)
}

type whRouter interface {
	Route(ctx context.Context, msg *model.Message) []message.RouteTarget
}

type whSysMsgSender interface {
	SendSystemMessage(ctx context.Context, convID, body string, senderID ...string) (*model.Message, error)
}



// ---------- rate limiter per IP ----------

type ipRateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*ipBucket
	rate      int
	burst     int
}

type ipBucket struct {
	tokens    float64
	lastCheck time.Time
}

func newIPRateLimiter(rate, burst int) *ipRateLimiter {
	return &ipRateLimiter{
		buckets: make(map[string]*ipBucket),
		rate:    rate,
		burst:   burst,
	}
}

func (rl *ipRateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[ip]
	if !ok {
		b = &ipBucket{tokens: float64(rl.burst), lastCheck: time.Now()}
		rl.buckets[ip] = b
	}

	now := time.Now()
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * float64(rl.rate)
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.lastCheck = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// ---------- helpers ----------

func generateToken(prefix string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}

func hashAPIKey(key string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func checkCIDR(cidrList []string, ipStr string) bool {
	if len(cidrList) == 0 {
		return true
	}
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false
	}
	for _, cidr := range cidrList {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			logger.Warn("invalid CIDR in webhook whitelist", "cidr", cidr)
			continue
		}
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func callerIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ---------- handler ----------

type whIDGen interface {
	NextID() int64
}

type whSeqCache interface {
	GetAndIncrementConvSeq(ctx context.Context, convID string) (int64, error)
	SetUserSeq(ctx context.Context, userID, convID string, seq int64) error
	SetRecentMsg(ctx context.Context, convID string, msgID int64, score float64) error
}

type WebhookHandler struct {
	webhookDB db.ConvWebhookDB
	idGen    whIDGen
	convMgr  whConvManager
	userDB   whUserGetter
	store    whMessageStore
	router   whRouter
	pusher   whPusher
	seqCache whSeqCache
	sysMsg   whSysMsgSender
	rateLmt  *ipRateLimiter
}

func NewWebhookHandler(
	whDB db.ConvWebhookDB,
	idGen whIDGen, convMgr whConvManager, userDB whUserGetter,
	store whMessageStore, router whRouter, pusher whPusher,
	seqCache whSeqCache, sysMsg whSysMsgSender,
) *WebhookHandler {
	return &WebhookHandler{
		webhookDB: whDB,
		idGen:     idGen,
		convMgr:   convMgr,
		userDB:    userDB,
		store:     store,
		router:    router,
		pusher:    pusher,
		seqCache:  seqCache,
		sysMsg:    sysMsg,
		rateLmt:   newIPRateLimiter(10, 20),
	}
}

// ---------- auth helpers ----------

func (h *WebhookHandler) isConvAdmin(ctx context.Context, convID, userID string) bool {
	members, err := h.convMgr.GetMembers(ctx, convID)
	if err != nil {
		return false
	}
	for _, m := range members {
		if m.UserID == userID && (m.Role == model.ConvRoleAdmin || m.Role == model.ConvRoleOwner) {
			return true
		}
	}
	return false
}

func (h *WebhookHandler) canManageWebhook(ctx context.Context, wh *model.ConvWebhook, userID string) bool {
	if wh.CreatedBy == userID {
		return true
	}
	return h.isConvAdmin(ctx, wh.ConvID, userID)
}

// ---------- CRUD ----------

func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())
	if !h.isConvAdmin(r.Context(), convID, userID) {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
		return
	}
	list, err := h.webhookDB.ListByConvID(r.Context(), convID)
	if err != nil {
		logger.Error("list webhooks failed", "conv_id", convID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if list == nil {
		list = []*model.ConvWebhook{}
	}
	// Strip sensitive fields
	type safeWebhook struct {
		ID            int64           `json:"id"`
		ConvID        string          `json:"conv_id"`
		Name          string          `json:"name"`
		CallbackURL   string          `json:"callback_url,omitempty"`
		Headers       []model.WebhookHeader `json:"headers,omitempty"`
		CIDRWhitelist []string        `json:"cidr_whitelist,omitempty"`
		RequireAudit  bool            `json:"require_audit"`
		CreatedBy     string          `json:"created_by"`
		CreatedAt     int64           `json:"created_at"`
	}
	safe := make([]safeWebhook, 0, len(list))
	for _, wh := range list {
		safe = append(safe, safeWebhook{
			ID: wh.ID, ConvID: wh.ConvID, Name: wh.Name,
			CallbackURL: wh.CallbackURL, Headers: wh.Headers,
			CIDRWhitelist: wh.CIDRWhitelist, RequireAudit: wh.RequireAudit,
			CreatedBy: wh.CreatedBy, CreatedAt: wh.CreatedAt,
		})
	}
	JSON(w, safe)
}

func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())
	if !h.isConvAdmin(r.Context(), convID, userID) {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
		return
	}

	var body struct {
		Name          string                `json:"name"`
		CallbackURL   string                `json:"callback_url"`
		Headers       []model.WebhookHeader `json:"headers"`
		CIDRWhitelist []string              `json:"cidr_whitelist"`
		RequireAudit  bool                  `json:"require_audit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_json"))
		return
	}
	if body.Name == "" {
		BadRequest(w, r, "name is required")
		return
	}

	token, err := generateToken("wh_")
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	apiKey, err := generateToken("")
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	hash, err := hashAPIKey(apiKey)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	if body.Headers == nil {
		body.Headers = []model.WebhookHeader{}
	}
	if body.CIDRWhitelist == nil {
		body.CIDRWhitelist = []string{}
	}

	wh := &model.ConvWebhook{
		ConvID:        convID,
		Name:          body.Name,
		Token:         token,
		APIKeyHash:    hash,
		CallbackURL:   body.CallbackURL,
		Headers:       body.Headers,
		CIDRWhitelist: body.CIDRWhitelist,
		RequireAudit:  body.RequireAudit,
		CreatedBy:     userID,
	}

	created, err := h.webhookDB.Create(r.Context(), wh)
	if err != nil {
		logger.Error("create webhook failed", "error", err)
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			BadRequest(w, r, "webhook name already exists in this conversation")
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	JSON(w, map[string]any{
		"id":             created.ID,
		"token":          token,
		"api_key":        apiKey,
		"name":           created.Name,
		"callback_url":   created.CallbackURL,
		"headers":        created.Headers,
		"cidr_whitelist": created.CIDRWhitelist,
		"require_audit":  created.RequireAudit,
	})
}

func (h *WebhookHandler) Update(w http.ResponseWriter, r *http.Request) {
	webhookID, err := strconvParseInt(chi.URLParam(r, "webhook_id"))
	if err != nil {
		BadRequest(w, r, "invalid webhook_id")
		return
	}
	userID := auth.UserFromCtx(r.Context())

	wh, err := h.webhookDB.GetByID(r.Context(), webhookID)
	if err != nil {
		NotFound(w, r)
		return
	}
	if !h.canManageWebhook(r.Context(), wh, userID) {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
		return
	}

	var body struct {
		Name          string                `json:"name"`
		CallbackURL   string                `json:"callback_url"`
		Headers       []model.WebhookHeader `json:"headers"`
		CIDRWhitelist []string              `json:"cidr_whitelist"`
		RequireAudit  *bool                 `json:"require_audit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_json"))
		return
	}

	if body.Name != "" {
		wh.Name = body.Name
	}
	wh.CallbackURL = body.CallbackURL
	if body.Headers != nil {
		wh.Headers = body.Headers
	}
	if body.CIDRWhitelist != nil {
		wh.CIDRWhitelist = body.CIDRWhitelist
	}
	if body.RequireAudit != nil {
		wh.RequireAudit = *body.RequireAudit
	}

	if err := h.webhookDB.Update(r.Context(), wh); err != nil {
		logger.Error("update webhook failed", "id", webhookID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"status": "ok"})
}

func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	webhookID, err := strconvParseInt(chi.URLParam(r, "webhook_id"))
	if err != nil {
		BadRequest(w, r, "invalid webhook_id")
		return
	}
	userID := auth.UserFromCtx(r.Context())

	wh, err := h.webhookDB.GetByID(r.Context(), webhookID)
	if err != nil {
		NotFound(w, r)
		return
	}
	if !h.canManageWebhook(r.Context(), wh, userID) {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
		return
	}

	if err := h.webhookDB.Delete(r.Context(), webhookID); err != nil {
		logger.Error("delete webhook failed", "id", webhookID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"status": "ok"})
}

func (h *WebhookHandler) RegenerateKey(w http.ResponseWriter, r *http.Request) {
	webhookID, err := strconvParseInt(chi.URLParam(r, "webhook_id"))
	if err != nil {
		BadRequest(w, r, "invalid webhook_id")
		return
	}
	userID := auth.UserFromCtx(r.Context())

	wh, err := h.webhookDB.GetByID(r.Context(), webhookID)
	if err != nil {
		NotFound(w, r)
		return
	}
	if !h.canManageWebhook(r.Context(), wh, userID) {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
		return
	}

	apiKey, err := generateToken("")
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	hash, err := hashAPIKey(apiKey)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	if err := h.webhookDB.UpdateAPIKeyHash(r.Context(), webhookID, hash); err != nil {
		logger.Error("regenerate key failed", "id", webhookID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"api_key": apiKey})
}

// ---------- audit logs ----------

func (h *WebhookHandler) Logs(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())
	if !h.isConvAdmin(r.Context(), convID, userID) {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
		return
	}
	page, size := parsePageSize(r)
	logs, total, err := h.webhookDB.ListAuditLogs(r.Context(), convID, page, size)
	if err != nil {
		logger.Error("list audit logs failed", "conv_id", convID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if logs == nil {
		logs = []*model.WebhookAuditLog{}
	}
	Paginated(w, logs, total, page, size)
}

// ---------- pending messages ----------

func (h *WebhookHandler) PendingMessages(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())
	if !h.isConvAdmin(r.Context(), convID, userID) {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
		return
	}
	list, err := h.webhookDB.ListPendingAudit(r.Context(), convID)
	if err != nil {
		logger.Error("list pending audit failed", "conv_id", convID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if list == nil {
		list = []*model.WebhookMessage{}
	}
	JSON(w, list)
}

// ---------- approve / reject ----------

type auditActionReq struct {
	Reason string `json:"reason,omitempty"`
}

func (h *WebhookHandler) ApproveMessage(w http.ResponseWriter, r *http.Request) {
	h.handleAudit(w, r, "approved")
}

func (h *WebhookHandler) RejectMessage(w http.ResponseWriter, r *http.Request) {
	h.handleAudit(w, r, "rejected")
}

func (h *WebhookHandler) handleAudit(w http.ResponseWriter, r *http.Request, newStatus string) {
	msgID, err := strconvParseInt(chi.URLParam(r, "msg_id"))
	if err != nil {
		BadRequest(w, r, "invalid msg_id")
		return
	}
	userID := auth.UserFromCtx(r.Context())

	wm, err := h.webhookDB.GetWebhookMessage(r.Context(), msgID)
	if err != nil {
		NotFound(w, r)
		return
	}
	if wm.AuditStatus != "pending" {
		BadRequest(w, r, "message is not pending audit")
		return
	}

	// Verify admin
	if !h.isConvAdmin(r.Context(), wm.ConvID, userID) {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
		return
	}

	var body auditActionReq
	json.NewDecoder(r.Body).Decode(&body)

	// Update status
	if err := h.webhookDB.UpdateAuditStatus(r.Context(), msgID, newStatus, userID); err != nil {
		logger.Error("update audit status failed", "msg_id", msgID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	// Write audit log
	h.webhookDB.InsertAuditLog(r.Context(), &model.WebhookAuditLog{
		WebhookID: wm.WebhookID,
		ConvID:    wm.ConvID,
		MsgID:     msgID,
		Action:    newStatus,
		ActorID:   userID,
		Reason:    body.Reason,
		CreatedAt: time.Now().UnixMilli(),
	})

	// If approved, push to all members
	if newStatus == "approved" {
		h.pushWebhookMessage(r.Context(), wm.ConvID, msgID)
	}

	JSON(w, map[string]string{"status": newStatus})
}

func (h *WebhookHandler) pushWebhookMessage(ctx context.Context, convID string, msgID int64) {
	msg := &model.Message{MsgID: msgID, ConvID: convID, Status: model.MsgSent}
	targets := h.router.Route(ctx, msg)
	if len(targets) > 0 {
		h.pusher.Push(ctx, msg, targets)
	}
}

// ---------- receive message (public) ----------

type webhookReceiveReq struct {
	ContentType int      `json:"content_type"`
	Body        string   `json:"body"`
	ReplyTo     int64    `json:"reply_to"`
}

func (h *WebhookHandler) ReceiveMessage(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	// Rate limit
	ip := callerIP(r)
	if !h.rateLmt.Allow(ip) {
		http.Error(w, `{"code":429,"msg":"too many requests"}`, http.StatusTooManyRequests)
		return
	}

	// Lookup webhook by token
	wh, err := h.webhookDB.GetByToken(r.Context(), token)
	if err != nil {
		http.Error(w, `{"code":404,"msg":"not found"}`, http.StatusNotFound)
		return
	}

	// API key verification
	if wh.APIKeyHash != "" {
		providedKey := extractBearerToken(r)
		if providedKey == "" {
			http.Error(w, `{"code":401,"msg":"missing api key"}`, http.StatusUnauthorized)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(wh.APIKeyHash), []byte(providedKey)); err != nil {
			http.Error(w, `{"code":401,"msg":"invalid api key"}`, http.StatusUnauthorized)
			return
		}
	}

	// CIDR check
	if !checkCIDR(wh.CIDRWhitelist, ip) {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: "ip not in whitelist"})
		return
	}

	// Parse body
	var req webhookReceiveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"code":400,"msg":"invalid json"}`, http.StatusBadRequest)
		return
	}
	if req.Body == "" {
		http.Error(w, `{"code":400,"msg":"body is required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Body) > 65536 {
		http.Error(w, `{"code":413,"msg":"body too large"}`, http.StatusRequestEntityTooLarge)
		return
	}
	if req.ContentType == 0 {
		req.ContentType = 1 // default to text
	}

	// Build message
	now := time.Now().UnixMilli()
	msgID := h.idGen.NextID()
	convSeq, err := h.seqCache.GetAndIncrementConvSeq(r.Context(), wh.ConvID)
	if err != nil {
		logger.Error("get conv seq failed", "conv_id", wh.ConvID, "error", err)
		http.Error(w, `{"code":500,"msg":"server error"}`, http.StatusInternalServerError)
		return
	}

	senderID := fmt.Sprintf("webhook:%d", wh.ID)
	msg := &model.Message{
		MsgID:       msgID,
		ConvID:      wh.ConvID,
		SenderID:    senderID,
		SenderName:  wh.Name,
		ContentType: model.ContentType(req.ContentType),
		Body:        req.Body,
		ReplyTo:     req.ReplyTo,
		Timestamp:   now,
		ConvSeq:     convSeq,
		Status:      model.MsgSent,
	}

	// Persist
	if err := h.store.Insert(r.Context(), msg); err != nil {
		logger.Error("persist webhook message failed", "error", err)
		http.Error(w, `{"code":500,"msg":"server error"}`, http.StatusInternalServerError)
		return
	}

	// Record webhook_message association
	wm := &model.WebhookMessage{
		MsgID:     msgID,
		WebhookID: wh.ID,
		ConvID:    wh.ConvID,
		SourceIP:  ip,
		CreatedAt: now,
	}

	// Audit flow
	if wh.RequireAudit {
		wm.AuditStatus = "pending"
		if err := h.webhookDB.InsertWebhookMessage(r.Context(), wm); err != nil {
			logger.Error("insert webhook_message failed", "error", err)
		}

		// Only push to admins
		h.pushToAdmins(r.Context(), wh.ConvID, msg)

		// Notify admins
		h.sysMsg.SendSystemMessage(r.Context(), wh.ConvID,
			fmt.Sprintf("webhook %s 发来一条消息待审核", wh.Name))

		// Audit log
		h.webhookDB.InsertAuditLog(r.Context(), &model.WebhookAuditLog{
			WebhookID: wh.ID,
			ConvID:    wh.ConvID,
			MsgID:     msgID,
			Action:    "send",
			ActorID:   senderID,
			CallerIP:  ip,
			CreatedAt: now,
		})

		JSON(w, map[string]any{
			"msg_id":       msgID,
			"audit_status": "pending",
			"timestamp":    now,
		})
		return
	}

	// No audit required — push to all
	wm.AuditStatus = ""
	if err := h.webhookDB.InsertWebhookMessage(r.Context(), wm); err != nil {
		logger.Error("insert webhook_message failed", "error", err)
	}

	targets := h.router.Route(r.Context(), msg)
	if len(targets) > 0 {
		h.pusher.Push(r.Context(), msg, targets)
	}

	// Audit log
	h.webhookDB.InsertAuditLog(r.Context(), &model.WebhookAuditLog{
		WebhookID: wh.ID,
		ConvID:    wh.ConvID,
		MsgID:     msgID,
		Action:    "send",
		ActorID:   senderID,
		CallerIP:  ip,
		CreatedAt: now,
	})

	h.seqCache.SetRecentMsg(r.Context(), wh.ConvID, msgID, float64(msgID))

	JSON(w, map[string]any{
		"msg_id":       msgID,
		"audit_status": "approved",
		"timestamp":    now,
	})
}

func (h *WebhookHandler) pushToAdmins(ctx context.Context, convID string, msg *model.Message) {
	members, err := h.convMgr.GetMembers(ctx, convID)
	if err != nil {
		return
	}
	var adminIDs []string
	for _, m := range members {
		if m.Role == model.ConvRoleAdmin || m.Role == model.ConvRoleOwner {
			adminIDs = append(adminIDs, m.UserID)
		}
	}
	if len(adminIDs) == 0 {
		return
	}
	// Route to specific admin sessions
	targets := h.router.Route(ctx, msg)
	if len(targets) > 0 {
		// Filter targets to admins only
		var filtered []message.RouteTarget
		for _, t := range targets {
			for _, aid := range adminIDs {
				if t.UserID == aid {
					filtered = append(filtered, t)
					break
				}
			}
		}
		if len(filtered) > 0 {
			h.pusher.Push(ctx, msg, filtered)
		}
	}
}

// ---------- helpers ----------

func strconvParseInt(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}

func parsePageSize(r *http.Request) (int, int) {
	page := 1
	size := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconvParseInt(p); err == nil && v > 0 {
			page = int(v)
		}
	}
	if s := r.URL.Query().Get("size"); s != "" {
		if v, err := strconvParseInt(s); err == nil && v > 0 {
			size = int(v)
		}
	}
	return page, size
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// ---------- HMAC signature (for outgoing callbacks) ----------

func ComputeSignature(secret []byte, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// ---------- extract @mentions from message body ----------

func ExtractMentions(body string) map[string]bool {
	result := make(map[string]bool)
	words := strings.Fields(body)
	for _, w := range words {
		if strings.HasPrefix(w, "@") {
			name := strings.TrimLeft(w, "@,.!?;:\"'")
			if name != "" {
				result[name] = true
			}
		}
	}
	return result
}

