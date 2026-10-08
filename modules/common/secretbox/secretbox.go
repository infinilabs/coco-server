/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

// Package secretbox provides at-rest encryption for credentials (S1).
//
// Design (DESIGN-knowledge-hub-v2.md §W1/S1):
//   - AES-256-GCM under a master key from the COCO_SECRET_KEY env var
//     (64-char hex recommended; any passphrase is stretched via SHA-256).
//   - Ciphertext carries the "enc:v1:" prefix. Read paths treat values
//     WITHOUT the prefix as legacy plaintext and pass them through —
//     gradual migration, zero downtime, no forced rewrite.
//   - Encrypt is idempotent: already-prefixed values are returned
//     unchanged, so a save round-trip of an encrypted blob never
//     double-wraps it.
//   - Missing master key = plaintext compatibility mode: writes stay
//     plaintext (one-time warn), prefixed values fail to decrypt.
//     A lost key means credentials are unreadable and must be re-entered;
//     there is no escrow by design.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

// Prefix marks a value as encrypted at rest.
const Prefix = "enc:v1:"

// MasterKeyEnv names the deployment's master key.
const MasterKeyEnv = "COCO_SECRET_KEY"

const nonceSize = 12

var (
	gcm             atomic.Pointer[cipher.AEAD]
	initMu          sync.Mutex
	warned          bool
	decryptFailSeen atomic.Bool // suppresses repeated decrypt-failure logs
)

// Enabled reports whether a master key is configured (encryption active).
func Enabled() bool {
	return aead() != nil
}

// ResetMasterKey drops the cached key so the next use re-reads the
// environment. Test seam only — production sets the key once per process.
func ResetMasterKey() {
	gcm.Store(nil)
	warned = false
	decryptFailSeen.Store(false)
}

func aead() cipher.AEAD {
	if v := gcm.Load(); v != nil {
		return *v
	}
	initMu.Lock()
	defer initMu.Unlock()
	if v := gcm.Load(); v != nil {
		return *v
	}
	raw := strings.TrimSpace(os.Getenv(MasterKeyEnv))
	if raw == "" {
		warnOnce("secretbox: %s not set — credentials stay plaintext at rest (compatibility mode)", MasterKeyEnv)
		return nil
	}
	block, err := aes.NewCipher(deriveKey(raw))
	if err != nil {
		warnOnce("secretbox: master key unusable: %v — staying plaintext", err)
		return nil
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		warnOnce("secretbox: GCM init failed: %v — staying plaintext", err)
		return nil
	}
	gcm.Store(&g)
	return g
}

// deriveKey accepts 64-char hex (recommended, exact 32 bytes) or stretches
// any passphrase through SHA-256 so operators are not forced into hex.
func deriveKey(raw string) []byte {
	if len(raw) == 64 {
		if b, err := hex.DecodeString(raw); err == nil {
			return b
		}
	}
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

func warnOnce(format string, args ...interface{}) {
	if !warned {
		log.Warnf(format, args...)
		warned = true
	}
}

// Encrypt seals a plaintext value. Disabled mode, empty input and
// already-encrypted values are returned unchanged.
func Encrypt(plain string) string {
	if plain == "" || strings.HasPrefix(plain, Prefix) {
		return plain
	}
	g := aead()
	if g == nil {
		return plain
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Errorf("secretbox: nonce generation failed: %v", err)
		return plain
	}
	sealed := g.Seal(nonce, nonce, []byte(plain), nil)
	return Prefix + base64.StdEncoding.EncodeToString(sealed)
}

// Decrypt opens an encrypted value. Values without the prefix are legacy
// plaintext and pass through. A decryption failure (wrong/lost key,
// tampered ciphertext) logs once and returns the empty string — a
// ciphertext must never be sent to a remote service as if it were the
// credential; an empty key produces a clean auth failure instead.
func Decrypt(value string) string {
	if !strings.HasPrefix(value, Prefix) {
		return value
	}
	g := aead()
	if g == nil {
		reportDecryptFailure("no master key configured")
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, Prefix))
	if err != nil {
		reportDecryptFailure("invalid base64 payload")
		return ""
	}
	if len(raw) < nonceSize {
		reportDecryptFailure("payload too short")
		return ""
	}
	plain, err := g.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		reportDecryptFailure("wrong master key or tampered ciphertext")
		return ""
	}
	return string(plain)
}

func reportDecryptFailure(reason string) {
	if !decryptFailSeen.Swap(true) {
		log.Warnf("secretbox: credential decryption failed (%s) — further failures suppressed; re-enter the affected credentials or restore %s", reason, MasterKeyEnv)
	}
}

// sensitiveConfigKeys are exact (lower-cased) config keys whose string
// values are credentials. Suffixes cover conventionally-named variants
// (github_token, client_secret, s3_secret_access_key ...) without
// matching unrelated words ("tokenizer" is safe).
var sensitiveConfigKeys = map[string]bool{
	"api_key": true, "apikey": true,
	"token": true, "access_token": true, "refresh_token": true,
	"secret": true, "secret_key": true, "secret_access_key": true,
	"client_secret": true, "app_secret": true,
	"password": true, "passwd": true, "private_key": true, "access_key": true,
}

var sensitiveConfigKeySuffixes = []string{"_token", "_secret", "_password", "_api_key", "_access_key"}

// IsSensitiveKey reports whether a config key name marks a credential.
// Hyphen/space variants normalize to underscores ("Api-Key" ≡ "api_key").
func IsSensitiveKey(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, "-", "_")
	n = strings.ReplaceAll(n, " ", "_")
	if sensitiveConfigKeys[n] {
		return true
	}
	for _, s := range sensitiveConfigKeySuffixes {
		if strings.HasSuffix(n, s) {
			return true
		}
	}
	return false
}

// EncryptConfig walks a connector/MCP config tree and encrypts the string
// values found under sensitive keys. Returns the same tree (mutated in
// place where possible) so callers can assign it back to interface{} fields.
func EncryptConfig(v interface{}) interface{} { return walk(v, true) }

// DecryptConfig is the read-side twin of EncryptConfig: prefixed values
// are opened, everything else passes through.
func DecryptConfig(v interface{}) interface{} { return walk(v, false) }

func walk(v interface{}, encrypt bool) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			t[k] = walkValue(k, val, encrypt)
		}
		return t
	case util.MapStr:
		for k, val := range t {
			t[k] = walkValue(k, val, encrypt)
		}
		return t
	case map[string]string:
		// struct-typed config (e.g. MCPServer stdio Env) saved without a
		// JSON round-trip still gets its credential-ish names sealed
		for k, val := range t {
			if val != "" && IsSensitiveKey(k) {
				if encrypt {
					t[k] = Encrypt(val)
				} else {
					t[k] = Decrypt(val)
				}
			}
		}
		return t
	case core.StdioConfig:
		// Config may hold the typed struct directly (no JSON round-trip);
		// Env is a map — sealed in place through the shared header
		for k, val := range t.Env {
			if val != "" && IsSensitiveKey(k) {
				if encrypt {
					t.Env[k] = Encrypt(val)
				} else {
					t.Env[k] = Decrypt(val)
				}
			}
		}
		return t
	case []interface{}:
		for i := range t {
			t[i] = walk(t[i], encrypt)
		}
		return t
	default:
		return v
	}
}

func walkValue(key string, val interface{}, encrypt bool) interface{} {
	if s, ok := val.(string); ok && IsSensitiveKey(key) {
		if encrypt {
			return Encrypt(s)
		}
		return Decrypt(s)
	}
	return walk(val, encrypt)
}
