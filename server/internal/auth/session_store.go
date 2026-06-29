package auth

import "sync"
import "time"

// Simple in-memory store for MFA session tokens and signup codes
var (
	storeMu   sync.RWMutex
	codeStore = map[string]codeEntry{}
)

type codeEntry struct {
	UserID  string
	Expires time.Time
}

// SetSignupCode stores a code mapped to a user with TTL in seconds.
func SetSignupCode(code, userID string, ttlSeconds int) {
	storeMu.Lock()
	defer storeMu.Unlock()
	codeStore[code] = codeEntry{
		UserID:  userID,
		Expires: time.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}
}

// GetSignupCode returns the user ID for a code, or empty string if not found/expired.
func GetSignupCode(code string) string {
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
	storeMu.Lock()
	defer storeMu.Unlock()
	delete(codeStore, code)
}
