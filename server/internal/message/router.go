package message

import (
	"context"

	"siciv.space/agent/panda_ai/pkg/model"
	"siciv.space/agent/panda_ai/pkg/logger"
)

type RouteTarget struct {
	UserID    string
	SessionIDs []string
}

type Router struct {
	sessManager  sessionGetter
	convManager  convProvider
	gateway      connBySessionID
}

type sessionGetter interface {
	GetUserSessionIDs(ctx context.Context, userID string) []string
}

type convProvider interface {
	GetMembers(ctx context.Context, convID string) ([]*model.ConvMember, error)
	IsMember(ctx context.Context, convID, userID string) (bool, error)
	Get(ctx context.Context, convID string) (*model.Conversation, error)
}

type connBySessionID interface {
	GetBySessionID(ctx context.Context, sessionID string) interface{}
}

func NewRouter(sessManager sessionGetter, convManager convProvider, gateway connBySessionID) *Router {
	return &Router{
		sessManager: sessManager,
		convManager: convManager,
		gateway:     gateway,
	}
}

func (r *Router) Route(ctx context.Context, msg *model.Message) []RouteTarget {
	conv, err := r.convManager.Get(ctx, msg.ConvID)
	if err != nil {
		logger.Error("route: conv not found", "conv_id", msg.ConvID, "error", err)
		return nil
	}

	if conv.Type == model.ConvP2P {
		return r.routeP2P(ctx, msg, conv)
	}
	return r.routeGroup(ctx, msg, conv)
}

func (r *Router) route(ctx context.Context, msg *model.Message, members []*model.ConvMember) []RouteTarget {
	var targets []RouteTarget
	for _, m := range members {
		sessionIDs := r.sessManager.GetUserSessionIDs(ctx, m.UserID)
		if len(sessionIDs) == 0 {
			continue
		}
		if m.UserID == msg.SenderID {
			// Same user on other devices should still receive the push.
			// Only filter out the sending session to avoid echoing back.
			filtered := make([]string, 0, len(sessionIDs))
			for _, sid := range sessionIDs {
				if sid != msg.SenderSessionID {
					filtered = append(filtered, sid)
				}
			}
			sessionIDs = filtered
		}
		if len(sessionIDs) > 0 {
			targets = append(targets, RouteTarget{
				UserID:     m.UserID,
				SessionIDs: sessionIDs,
			})
		}
	}
	return targets
}

func (r *Router) routeP2P(ctx context.Context, msg *model.Message, conv *model.Conversation) []RouteTarget {
	members, err := r.convManager.GetMembers(ctx, conv.ConvID)
	if err != nil {
		logger.Error("routeP2P: get members failed", "conv_id", conv.ConvID, "error", err)
		return nil
	}
	return r.route(ctx, msg, members)
}

func (r *Router) routeGroup(ctx context.Context, msg *model.Message, conv *model.Conversation) []RouteTarget {
	members, err := r.convManager.GetMembers(ctx, conv.ConvID)
	if err != nil {
		logger.Error("routeGroup: get members failed", "conv_id", conv.ConvID, "error", err)
		return nil
	}
	return r.route(ctx, msg, members)
}
