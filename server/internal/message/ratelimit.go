package message

import (
	"context"
	"sync"
	"time"

	"github.com/dolphinz/im-server/pkg/model"
)

type RateLimiter struct {
	mu          sync.Mutex
	userBuckets map[string]*bucket
	msgPerSec   int
	burstSize   int
	maxBodyBytes int
}

type bucket struct {
	tokens    int
	lastCheck time.Time
}

func NewRateLimiter(msgPerSec, burstSize, maxBodyBytes int) *RateLimiter {
	return &RateLimiter{
		userBuckets:  make(map[string]*bucket),
		msgPerSec:    msgPerSec,
		burstSize:    burstSize,
		maxBodyBytes: maxBodyBytes,
	}
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
