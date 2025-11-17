/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package dispatcher

type Config struct {
	MaxRunningTimeoutInSeconds int  `json:"max_running_timeout_in_seconds" config:"max_running_timeout_in_seconds"`
	PipelinesInSync            bool `json:"pipelines_in_sync" config:"pipelines_in_sync"`
}
