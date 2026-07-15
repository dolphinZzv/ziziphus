package message

import (
	"context"
	"sync"
	"time"

	"ziziphus/pkg/model"
)

type RateLimiter struct {
	mu              sync.Mutex
	userBuckets     map[string]*bucket
	msgPerSec       int
	burstSize       int
	maxBodyBytes    int
	cleanupInterval time.Duration
	stopped         chan struct{}
}

type bucket struct {
	tokens    int
	lastCheck time.Time
}

func NewRateLimiter(msgPerSec, burstSize, maxBodyBytes int) *RateLimiter {
	rl := &RateLimiter{
		userBuckets:     make(map[string]*bucket),
		msgPerSec:       msgPerSec,
		burstSize:       burstSize,
		maxBodyBytes:    maxBodyBytes,
		cleanupInterval: 5 * time.Minute,
		stopped:         make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) Check(ctx context.Context, userID string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.userBuckets[userID]
	if !ok {
		b = &bucket{tokens: rl.burstSize, lastCheck: time.Now()}
		rl.userBuckets[userID] = b
	}

	now := time.Now()
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens = b.tokens + int(elapsed*float64(rl.msgPerSec))
	if b.tokens > rl.burstSize {
		b.tokens = rl.burstSize
	}
	b.lastCheck = now

	if b.tokens <= 0 {
		return model.ErrRateLimited
	}
	b.tokens--
	return nil
}

func (rl *RateLimiter) CheckBodySize(body string) error {
	if rl.maxBodyBytes > 0 && len(body) > rl.maxBodyBytes {
		return model.ErrMsgTooLarge
	}
	return nil
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopped:
			return
		}
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	threshold := time.Now().Add(-rl.cleanupInterval * 2)
	for userID, b := range rl.userBuckets {
		if b.lastCheck.Before(threshold) {
			delete(rl.userBuckets, userID)
		}
	}
}
