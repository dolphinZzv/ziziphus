package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/dolphinz/im-server/pkg/model"
)

const sessionTTL = 24 * time.Hour

type SessionCache struct {
	client *redis.Client
}

func NewSessionCache(client *redis.Client) *SessionCache {
	return &SessionCache{client: client}
}

func (c *SessionCache) Set(ctx context.Context, s *model.Session) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	pipe := c.client.Pipeline()
	pipe.Set(ctx, "session:"+s.SessionID, string(data), sessionTTL)
	pipe.SAdd(ctx, "user:sessions:"+s.UserID, s.SessionID)
	_, err = pipe.Exec(ctx)
	return err
}

func (c *SessionCache) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	data, err := c.client.Get(ctx, "session:"+sessionID).Bytes()
	if err != nil {
		return nil, err
	}
	s := &model.Session{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (c *SessionCache) Delete(ctx context.Context, sessionID, userID string) error {
	pipe := c.client.Pipeline()
	pipe.Del(ctx, "session:"+sessionID)
	pipe.SRem(ctx, "user:sessions:"+userID, sessionID)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *SessionCache) GetUserSessionIDs(ctx context.Context, userID string) ([]string, error) {
	return c.client.SMembers(ctx, "user:sessions:"+userID).Result()
}

func (c *SessionCache) SetConnSession(ctx context.Context, connID, sessionID string) error {
	return c.client.Set(ctx, "conn:session:"+connID, sessionID, sessionTTL).Err()
}

func (c *SessionCache) GetConnSession(ctx context.Context, connID string) (string, error) {
	return c.client.Get(ctx, "conn:session:"+connID).Result()
}

func (c *SessionCache) DelConnSession(ctx context.Context, connID string) error {
	return c.client.Del(ctx, "conn:session:"+connID).Err()
}
