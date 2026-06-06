package model

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	UserIDPrefix      = "user_"
	SessionIDPrefix   = "sess_"
	GroupConvIDPrefix = "group_"
)

func GenerateUserID(snowflake func() int64) string {
	return fmt.Sprintf("%s%d", UserIDPrefix, snowflake())
}

func GenerateSessionID(snowflake func() int64) string {
	return fmt.Sprintf("%s%d", SessionIDPrefix, snowflake())
}

func GenerateGroupConvID(snowflake func() int64) string {
	return fmt.Sprintf("%s%d", GroupConvIDPrefix, snowflake())
}

func MakeP2PConvID(a, b string) string {
	ids := []string{a, b}
	sort.Strings(ids)
	return strings.Join(ids, ":")
}

func IsP2PConvID(convID string) bool {
	return strings.Contains(convID, ":") && !strings.HasPrefix(convID, GroupConvIDPrefix)
}

func IsGroupConvID(convID string) bool {
	return strings.HasPrefix(convID, GroupConvIDPrefix)
}

type Snowflake struct {
	mu         sync.Mutex
	lastTime   int64
	sequence   int64
	workerID   int64
	epoch      int64
}

func NewSnowflake(workerID int64, epoch time.Time) *Snowflake {
	return &Snowflake{
		workerID: workerID & 0x3FF,
		epoch:    epoch.UnixMilli(),
	}
}

func (s *Snowflake) NextID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli() - s.epoch
	if now < s.lastTime {
		now = s.lastTime
	}
	if now == s.lastTime {
		s.sequence = (s.sequence + 1) & 0xFFF
		if s.sequence == 0 {
			for now <= s.lastTime {
				now = time.Now().UnixMilli() - s.epoch
			}
		}
	} else {
		s.sequence = 0
	}
	s.lastTime = now
	return (now << 22) | (s.workerID << 12) | s.sequence
}
