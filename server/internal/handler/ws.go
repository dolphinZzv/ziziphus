package handler

import (
	"fmt"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"ziziphus/internal/auth"
	"ziziphus/internal/gateway"
	"ziziphus/pkg/i18n"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
	"ziziphus/pkg/protocol"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // non-browser clients
		}
		// Validate Origin matches the Host to prevent CSWSH attacks
		originHost := extractHost(origin)
		host := extractHost(r.Host)
		if originHost == host {
			return true
		}
		logger.Warn("ws origin mismatch", "origin", origin, "host", host)
		return false
	}}

var wsTracer = otel.Tracer("ziziphus.ws")

type msgEditor interface {
	Get(ctx context.Context, msgID int64) (*model.Message, error)
	UpdateBody(ctx context.Context, msgID int64, newBody string) error
	Recall(ctx context.Context, msgID int64) error
}

type WSHandler struct {
	authMW  func(ctx context.Context, token string) (context.Context, error)
	sessMgr sessionManager
	connMgr *gateway.Manager
	ingest  messageIngester
	sync    syncHandler
	receipt readReceiptHandler
	msgRepo msgEditor
}

type sessionManager interface {
	Create(ctx context.Context, userID string, device model.DeviceType, deviceName string, clientIP string, deviceID string) (*model.Session, error)
	Get(ctx context.Context, sessionID string) *model.Session
	GetUserSessionIDs(ctx context.Context, userID string) []string
	Delete(ctx context.Context, sessionID string) error
	BindConnection(ctx context.Context, sessionID, connID string) error
}

type messageIngester interface {
	Ingest(ctx context.Context, senderID, sessionID string, payload protocol.MsgSendPayload) (*protocol.MsgSendAckPayload, error)
}

type syncHandler interface {
	Handle(ctx context.Context, sessionID string, req protocol.SyncReqPayload) (*protocol.SyncResPayload, error)
}

type readReceiptHandler interface {
	MarkRead(ctx context.Context, userID, convID string, msgID int64) error
}

func NewWSHandler(
	authMW func(ctx context.Context, token string) (context.Context, error),
	sessMgr sessionManager,
	connMgr *gateway.Manager,
	ingest messageIngester,
	sync syncHandler,
	receipt readReceiptHandler,
	msgRepo msgEditor,
) *WSHandler {
	return &WSHandler{
		authMW:  authMW,
		sessMgr: sessMgr,
		connMgr: connMgr,
		ingest:  ingest,
		sync:    sync,
		receipt: receipt,
		msgRepo: msgRepo,
	}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("ws upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	token := r.URL.Query().Get("token")
	platform := r.URL.Query().Get("platform")

	deviceType := model.DeviceDesktop
	deviceName := "macOS"
	switch platform {
	case "ios":
		deviceType = model.DevicePhone
		deviceName = "iOS"
	case "ipados":
		deviceType = model.DeviceTablet
		deviceName = "iPadOS"
	case "web":
		deviceType = model.DeviceWeb
		deviceName = "Web"
	case "android":
		deviceType = model.DevicePhone
		deviceName = "Android"
	case "windows":
		deviceType = model.DeviceDesktop
		deviceName = "Windows"
	}

	deviceID := r.URL.Query().Get("device_id")

	// extract client IP
	clientIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(clientIP); err == nil {
		clientIP = host
	}

	ctx, err := h.authMW(r.Context(), token)
	if err != nil {
		logger.Warn("ws auth failed", "error", err)
		writeWSError(conn, model.ErrNoPermission, i18n.T(r.Context(), "err.unauthorized"))
		return
	}
	userID := auth.UserFromCtx(ctx)

	// dedup: remove any existing session with the same device_id
	if deviceID != "" {
		for _, sid := range h.sessMgr.GetUserSessionIDs(ctx, userID) {
			existing := h.sessMgr.Get(ctx, sid)
			if existing != nil && existing.DeviceID == deviceID {
				h.connMgr.DisconnectBySessionID(ctx, sid)
				_ = h.sessMgr.Delete(ctx, sid)
			}
		}
	}

	sess, err := h.sessMgr.Create(ctx, userID, deviceType, deviceName, clientIP, deviceID)
	if err != nil {
		logger.Error("ws session create failed", "user_id", userID, "error", err)
		writeWSError(conn, model.ErrInternal, i18n.T(r.Context(), "err.create_session_failed"))
		return
	}

	connID := "conn_" + uuid.New().String()[:8]
	gwConn := gateway.NewConnection(connID, userID, sess.SessionID, int(deviceType), conn)

	h.connMgr.Add(ctx, gwConn)
	_ = h.sessMgr.BindConnection(ctx, sess.SessionID, connID)

	logger.Info("ws connected", "user_id", userID, "session_id", sess.SessionID, "conn_id", connID)

	// Start a span for the WebSocket connection lifecycle
	ctx, wsSpan := wsTracer.Start(ctx, "ws.connect",
		trace.WithAttributes(
			attribute.String("user_id", userID),
			attribute.String("session_id", sess.SessionID),
			attribute.String("conn_id", connID),
			attribute.Int("device_type", int(deviceType)),
		),
	)

	// notify other online users
	h.broadcastSessionEvent(ctx, userID, sess.SessionID, int(deviceType), protocol.SessionOnline)

	// send welcome frame so the client can transition out of .connecting
	welcome := protocol.SessionRecoverAckPayload{
		SessionID: sess.SessionID,
		UserID:    userID,
		Timestamp: time.Now().UnixMilli(),
	}
	welcomeData, _ := json.Marshal(welcome)
	_ = gwConn.SendFrame(protocol.Frame{Type: protocol.SessionRecoverAck, Payload: welcomeData})

	// read loop (blocks until disconnect)
	h.readLoop(ctx, gwConn, userID, sess.SessionID)

	wsSpan.End()

	// cleanup on disconnect
	logger.Info("ws disconnected", "user_id", userID, "session_id", sess.SessionID, "conn_id", connID)
	h.broadcastSessionEvent(context.Background(), userID, sess.SessionID, int(deviceType), protocol.SessionOffline)
	h.connMgr.Remove(context.Background(), connID)
}

func (h *WSHandler) readLoop(ctx context.Context, gwConn *gateway.Connection, userID, sessionID string) {
	for {
		frame, err := gwConn.ReadFrame()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				logger.Info("ws read loop ended", "user_id", userID, "session_id", sessionID)
			} else {
				logger.Warn("ws read error", "user_id", userID, "session_id", sessionID, "error", err)
			}
			return
		}

		if err := h.dispatch(ctx, userID, sessionID, frame, gwConn); err != nil {
			logger.Warn("ws dispatch error", "user_id", userID, "session_id", sessionID, "type", frame.Type, "error", err)
		}
	}
}

func (h *WSHandler) dispatch(ctx context.Context, userID, sessionID string, frame protocol.Frame, conn *gateway.Connection) error {
	// Create a child span scoped to this frame dispatch
	ctx, span := wsTracer.Start(ctx, "ws."+fmt.Sprint(frame.Type),
		trace.WithAttributes(
			attribute.String("user_id", userID),
			attribute.String("session_id", sessionID),
			attribute.Int("frame_type", int(frame.Type)),
			attribute.String("frame_id", frame.ID),
		),
	)
	defer span.End()

	// Log the OTel trace ID so operators can correlate this message across
	// all downstream operations (DB queries, Redis, pushes, etc.) by searching
	// the trace ID in the observability backend.
	sc := span.SpanContext()
	if sc.HasTraceID() {
		logger.Debug("ws.dispatch",
			"trace_id", sc.TraceID().String(),
			"user_id", userID,
			"session_id", sessionID,
			"frame_type", frame.Type,
			"frame_id", frame.ID,
		)
	}

	// Use a 30-second timeout context derived from the trace context,
	// so spans are properly propagated through all downstream operations.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	switch frame.Type {
	case protocol.Ping:
		return conn.SendFrame(protocol.Frame{Type: protocol.Pong})

	case protocol.MsgSend:
		var payload protocol.MsgSendPayload
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			return err
		}
		ack, err := h.ingest.Ingest(ctx, userID, sessionID, payload)
		if err != nil {
			errCode := model.ErrInternal
			if appErr, ok := err.(*model.AppError); ok {
				errCode = appErr.Code
			}
			errPayload, _ := json.Marshal(protocol.ErrorPayload{Code: errCode, Message: err.Error()})
			return conn.SendFrame(protocol.Frame{Type: protocol.Error, ID: frame.ID, Payload: errPayload})
		}
		ackData, _ := json.Marshal(ack)
		return conn.SendFrame(protocol.Frame{Type: protocol.MsgSendAck, ID: frame.ID, Payload: ackData})

	case protocol.SyncReq:
		var payload protocol.SyncReqPayload
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			return err
		}
		res, err := h.sync.Handle(ctx, sessionID, payload)
		if err != nil {
			errCode := model.ErrInternal
			if appErr, ok := err.(*model.AppError); ok {
				errCode = appErr.Code
			}
			errPayload, _ := json.Marshal(protocol.ErrorPayload{Code: errCode, Message: err.Error()})
			return conn.SendFrame(protocol.Frame{Type: protocol.Error, ID: frame.ID, Payload: errPayload})
		}
		resData, _ := json.Marshal(res)
		return conn.SendFrame(protocol.Frame{Type: protocol.SyncRes, ID: frame.ID, Payload: resData})

	case protocol.MsgReadNotify:
		var payload protocol.MsgReadNotifyPayload
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			return err
		}
		return h.receipt.MarkRead(ctx, userID, payload.ConvID, payload.MsgID)

	case protocol.MsgEdit:
		var p protocol.MsgEditPayload
		if err := json.Unmarshal(frame.Payload, &p); err != nil {
			return err
		}
		msg, err := h.msgRepo.Get(ctx, p.MsgID)
		if err != nil || msg.SenderID != userID {
			return nil
		}
		if err := h.msgRepo.UpdateBody(ctx, p.MsgID, p.NewBody); err != nil {
			return err
		}
		now := time.Now().UnixMilli()
		push, _ := json.Marshal(protocol.MsgEditPushPayload{
			ConvID: p.ConvID, MsgID: p.MsgID, SenderID: userID,
			NewBody: p.NewBody, EditedAt: now, Timestamp: now,
		})
		for _, c := range h.connMgr.All() {
			_ = c.SendFrame(protocol.Frame{Type: protocol.MsgEdit, Payload: push})
		}

	case protocol.MsgRecall:
		var p protocol.MsgRecallPayload
		if err := json.Unmarshal(frame.Payload, &p); err != nil {
			return err
		}
		m, err := h.msgRepo.Get(ctx, p.MsgID)
		if err != nil || m.SenderID != userID {
			return nil
		}
		if err := h.msgRepo.Recall(ctx, p.MsgID); err != nil {
			return err
		}
		now := time.Now().UnixMilli()
		push, _ := json.Marshal(protocol.MsgRecallPushPayload{
			ConvID: p.ConvID, MsgID: p.MsgID, SenderID: userID,
			RecalledAt: now, Timestamp: now,
		})
		for _, c := range h.connMgr.All() {
			_ = c.SendFrame(protocol.Frame{Type: protocol.MsgRecall, Payload: push})
		}

	case protocol.Typing:
		var payload protocol.TypingPayload
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			return err
		}

	case protocol.SessionRecover:
		var payload protocol.SessionRecoverPayload
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			return err
		}
		recoveredID := sessionID
		existingSess := h.sessMgr.Get(ctx, payload.SessionID)
		if existingSess != nil && existingSess.UserID == userID {
			recoveredID = payload.SessionID
			if err := h.sessMgr.BindConnection(ctx, payload.SessionID, conn.ConnID); err != nil {
				recoveredID = sessionID
			}
		}
		ack := protocol.SessionRecoverAckPayload{
			SessionID: recoveredID,
			UserID:    userID,
			Timestamp: time.Now().UnixMilli(),
		}
		ackData, _ := json.Marshal(ack)
		return conn.SendFrame(protocol.Frame{Type: protocol.SessionRecoverAck, ID: frame.ID, Payload: ackData})

	default:
		logger.Debug("unknown frame type", "type", frame.Type)
	}
	return nil
}

func (h *WSHandler) broadcastSessionEvent(ctx context.Context, userID, sessionID string, device int, eventType protocol.MessageType) {
	payload := protocol.SessionEventPayload{
		UserID:    userID,
		SessionID: sessionID,
		Device:    device,
	}
	data, _ := json.Marshal(payload)
	frame := protocol.Frame{
		Type:    eventType,
		Payload: data,
	}

	for _, c := range h.connMgr.All() {
		if c.UserID != userID {
			_ = c.SendFrame(frame)
		}
	}
}

func writeWSError(conn *websocket.Conn, code int, msg string) {
	payload, _ := json.Marshal(protocol.ErrorPayload{Code: code, Message: msg})
	_ = conn.WriteJSON(protocol.Frame{Type: protocol.Error, Payload: payload})
}

func extractHost(origin string) string {
	if strings.HasPrefix(origin, "https://") {
		origin = origin[8:]
	} else if strings.HasPrefix(origin, "http://") {
		origin = origin[7:]
	}
	if idx := strings.Index(origin, ":"); idx >= 0 {
		return origin[:idx]
	}
	return origin
}
