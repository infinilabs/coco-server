/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package netguard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateURLDenyListMode(t *testing.T) {
	t.Setenv(WhitelistEnv, "")

	// public https passes
	require.NoError(t, ValidateURL("https://parser.example.com/parse"))
	require.NoError(t, ValidateURL("http://203.0.113.10:8080/api")) // TEST-NET-3, global unicast

	// non-routable ranges are out
	for _, bad := range []string{
		"http://127.0.0.1:9200/",
		"http://localhost/x",
		"http://10.1.2.3/x",
		"http://192.168.1.1/x",
		"http://172.16.0.9/x",
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/x",
		"http://[::1]/x",
		"http://0.0.0.0/x",
	} {
		assert.Error(t, ValidateURL(bad), "%s must be denied", bad)
	}

	// scheme and port rules
	assert.Error(t, ValidateURL("file:///etc/passwd"), "non-http schemes are out")
	assert.Error(t, ValidateURL("ftp://example.com/x"))
	assert.Error(t, ValidateURL("http://parser.example.com:6379/"), "redis port is not fetchable")
	require.NoError(t, ValidateURL("http://parser.example.com:9000/"), "ordinary ports pass")
}

func TestValidateURLWhitelistMode(t *testing.T) {
	t.Setenv(WhitelistEnv, "on")
	t.Setenv(WhitelistExtraEnv, "parser.example.com, other.internal")

	require.NoError(t, ValidateURL("https://parser.example.com/parse"), "listed host passes")
	require.NoError(t, ValidateURL("https://OTHER.INTERNAL/x"), "whitelist match is case-insensitive")

	// an empty list denies everything — fail-closed by design
	t.Setenv(WhitelistExtraEnv, "")
	assert.Error(t, ValidateURL("https://parser.example.com/parse"))

	// unlisted public host is denied even though deny-list mode would allow it
	t.Setenv(WhitelistExtraEnv, "only-this.one")
	assert.Error(t, ValidateURL("https://public.example.com/x"))
}

func TestValidateURLHostnameResolution(t *testing.T) {
	t.Setenv(WhitelistEnv, "")
	// "localhost" resolves to loopback — denied through the resolver path
	assert.Error(t, ValidateURL("http://localhost:8080/x"))
	// an unresolvable name is not an SSRF verdict: the fetch itself fails
	assert.NoError(t, ValidateURL("http://this-host-does-not-exist-zzz9.invalid/x"))
}
