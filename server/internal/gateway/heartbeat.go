package gateway

import (
	"context"
	"sync/atomic"
	"time"

	"ziziphus/pkg/logger"
)

type HeartbeatConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

func DefaultHeartbeatConfig() HeartbeatConfig {
	return HeartbeatConfig{
		Interval: 30 * time.Second,
		Timeout:  90 * time.Second,
	}
}

type Heartbeat struct {
	manager *Manager
	config  HeartbeatConfig
	stopped atomic.Bool
}

func NewHeartbeat(manager *Manager, config HeartbeatConfig) *Heartbeat {
	return &Heartbeat{
		manager: manager,
		config:  config,
	}
}

func (h *Heartbeat) Start(ctx context.Context, onTimeout func(ctx context.Context, connID string)) {
	logger.Info("heartbeat started", "interval", h.config.Interval, "timeout", h.config.Timeout)
	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.checkTimeouts(ctx, onTimeout)
		}
	}
}

func (h *Heartbeat) checkTimeouts(ctx context.Context, onTimeout func(ctx context.Context, connID string)) {
	now := time.Now().UnixMilli()
	for _, conn := range h.manager.All() {
		lastHB := atomic.LoadInt64(&conn.LastHeartbeat)
		if now-lastHB > h.config.Timeout.Milliseconds() {
			logger.Warn("heartbeat timeout", "conn_id", conn.ConnID, "user_id", conn.UserID)
			conn.Close()
			if onTimeout != nil {
				onTimeout(ctx, conn.ConnID)
			}
		}
	}
}

func (h *Heartbeat) Stop() {
	h.stopped.Store(true)
}
