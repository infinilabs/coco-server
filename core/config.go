/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "infini.sh/framework/core/param"

var DefaultSettingBucketKey = "default_setting_bucket"
var DefaultUserProfileKey = "default_user_profile" //TODO to be removed
var UserProfileKey = "user_profile"
var DefaultUserPasswordKey = "default_user_password"
var DefaultServerConfigKey = "default_server_config"
var DefaultAppSettingsKey = "default_app_settings"
var DefaultSearchSettingsKey = "default_search_settings"

var DefaultUserLogin = "coco-default-user"

var WidgetRole = "widget"

var PipelineContextConnector param.ParaKey = "__connector"
var PipelineContextDatasource param.ParaKey = "__datasource"
var PipelineContextDocuments param.ParaKey = "messages"

// re-export
const FeatureMaskSensitiveField = "feature_sensitive_fields"
const FeatureRemoveSensitiveField = "feature_sensitive_fields_remove_sensitive_field"
const SensitiveFields = "feature_sensitive_fields_extra_keys"

const FeatureCORS = "feature_cors"
const FeatureNotAllowCredentials = "feature_not_allow_credentials"
const FeatureByPassCORSCheck = "feature_bypass_cors_check"

const FeatureFingerprintThrottle = "fingerprint_throttle"
