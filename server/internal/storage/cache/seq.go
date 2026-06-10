package cache

import (
	"context"

	"github.com/go-redis/redis/v8"
)

type SeqCache struct {
	client *redis.Client
}

func NewSeqCache(client *redis.Client) *SeqCache {
	return &SeqCache{client: client}
}

func (c *SeqCache) GetAndIncrementConvSeq(ctx context.Context, convID string) (int64, error) {
	return c.client.Incr(ctx, "conv:seq:"+convID).Result()
}

func (c *SeqCache) SetUserSeq(ctx context.Context, userID, convID string, seq int64) error {
	return c.client.Set(ctx, "user:seq:"+userID+":"+convID, seq, 0).Err()
}

func (c *SeqCache) GetUserSeq(ctx context.Context, userID, convID string) (int64, error) {
	return c.client.Get(ctx, "user:seq:"+userID+":"+convID).Int64()
}

func (c *SeqCache) SetSessionSeq(ctx context.Context, sessionID, convID string, seq int64) error {
	return c.client.Set(ctx, "session:seq:"+sessionID+":"+convID, seq, 0).Err()
}

func (c *SeqCache) GetSessionSeq(ctx context.Context, sessionID, convID string) (int64, error) {
	return c.client.Get(ctx, "session:seq:"+sessionID+":"+convID).Int64()
}

func (c *SeqCache) MarkRead(ctx context.Context, userID, convID string, msgID int64) (int64, error) {
	pipe := c.client.Pipeline()
	// For simplicity, store user_seq as max read seq
	pipe.Set(ctx, "user:seq:"+userID+":"+convID, msgID, 0)
	// Get current conv seq
	convSeqCmd := pipe.Get(ctx, "conv:seq:"+convID)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	convSeq, _ := convSeqCmd.Int64()
	unread := convSeq - msgID
	if unread < 0 {
		unread = 0
	}
	return unread, nil
}

func (c *SeqCache) GetUnreadCount(ctx context.Context, userID, convID string) (int64, error) {
	userSeq, err := c.client.Get(ctx, "user:seq:"+userID+":"+convID).Int64()
	if err == redis.Nil {
		userSeq = 0
	} else if err != nil {
		return 0, err
	}
	convSeq, err := c.client.Get(ctx, "conv:seq:"+convID).Int64()
	if err == redis.Nil {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	unread := convSeq - userSeq
	if unread < 0 {
		return 0, nil
	}
	return unread, nil
}

func (c *SeqCache) SetRecentMsg(ctx context.Context, convID string, msgID int64, score float64) error {
	pipe := c.client.Pipeline()
	pipe.ZAdd(ctx, "conv:recent:"+convID, &redis.Z{Score: score, Member: msgID})
	pipe.ZRemRangeByRank(ctx, "conv:recent:"+convID, 0, -101)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *SeqCache) InitConvSeq(ctx context.Context, convID string, seq int64) error {
	ok, err := c.client.SetNX(ctx, "conv:seq:"+convID, seq, 0).Result()
	if err != nil {
		return err
	}
	if !ok {
		// already exists, only initialize if current is lower
		_, err = c.client.Eval(ctx,
			`local cur = redis.call("GET", KEYS[1])
			 if cur and tonumber(cur) < tonumber(ARGV[1]) then
			   return redis.call("SET", KEYS[1], ARGV[1])
			 end
			 return nil`, []string{"conv:seq:" + convID}, seq).Result()
		if err == redis.Nil {
			err = nil // existing seq is higher or equal, not an error
		}
	}
	return err
}

func (c *SeqCache) RecoverConvSeq(ctx context.Context, convID string, fromDBSeq int64) error {
	return c.InitConvSeq(ctx, convID, fromDBSeq)
}
