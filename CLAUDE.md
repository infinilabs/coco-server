# COCO-Server Latest Status

**Last Updated**: 2025-11-17

## Current Work: MongoDB Connector

**Branch**: `mongo-connector`

**Build Status**: ✅ Backend and Frontend compile successfully

**Objective**: Align MongoDB connector with new package structure after main branch merge.

### Recent Changes (2025-11-17)

**Package Structure Migration** ✅ **COMPLETED**
- **Objective**: Update MongoDB and Neo4j connectors to use renamed core package after merging main branch
- **Changes Made**:

  1. **MongoDB Connector Updated**:
     - Changed imports from `"infini.sh/coco/modules/common"` to `"infini.sh/coco/core"`
     - Updated all type references throughout the codebase:
       - `common.Connector` → `core.Connector`
       - `common.DataSource` → `core.DataSource`
       - `common.Document` → `core.Document`
       - `common.DataSourceReference` → `core.DataSourceReference`
     - **Files Updated**:
       - `plugins/connectors/mongodb/plugin.go` - Updated imports and type references
       - `plugins/connectors/mongodb/scanner.go` - Updated all type references in scanner struct and methods
       - `plugins/connectors/mongodb/converter.go` - Updated document conversion functions

  2. **Neo4j Connector Updated** (side effect):
     - Also needed package migration after main merge
     - Updated imports and type references to use `core` package
     - **Files Updated**:
       - `plugins/connectors/neo4j/plugin.go` - Updated imports and Fetch() signature
       - `plugins/connectors/neo4j/scanner.go` - Updated scanner struct and transform() function

  3. **Generated Plugins File**:
     - Removed jira connector import (not present in this branch)
     - Added mongodb connector import
     - File: `plugins/generated_plugins.go`

  4. **Web Build Fix**:
     - Fixed broken import path for OAuthConnect component
     - Component was moved from `components/` to `pages/data-source/modules/`
     - **File Updated**:
       - `web/src/pages/data-source/new/index.tsx` - Updated import path from `@/components/oauth_connect` to `@/pages/data-source/modules/OauthConnect`

  5. **MongoDB Form Switch Components Fix**:
     - Fixed issue where Switch components were not saving state properly
     - Added `initialValue={false}` to all Switch components in MongoDB datasource form
     - Ensures form fields are initialized correctly and included in submission
     - **Files Updated**:
       - `web/src/pages/data-source/new/mongodb.tsx` - Added initialValue to pagination and field_mapping switches
       - `web/src/pages/data-source/modules/IncrementalSyncFields.tsx` - Added initialValue to incremental sync switch

  6. **MongoDB Authentication Documentation**:
     - Added comprehensive authentication section to MongoDB connector documentation
     - Documented connection URI formats with credentials
     - Added troubleshooting guide for common auth errors like "Command find requires authentication"
     - Updated examples to include authentication credentials
     - **File Updated**:
       - `docs/content.en/docs/references/connectors/mongodb.md` - Added authentication section and troubleshooting

- **Build Verification**: ✅ Full project builds successfully
  ```bash
  go build ./plugins/connectors/mongodb  # ✅ Success
  go build                               # ✅ Success
  make build-web                         # ✅ Success
  ```

### Git Merge History
- Successfully merged `main` branch into `mongo-connector`
- Resolved conflicts in:
  - `config/setup/en-US/connector.tpl` - Kept both MongoDB and gitlab_webhook_receiver
  - `config/setup/zh-CN/connector.tpl` - Kept both connectors
  - `docs/content.en/docs/release-notes/_index.md` - Merged feature lists
  - `go.mod` & `go.sum` - Merged dependencies
  - `plugins/connectors/common/converter.go` - Added reflect import

### MongoDB Connector Features
- ✅ Full MongoDB collection scanning
- ✅ Incremental sync with cursor-based pagination
- ✅ Field mapping support with Transformer
- ✅ BSON type normalization
- ✅ Cursor state management
- ✅ Configurable page size
- ✅ Sort specification support
- ✅ Custom query filters

### Architecture Patterns
- Uses `ConnectorProcessorBase` as base class
- Implements `Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource)` method
- Cursor-based incremental sync with watermark tracking
- BSON to core.Document transformation
- Field mapping with metadata/payload separation

### Next Steps
- ✅ Package migration complete
- ✅ Build verification passed
- Ready for testing and potential merge to main
- Consider adding automated tests
- Consider updating documentation

### Files
- `plugins/connectors/mongodb/plugin.go` - Main plugin with Fetch() implementation
- `plugins/connectors/mongodb/config.go` - Configuration structures and validation
- `plugins/connectors/mongodb/scanner.go` - Collection scanning with pagination
- `plugins/connectors/mongodb/converter.go` - BSON to Document transformation
- `plugins/connectors/mongodb/cursor.go` - Cursor extraction for incremental sync
