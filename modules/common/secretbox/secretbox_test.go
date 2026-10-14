/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package secretbox

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/framework/core/util"
)

// withKey installs a master key for the test and restores the
// no-key state after (test seam: ResetMasterKey re-reads the env).
func withKey(t *testing.T, key string) {
	t.Helper()
	t.Setenv(MasterKeyEnv, key)
	ResetMasterKey()
	t.Cleanup(ResetMasterKey)
}

func TestRoundTripHexAndPassphrase(t *testing.T) {
	for _, key := range []string{
		strings.Repeat("ab", 32), // 64-char hex → exact bytes
		"correct horse battery staple",
	} {
		withKey(t, key)
		require.True(t, Enabled())

		sealed := Encrypt("sk-live-9f2c")
		require.True(t, strings.HasPrefix(sealed, Prefix), "ciphertext must carry the %s prefix", Prefix)
		assert.NotContains(t, sealed, "sk-live-9f2c")

		assert.Equal(t, "sk-live-9f2c", Decrypt(sealed))
	}
}

func TestEncryptIdempotentAndEmptyPassthrough(t *testing.T) {
	withKey(t, "k1")
	sealed := Encrypt("secret-value")
	assert.Equal(t, sealed, Encrypt(sealed), "double-encrypt must be a no-op")
	assert.Equal(t, "", Encrypt(""), "empty stays empty")
}

func TestDisabledModeIsPlaintextCompat(t *testing.T) {
	withKey(t, "")
	require.False(t, Enabled())
	assert.Equal(t, "plain-value", Encrypt("plain-value"), "no key → writes stay plaintext")
	assert.Equal(t, "legacy-plaintext", Decrypt("legacy-plaintext"), "legacy plaintext passes through")
}

func TestDecryptWrongKeyYieldsEmpty(t *testing.T) {
	withKey(t, "first-key")
	sealed := Encrypt("the-credential")

	withKey(t, "second-key")
	assert.Equal(t, "", Decrypt(sealed), "lost/rotated key must yield empty, never the ciphertext")
}

func TestDecryptTamperedYieldsEmpty(t *testing.T) {
	withKey(t, "k")
	sealed := Encrypt("the-credential")
	tampered := sealed[:len(sealed)-2] + "A="
	assert.Equal(t, "", Decrypt(tampered))
	assert.Equal(t, "", Decrypt(Prefix+"not-base64!!!"))
	assert.Equal(t, "", Decrypt(Prefix+"AAA"))
}

func TestConfigWalkEncryptsSensitiveKeysOnly(t *testing.T) {
	withKey(t, "k")
	cfg := map[string]interface{}{
		"api_key":      "sk-123",
		"url":          "http://127.0.0.1:9200",
		"page_size":    100,
		"github_token": "gh-token",
		"tokenizer":    "ik", // similar-looking, must stay untouched
		"nested": map[string]interface{}{
			"password": "hunter2",
			"name":     "inner",
		},
		"list": []interface{}{map[string]interface{}{"client_secret": "cs-1"}},
	}

	out := EncryptConfig(cfg).(map[string]interface{})
	assert.True(t, strings.HasPrefix(out["api_key"].(string), Prefix))
	assert.Equal(t, "http://127.0.0.1:9200", out["url"])
	assert.Equal(t, 100, out["page_size"])
	assert.True(t, strings.HasPrefix(out["github_token"].(string), Prefix), "suffixed credential keys must match")
	assert.Equal(t, "ik", out["tokenizer"])

	nested := out["nested"].(map[string]interface{})
	assert.True(t, strings.HasPrefix(nested["password"].(string), Prefix))
	assert.Equal(t, "inner", nested["name"])

	list := out["list"].([]interface{})
	assert.True(t, strings.HasPrefix(list[0].(map[string]interface{})["client_secret"].(string), Prefix))

	// and back
	restored := DecryptConfig(out).(map[string]interface{})
	assert.Equal(t, "sk-123", restored["api_key"])
	assert.Equal(t, "hunter2", restored["nested"].(map[string]interface{})["password"])
	assert.Equal(t, "gh-token", restored["github_token"])
}

func TestConfigWalkUtilMapStrAndNil(t *testing.T) {
	withKey(t, "k")
	ms := util.MapStr{"token": "t1", "other": "o1"}
	out := EncryptConfig(ms).(util.MapStr)
	assert.True(t, strings.HasPrefix(out["token"].(string), Prefix))
	assert.Equal(t, "o1", out["other"])
	assert.Equal(t, "t1", DecryptConfig(out).(util.MapStr)["token"])

	assert.Nil(t, EncryptConfig(nil))
	assert.Nil(t, DecryptConfig(nil))
}

func TestIsSensitiveKey(t *testing.T) {
	for _, k := range []string{"api_key", "API_KEY", "Api-Key", "access_token", "s3_secret_access_key", "GITHUB_TOKEN", "db_password"} {
		assert.True(t, IsSensitiveKey(k), "%s should be sensitive", k)
	}
	for _, k := range []string{"url", "name", "tokenizer", "max_tokens", "page_size", "key_hint"} {
		assert.False(t, IsSensitiveKey(k), "%s should not be sensitive", k)
	}
}
