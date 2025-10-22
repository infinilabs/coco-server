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

var FeatureByPassCORSCheck = "feature_bypass_cors_check"

var PipelineContextConnector param.ParaKey = "__connector"
var PipelineContextDatasource param.ParaKey = "__datasource"
var PipelineContextDocuments param.ParaKey = "messages"
