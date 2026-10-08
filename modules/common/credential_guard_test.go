/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common/secretbox"
)

func guardKey(t *testing.T) {
	t.Helper()
	t.Setenv(secretbox.MasterKeyEnv, "guard-test-key")
	secretbox.ResetMasterKey()
	t.Cleanup(secretbox.ResetMasterKey)
}

func TestCredentialGuardCoversThreeModels(t *testing.T) {
	guardKey(t)

	provider := &core.ModelProvider{APIKey: "sk-provider"}
	applyCredentialGuard(provider)
	require.True(t, strings.HasPrefix(provider.APIKey, secretbox.Prefix))
	assert.Equal(t, "sk-provider", secretbox.Decrypt(provider.APIKey))

	ds := &core.DataSource{}
	ds.Connector.Config = map[string]interface{}{"api_key": "gh-1", "url": "http://x"}
	applyCredentialGuard(ds)
	cfg := ds.Connector.Config.(map[string]interface{})
	assert.True(t, strings.HasPrefix(cfg["api_key"].(string), secretbox.Prefix))
	assert.Equal(t, "http://x", cfg["url"])

	mcp := &core.MCPServer{}
	mcp.Config = map[string]interface{}{
		"command": "npx",
		"env": map[string]interface{}{
			"GITHUB_TOKEN": "gh-2",
			"PATH":         "/usr/bin",
		},
	}
	applyCredentialGuard(mcp)
	env := mcp.Config.(map[string]interface{})["env"].(map[string]interface{})
	require.True(t, strings.HasPrefix(env["GITHUB_TOKEN"].(string), secretbox.Prefix), "env vars with credential-ish names must be sealed")
	assert.Equal(t, "/usr/bin", env["PATH"], "innocent env vars stay readable")

	// struct-typed stdio config (map[string]string Env) seals too
	structured := &core.MCPServer{Config: core.StdioConfig{
		Command: "npx",
		Env:     map[string]string{"GITHUB_TOKEN": "gh-3", "LANG": "C"},
	}}
	applyCredentialGuard(structured)
	require.True(t, strings.HasPrefix(structured.Config.(core.StdioConfig).Env["GITHUB_TOKEN"], secretbox.Prefix))
	assert.Equal(t, "C", structured.Config.(core.StdioConfig).Env["LANG"])
}

func TestCredentialGuardIdempotentRoundTrip(t *testing.T) {
	guardKey(t)

	provider := &core.ModelProvider{APIKey: "sk-roundtrip"}
	applyCredentialGuard(provider)
	sealed := provider.APIKey

	// a save round-trip (GET echo → PUT) must not double-wrap
	applyCredentialGuard(provider)
	assert.Equal(t, sealed, provider.APIKey)
}

func TestCredentialGuardIgnoresOtherModels(t *testing.T) {
	guardKey(t)
	doc := &core.Document{Content: "api_key: not-a-map"}
	applyCredentialGuard(doc)
	assert.Equal(t, "api_key: not-a-map", doc.Content)
	applyCredentialGuard(nil)
	applyCredentialGuard(&core.ModelProvider{}) // nil-safe empty key
}

func TestCredentialGuardDisabledModeNoop(t *testing.T) {
	t.Setenv(secretbox.MasterKeyEnv, "")
	secretbox.ResetMasterKey()
	t.Cleanup(secretbox.ResetMasterKey)

	provider := &core.ModelProvider{APIKey: "plain-when-disabled"}
	applyCredentialGuard(provider)
	assert.Equal(t, "plain-when-disabled", provider.APIKey, "compatibility mode must not mangle plaintext writes")
}
