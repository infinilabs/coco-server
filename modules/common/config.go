/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import "infini.sh/framework/core/global"

var config *Config

func AppConfig() *Config {

	if config == nil {
		v := global.Lookup("APP_CONFIG")
		c, ok := v.(*Config)
		if ok {
			if c != nil {
				config = c
			}
		}
	}

	//double check
	if config == nil {
		panic("invalid coco config")
	}

	return config
}

type Config struct {
	OllamaConfig   OllamaConfig   `config:"ollama"`
	OpenAIConfig   OpenAIConfig   `config:"openai"`
	GenerateConfig GenerateConfig `config:"generate"`
	ServerInfo     ServerInfo     `config:"server"`
}

type OllamaConfig struct {
	Model         string `config:"model"`
	Endpoint      string `config:"endpoint"`
	ContextLength int    `config:"context_length"`
	Keepalive     string `config:"keepalive"`
}

type OpenAIConfig struct {
	Model         string `config:"model"`
	Endpoint      string `config:"endpoint"`
	Token         string `config:"token"`
	ContextLength int    `config:"context_length"`
	Keepalive     string `config:"keepalive"`
}

type GenerateConfig struct {
	MaxLength int `config:"max_length"`
	MaxTokens int `config:"max_tokens"`
}

const WEBSOCKET_USER_SESSION = "websocket-user-session"
const WEBSOCKET_SESSION_USER = "websocket-session-user"
