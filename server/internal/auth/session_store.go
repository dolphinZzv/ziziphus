package auth

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// Simple in-memory store for MFA session tokens and signup codes
var (
	storeMu   sync.RWMutex
	codeStore = map[string]codeEntry{}
	rdbClient *redis.Client
	rdbMu     sync.RWMutex
)

type codeEntry struct {
	UserID  string
	Expires time.Time
}

// InitSignupCodeStore sets the optional Redis client for distributed
// signup code storage. When set, codes are stored in Redis first; the
// in-memory fallback is only used when Redis is nil.
func InitSignupCodeStore(rdb *redis.Client) {
	rdbMu.Lock()
	defer rdbMu.Unlock()
	rdbClient = rdb
}

// SetSignupCode stores a code mapped to a user with TTL in seconds.
func SetSignupCode(code, userID string, ttlSeconds int) {
	rdbMu.RLock()
	rdb := rdbClient
	rdbMu.RUnlock()
	if rdb != nil {
		data, _ := json.Marshal(codeEntry{
			UserID:  userID,
			Expires: time.Now().Add(time.Duration(ttlSeconds) * time.Second),
		})
		rdb.Set(context.Background(), "mfa_code:"+code, data, time.Duration(ttlSeconds)*time.Second)
		return
	}
	storeMu.Lock()
	defer storeMu.Unlock()
	codeStore[code] = codeEntry{
		UserID:  userID,
		Expires: time.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}
}

// GetSignupCode returns the user ID for a code, or empty string if not found/expired.
func GetSignupCode(code string) string {
	rdbMu.RLock()
	rdb := rdbClient
	rdbMu.RUnlock()
	if rdb != nil {
		data, err := rdb.GetDel(context.Background(), "mfa_code:"+code).Bytes()
		if err != nil {
			return ""
		}
		var entry codeEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return ""
		}
		if time.Now().After(entry.Expires) {
			return ""
		}
		return entry.UserID
	}
	storeMu.RLock()
	entry, ok := codeStore[code]
	storeMu.RUnlock()
	if !ok {
		return ""
	}
	if time.Now().After(entry.Expires) {
		storeMu.Lock()
		delete(codeStore, code)
		storeMu.Unlock()
		return ""
	}
	return entry.UserID
}

// ClearSignupCode removes a code from the store.
func ClearSignupCode(code string) {
	rdbMu.RLock()
	rdb := rdbClient
	rdbMu.RUnlock()
	if rdb != nil {
		rdb.Del(context.Background(), "mfa_code:"+code)
		return
	}
	storeMu.Lock()
	defer storeMu.Unlock()
	delete(codeStore, code)
}
