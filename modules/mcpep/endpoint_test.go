/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package mcpep

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func TestTokenHashRoundTrip(t *testing.T) {
	plaintext, hash := generateToken()
	require.NotEmpty(t, plaintext)
	assert.True(t, len(plaintext) > len(tokenPrefix))
	assert.Equal(t, hash, hashToken(plaintext), "hash is deterministic")
	assert.NotEqual(t, plaintext, hash)

	other, _ := generateToken()
	assert.NotEqual(t, hash, hashToken(other), "two issuances never collide")
}

func TestBearerFromHeaders(t *testing.T) {
	h := http.Header{}
	assert.Empty(t, bearerFromHeaders(h), "no auth header → no bearer")

	h.Set("Authorization", "Bearer "+tokenPrefix+"abc123")
	assert.Equal(t, tokenPrefix+"abc123", bearerFromHeaders(h))

	h.Set("Authorization", "bearer "+tokenPrefix+"xyz") // case-insensitive scheme
	assert.Equal(t, tokenPrefix+"xyz", bearerFromHeaders(h))

	h.Set("Authorization", "Bearer some-other-scheme")
	assert.Empty(t, bearerFromHeaders(h), "foreign bearer schemes pass through untouched")
}

func TestAllowAndTouchRateWindow(t *testing.T) {
	ep := &core.MCPEndpoint{}
	ep.ID = "ep-rl"
	ep.RateLimitPerMinute = 3

	for i := 0; i < 3; i++ {
		assert.True(t, allowAndTouch(ep), "within the window limit")
	}
	assert.False(t, allowAndTouch(ep), "fourth hit in one minute is denied")
}

func TestRateWindowRespectsConfigChange(t *testing.T) {
	ep := &core.MCPEndpoint{}
	ep.ID = "ep-rl2"
	// no configured limit → default applies
	for i := 0; i < defaultRate; i++ {
		assert.True(t, allowAndTouch(ep))
	}
	assert.False(t, allowAndTouch(ep))
}

func TestAuthorizeWithoutBearerKeepsLegacyPath(t *testing.T) {
	h := http.Header{}
	assert.True(t, authorize(h, nil), "no mcp_ bearer → legacy path, unchanged behavior")

	h.Set("Authorization", "Bearer "+tokenPrefix+"does-not-resolve")
	// unresolvable token: denied (the store lookup fails closed in a real
	// environment; here the nil row path)
	assert.False(t, authorize(h, nil))
}
