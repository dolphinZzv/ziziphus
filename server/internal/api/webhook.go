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
	"ziziphus/internal/auth"
	"ziziphus/internal/message"
	"ziziphus/internal/storage/db"
	"ziziphus/pkg/i18n"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
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
	mu      sync.Mutex
	buckets map[string]*ipBucket
	rate    int
	burst   int
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
	idGen     whIDGen
	convMgr   whConvManager
	userDB    whUserGetter
	store     whMessageStore
	router    whRouter
	pusher    whPusher
	seqCache  whSeqCache
	sysMsg    whSysMsgSender
	rateLmt   *ipRateLimiter
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

// List returns all webhooks for a conversation.
// @Summary List webhooks
// @Description Returns all webhooks configured for the specified conversation.
// @Tags webhooks
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Success 200 {array} APIResponse
// @Router /conversations/{conv_id}/webhooks [get]
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
	JSON(w, list)
}

// Create adds a new webhook to a conversation.
// @Summary Create webhook
// @Description Creates a new webhook configuration for the specified conversation.
// @Tags webhooks
// @Security Bearer
// @Accept json
// @Param conv_id path string true "Conversation ID"
// @Success 200 {object} APIResponse
// @Router /conversations/{conv_id}/webhooks [post]
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
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_json"))
		return
	}
	if body.Name == "" {
		BadRequest(w, r, "name is required")
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
		APIKeyPlain:   apiKey,
		APIKeyHash:    hash,
		CallbackURL:   body.CallbackURL,
		Headers:       body.Headers,
		CIDRWhitelist: body.CIDRWhitelist,
		CreatedBy:     userID,
	}

	created, err := h.webhookDB.Create(r.Context(), wh)
	if err != nil {
		logger.Error("create webhook failed", "error", err)
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			BadRequest(w, r, "webhook name already exists in this conversation")
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	JSON(w, created)
}

// Update modifies an existing webhook.
// @Summary Update webhook
// @Description Updates the configuration of an existing webhook.
// @Tags webhooks
// @Security Bearer
// @Accept json
// @Param conv_id path string true "Conversation ID"
// @Param webhook_id path string true "Webhook ID"
// @Success 200 {object} APIResponse
// @Router /conversations/{conv_id}/webhooks/{webhook_id} [put]
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

	if err := h.webhookDB.Update(r.Context(), wh); err != nil {
		logger.Error("update webhook failed", "id", webhookID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"status": "ok"})
}

// Delete removes a webhook.
// @Summary Delete webhook
// @Description Deletes a webhook configuration from the conversation.
// @Tags webhooks
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Param webhook_id path string true "Webhook ID"
// @Success 200 {object} APIResponse
// @Router /conversations/{conv_id}/webhooks/{webhook_id} [delete]
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

// Test sends a test message through the webhook.
// @Summary Test webhook
// @Description Sends a test message via the specified webhook to verify configuration.
// @Tags webhooks
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Param webhook_id path string true "Webhook ID"
// @Success 200 {object} map[string]interface{}
// @Router /conversations/{conv_id}/webhooks/{webhook_id}/test [post]
func (h *WebhookHandler) Test(w http.ResponseWriter, r *http.Request) {
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

	now := time.Now().UnixMilli()
	msgID := h.idGen.NextID()
	convSeq, _ := h.seqCache.GetAndIncrementConvSeq(r.Context(), wh.ConvID)
	msg := &model.Message{
		MsgID:           msgID,
		ConvID:          wh.ConvID,
		SenderID:        fmt.Sprintf("webhook:%d", wh.ID),
		SenderSessionID: fmt.Sprintf("test:%d", now),
		SenderName:      wh.Name,
		ContentType:     0,
		Body:            fmt.Sprintf("🔔 Webhook 测试消息 (%d)", now%10000),
		Timestamp:       now,
		ClientSeq:       now,
		ConvSeq:         convSeq,
		Status:          model.MsgSent,
	}
	if err := h.store.Insert(r.Context(), msg); err != nil {
		logger.Error("test webhook persist failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	targets := h.router.Route(r.Context(), msg)
	if len(targets) > 0 {
		h.pusher.Push(r.Context(), msg, targets)
	}

	JSON(w, map[string]any{
		"status": "ok",
		"msg_id": msgID,
		"body":   msg.Body,
	})
}

// RegenerateKey generates a new API key for the webhook.
// @Summary Regenerate API key
// @Description Generates a new API key for the specified webhook. The old key becomes invalid.
// @Tags webhooks
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Param webhook_id path string true "Webhook ID"
// @Success 200 {object} APIResponse
// @Router /conversations/{conv_id}/webhooks/{webhook_id}/regenerate-key [post]
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
	wh.APIKeyPlain = apiKey
	if err := h.webhookDB.Update(r.Context(), wh); err != nil {
		logger.Error("update api_key_plain failed", "id", webhookID, "error", err)
	}
	JSON(w, map[string]string{"api_key": apiKey})
}

// ---------- receive message (public) ----------

type webhookReceiveReq struct {
	ContentType int    `json:"content_type"`
	Body        string `json:"body"`
	ReplyTo     int64  `json:"reply_to"`
}

// ReceiveMessage receives a message from an external webhook caller.
// @Summary Receive webhook message
// @Description Receives and processes a message sent by an external service via webhook.
// @Tags webhooks
// @Accept json
// @Success 200 {object} map[string]interface{}
// @Router /webhooks/receive [post]
func (h *WebhookHandler) ReceiveMessage(w http.ResponseWriter, r *http.Request) {
	ip := callerIP(r)
	if !h.rateLmt.Allow(ip) {
		writeJSONError(w, http.StatusTooManyRequests, 429, "too many requests")
		return
	}

	providedKey := extractBearerToken(r)
	if providedKey == "" {
		writeJSONError(w, http.StatusUnauthorized, 401, "missing api key")
		return
	}

	wh, err := h.webhookDB.GetByAPIKey(r.Context(), providedKey)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, 404, "not found")
		return
	}

	if !checkCIDR(wh.CIDRWhitelist, ip) {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: "ip not in whitelist"})
		return
	}

	var req webhookReceiveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, 400, "invalid json")
		return
	}
	if req.Body == "" {
		writeJSONError(w, http.StatusBadRequest, 400, "body is required")
		return
	}
	if len(req.Body) > 65536 {
		writeJSONError(w, http.StatusRequestEntityTooLarge, 413, "body too large")
		return
	}

	now := time.Now().UnixMilli()
	msgID := h.idGen.NextID()
	convSeq, err := h.seqCache.GetAndIncrementConvSeq(r.Context(), wh.ConvID)
	if err != nil {
		logger.Error("get conv seq failed", "conv_id", wh.ConvID, "error", err)
		writeJSONError(w, http.StatusInternalServerError, 500, "server error")
		return
	}

	senderID := fmt.Sprintf("webhook:%d", wh.ID)
	msg := &model.Message{
		MsgID:           msgID,
		ConvID:          wh.ConvID,
		SenderID:        senderID,
		SenderSessionID: fmt.Sprintf("wh:%d", now),
		SenderName:      wh.Name,
		ContentType:     model.ContentType(req.ContentType),
		Body:            req.Body,
		ReplyTo:         req.ReplyTo,
		Timestamp:       now,
		ClientSeq:       now,
		ConvSeq:         convSeq,
		Status:          model.MsgSent,
	}

	if err := h.store.Insert(r.Context(), msg); err != nil {
		logger.Error("persist webhook message failed", "error", err)
		http.Error(w, `{"code":500,"msg":"server error"}`, http.StatusInternalServerError)
		return
	}

	targets := h.router.Route(r.Context(), msg)
	if len(targets) > 0 {
		h.pusher.Push(r.Context(), msg, targets)
	}

	_ = h.seqCache.SetRecentMsg(r.Context(), wh.ConvID, msgID, float64(msgID))

	JSON(w, map[string]any{
		"msg_id":    msgID,
		"timestamp": now,
	})
}

// ---------- helpers ----------

func strconvParseInt(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %c", c)
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}

//nolint:unused
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

func ComputeSignature(secret []byte, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

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

func writeJSONError(w http.ResponseWriter, httpStatus, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(map[string]any{"code": code, "msg": msg, "data": nil})
}
