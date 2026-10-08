/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package common

import (
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common/secretbox"
	"infini.sh/framework/core/orm"
)

// Credential guard (S1): credentials are encrypted at rest on every orm
// write and decrypted only at the load chokes that actually use them.
//
// Write side — one pre-hook over three models, mirroring the fingerprint
// hook's shape (modules/document/fingerprint_persist.go):
//   - ModelProvider.APIKey
//   - MCPServer.Config (stdio env tokens, header credentials)
//   - DataSource.Connector.Config (connector api keys/tokens/secrets)
//
// Read side — no orm read hook exists, so the decrypt calls live in the
// four load chokes every credential consumer goes through:
// GetModelProvider (langchain/embedding/rerank/engine-AI all use it),
// llm.GetMCPServersByID, GetDatasourceConfig (webhook/raw-content) and
// dispatcher.syncDatasource (connector pipelines). A missed choke fails
// loudly (ciphertext is never sent as a credential — Decrypt returns ""),
// never silently. API responses grow SAFER: what they echo today as
// plaintext becomes an enc:v1: blob; already-prefixed values round-trip
// through edits untouched (Encrypt is idempotent).
func init() {
	orm.RegisterDataOperationPreHook(200, func(ctx *orm.Context, _ orm.Operation, model interface{}) (*orm.Context, interface{}, error) {
		applyCredentialGuard(model)
		return ctx, model, nil
	}, orm.OpCreate, orm.OpUpdate, orm.OpSave)
}

// applyCredentialGuard encrypts credential fields in place; direct-call
// seam for tests.
func applyCredentialGuard(model interface{}) {
	switch m := model.(type) {
	case *core.ModelProvider:
		if m != nil {
			m.APIKey = secretbox.Encrypt(m.APIKey)
		}
	case *core.MCPServer:
		if m != nil {
			m.Config = secretbox.EncryptConfig(m.Config)
		}
	case *core.DataSource:
		if m != nil {
			m.Connector.Config = secretbox.EncryptConfig(m.Connector.Config)
		}
	}
}
