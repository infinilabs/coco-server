/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

type ChatStartPageConfig struct {
	Enabled bool `json:"enabled"`
	Logo    struct {
		Light string `json:"light"`
		Dark  string `json:"dark"`
	} `json:"logo"`
	Introduction      string   `json:"introduction"`
	DisplayAssistants []string `json:"display_assistants"`
}
