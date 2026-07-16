package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// GuestTokenManager mints and validates opaque, HMAC-signed guest cart tokens.
type GuestTokenManager struct {
	secret []byte
}

// NewGuestTokenManager creates a manager with the given signing secret.
func NewGuestTokenManager(secret string) *GuestTokenManager {
	return &GuestTokenManager{secret: []byte(secret)}
}

// NewGuestID returns a new opaque guest identity, e.g. "guest-<uuid>".
func (m *GuestTokenManager) NewGuestID() string {
	return "guest-" + uuid.New().String()
}

// Sign returns "<guestID>.<base64url(hmac)>".
func (m *GuestTokenManager) Sign(guestID string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(guestID))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return guestID + "." + sig
}

// Verify parses a token and returns the guestID only if the signature is valid.
func (m *GuestTokenManager) Verify(token string) (string, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("malformed guest token")
	}
	guestID := parts[0]
	if !strings.HasPrefix(guestID, "guest-") {
		return "", fmt.Errorf("invalid guest id")
	}
	expected := m.Sign(guestID)
	if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
		return "", fmt.Errorf("invalid guest token signature")
	}
	return guestID, nil
}
