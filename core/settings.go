/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

type Config struct {
	ServerInfo     *ServerInfo     `config:"server" json:"server,omitempty"`
	AppSettings    *AppSettings    `config:"app_settings" json:"app_settings,omitempty"`
	SearchSettings *SearchSettings `config:"search_settings" json:"search_settings,omitempty"`
}

type AppSettings struct {
	Chat *ChatConfig `json:"chat,omitempty" config:"chat" `
}

type ChatConfig struct {
	ChatStartPageConfig *ChatStartPageConfig `config:"start_page" json:"start_page,omitempty"`
}

type SearchSettings struct {
	Enabled     bool   `json:"enabled"`
	Integration string `json:"integration"`
}
