/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "infini.sh/framework/core/param"

const DefaultSettingBucketKey = "default_setting_bucket"

const UserProfileBucketKey = "user_profile"
const DefaultUserPasswordKey = "default_user_password"
const DefaultServerConfigKey = "default_server_config"
const DefaultAppSettingsKey = "default_app_settings"
const DefaultSearchSettingsKey = "default_search_settings"
const DefaultModelKey = "default_model"
const DefaultDocumentProcessingKey = "default_document_processing"

const AttachmentKVBucket = "file_attachments"
const AttachmentStatsBucket = "attachment_stats"

// AttachmentProcessingQueue is the name of the queue that the attachment upload
// handler pushes newly uploaded attachment IDs into. The process_attachments
// pipeline processor consumes this queue to run post-upload processing pipelines.
const AttachmentProcessingQueue = "attachment_processing"

// attachment status,  `pending`, `processing`, `completed`, `canceled`, or `failed`.
const AttachmentStageInitialParsing = "initial_parsing"
const StatusPending = "pending"
const StatusProcessing = "processing"
const StatusCompleted = "completed"
const StatusCanceled = "canceled"
const StatusFailed = "failed"

// UserSessionInfoKeyIntegration is a marker key stored on UserSessionInfo's
// embedded param.Parameters by the integration auth backend. When this key is
// present on a UserSessionInfo, the session was constructed by
// ValidateLoginByIntegrationHeader (i.e. it is an integration-authenticated
// guest run-as session, not a real user login). The associated value is the
// originating integration ID (string).
const UserSessionInfoKeyIntegration param.ParaKey = "integration"

const WidgetRole = "widget"

const PipelineContextConnector param.ParaKey = "__connector"
const PipelineContextDatasource param.ParaKey = "__datasource"
const PipelineContextDocuments param.ParaKey = "messages"


// re-export
const FeatureMaskSensitiveField = "feature_sensitive_fields"
const FeatureRemoveSensitiveField = "feature_sensitive_fields_remove_sensitive_field"
const SensitiveFields = "feature_sensitive_fields_extra_keys"

const FeatureCORS = "feature_cors"
const FeatureNotAllowCredentials = "feature_not_allow_credentials"
const FeatureByPassCORSCheck = "feature_bypass_cors_check"

const FeatureFingerprintThrottle = "fingerprint_throttle"

const DefaultSimpleAuthBackend = "default_simple_auth_backend"
const DefaultSimpleAuthUserLogin = "coco-default-user"

const SystemOwnerQueryField = "_system.owner_id"

const SuggestTagFieldNames = "field_names"
const SuggestTagFieldValues = "field_values"

const (
	AssistantTypeSimple           = "simple"
	AssistantTypeDeepThink        = "deep_think"
	AssistantTypeDeepResearch     = "deep_research"
	AssistantTypeExternalWorkflow = "external_workflow"

	AssistantCachePrimary = "assistant"
)
