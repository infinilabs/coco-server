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

const AttachmentKVBucket = "file_attachments"
const AttachmentStatsBucket = "attachment_stats"

// attachment status,  `pending`, `processing`, `completed`, `canceled`, or `failed`.
const AttachmentStageInitialParsing = "initial_parsing"
const StatusPending = "pending"
const StatusProcessing = "processing"
const StatusCompleted = "completed"
const StatusCanceled = "canceled"
const StatusFailed = "failed"

const ProviderIntegration = "INTEGRATION"

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
