package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuestTokenManagerSignVerifyRoundTrip(t *testing.T) {
	manager := NewGuestTokenManager("secret-one")
	guestID := manager.NewGuestID()

	token := manager.Sign(guestID)
	actualGuestID, err := manager.Verify(token)

	require.NoError(t, err)
	assert.Equal(t, guestID, actualGuestID)
}

func TestGuestTokenManagerVerifyRejectsTamperedSignature(t *testing.T) {
	manager := NewGuestTokenManager("secret-one")
	token := manager.Sign("guest-123")
	tampered := token[:len(token)-1] + "A"

	_, err := manager.Verify(tampered)

	assert.Error(t, err)
}

func TestGuestTokenManagerVerifyRejectsDifferentSecret(t *testing.T) {
	first := NewGuestTokenManager("secret-one")
	second := NewGuestTokenManager("secret-two")
	token := first.Sign("guest-123")

	_, err := second.Verify(token)

	assert.Error(t, err)
}

func TestGuestTokenManagerVerifyRejectsNonGuestID(t *testing.T) {
	manager := NewGuestTokenManager("secret-one")
	token := manager.Sign("user-123")

	_, err := manager.Verify(token)

	assert.Error(t, err)
}
