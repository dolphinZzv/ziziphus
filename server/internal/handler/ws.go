package handler

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"siciv.space/agent/panda_ai/internal/auth"
	"siciv.space/agent/panda_ai/internal/gateway"
	"siciv.space/agent/panda_ai/pkg/i18n"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
	"siciv.space/agent/panda_ai/pkg/protocol"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:   4096,
	WriteBufferSize:  4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	authMW   func(ctx context.Context, token string) (context.Context, error)
	sessMgr  sessionManager
	connMgr  *gateway.Manager
	ingest   messageIngester
	sync     syncHandler
	receipt  readReceiptHandler
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
) *WSHandler {
	return &WSHandler{
		authMW:  authMW,
		sessMgr: sessMgr,
		connMgr: connMgr,
		ingest:  ingest,
		sync:    sync,
		receipt: receipt,
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
				h.sessMgr.Delete(ctx, sid)
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
	h.sessMgr.BindConnection(ctx, sess.SessionID, connID)

	logger.Info("ws connected", "user_id", userID, "session_id", sess.SessionID, "conn_id", connID)

	// notify other online users
	h.broadcastSessionEvent(ctx, userID, sess.SessionID, int(deviceType), protocol.SessionOnline)

	// send welcome frame so the client can transition out of .connecting
	welcome := protocol.SessionRecoverAckPayload{
		SessionID: sess.SessionID,
		UserID:    userID,
		Timestamp: time.Now().UnixMilli(),
	}
	welcomeData, _ := json.Marshal(welcome)
	gwConn.SendFrame(protocol.Frame{Type: protocol.SessionRecoverAck, Payload: welcomeData})

	// read loop (blocks until disconnect)
	h.readLoop(ctx, gwConn, userID, sess.SessionID)

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

		if err := h.dispatch(userID, sessionID, frame, gwConn); err != nil {
			logger.Warn("ws dispatch error", "user_id", userID, "session_id", sessionID, "type", frame.Type, "error", err)
		}
	}
}

func (h *WSHandler) dispatch(userID, sessionID string, frame protocol.Frame, conn *gateway.Connection) error {
	switch frame.Type {
	case protocol.Ping:
		return conn.SendFrame(protocol.Frame{Type: protocol.Pong})

	case protocol.MsgSend:
		var payload protocol.MsgSendPayload
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			return err
		}
		ack, err := h.ingest.Ingest(context.Background(), userID, sessionID, payload)
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
		res, err := h.sync.Handle(context.Background(), sessionID, payload)
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
		return h.receipt.MarkRead(context.Background(), userID, payload.ConvID, payload.MsgID)

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
		existingSess := h.sessMgr.Get(context.Background(), payload.SessionID)
		if existingSess != nil && existingSess.UserID == userID {
			recoveredID = payload.SessionID
			if err := h.sessMgr.BindConnection(context.Background(), payload.SessionID, conn.ConnID); err != nil {
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
			c.SendFrame(frame)
		}
	}
}

func writeWSError(conn *websocket.Conn, code int, msg string) {
	payload, _ := json.Marshal(protocol.ErrorPayload{Code: code, Message: msg})
	conn.WriteJSON(protocol.Frame{Type: protocol.Error, Payload: payload})
}
