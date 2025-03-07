/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"sync"
	"time"
)

var (
	config   *Config
	configMu sync.Mutex
)

func AppConfig() Config {
	if config == nil {
		v := global.Lookup("APP_CONFIG")
		c, ok := v.(*Config)
		if ok {
			if c != nil {
				config = c
			}
		}
		if config == nil {
			config = &Config{}
		}
		//read settings from kv
		buf, _ := kv.GetValue(core.DefaultSettingBucketKey, []byte(core.DefaultServerConfigKey))
		if buf != nil {
			si := &ServerInfo{}
			err := util.FromJSONBytes(buf, si)
			if err == nil {
				config.ServerInfo = si
			}
		}
		buf, _ = kv.GetValue(core.DefaultSettingBucketKey, []byte(core.DefaultLLMConfigKey))
		if buf != nil {
			llm := &LLMConfig{}
			err := util.FromJSONBytes(buf, llm)
			if err == nil {
				config.LLMConfig = llm
			}
		}
		buf, _ = kv.GetValue(core.DefaultSettingBucketKey, []byte(core.DefaultConnectorConfigKey))
		if buf != nil {
			connectorCfg := &ConnectorInfo{}
			err := util.FromJSONBytes(buf, connectorCfg)
			if err == nil {
				config.Connector = connectorCfg
			}
		}
	}

	//double check
	if config == nil {
		panic("invalid coco config")
	}
	retCfg := *config
	if retCfg.LLMConfig.Parameters.MaxTokens <= 0 {
		retCfg.LLMConfig.Parameters.MaxTokens = 32000
	}

	if retCfg.LLMConfig.DefaultModel != "" {
		if retCfg.LLMConfig.IntentAnalysisModel == "" {
			retCfg.LLMConfig.IntentAnalysisModel = retCfg.LLMConfig.DefaultModel
		}
		if retCfg.LLMConfig.PickingDocModel == "" {
			retCfg.LLMConfig.PickingDocModel = retCfg.LLMConfig.DefaultModel
		}
		if retCfg.LLMConfig.AnsweringModel == "" {
			retCfg.LLMConfig.AnsweringModel = retCfg.LLMConfig.DefaultModel
		}
	}

	retCfg.ServerInfo.AuthProvider.SSO.URL = util.JoinPath(retCfg.ServerInfo.Endpoint, "/#/login")

	return *config
}
func SetAppConfig(c *Config) {
	configMu.Lock()
	defer configMu.Unlock()
	//save server's config
	err := kv.AddValue(core.DefaultSettingBucketKey, []byte(core.DefaultServerConfigKey), util.MustToJSONBytes(c.ServerInfo))
	if err != nil {
		panic(err)
	}
	//save LLM's config
	err = kv.AddValue(core.DefaultSettingBucketKey, []byte(core.DefaultLLMConfigKey), util.MustToJSONBytes(c.LLMConfig))
	if err != nil {
		panic(err)
	}
	//save connector's config
	err = kv.AddValue(core.DefaultSettingBucketKey, []byte(core.DefaultConnectorConfigKey), util.MustToJSONBytes(c.Connector))
	if err != nil {
		panic(err)
	}
	config = c
}

type Config struct {
	LLMConfig  *LLMConfig     `config:"llm" json:"llm,omitempty"`
	ServerInfo *ServerInfo    `config:"server" json:"server,omitempty"`
	Connector  *ConnectorInfo `config:"connector" json:"connector,omitempty"`
}
type ConnectorInfo struct {
	GoogleDrive GoogleDriveConfig `config:"google_drive" json:"google_drive,omitempty"`
	Updated     time.Time         `config:"updated" json:"updated,omitempty"`
}
type GoogleDriveConfig struct {
	// ClientID is the application's ID.
	ClientID string `json:"client_id"`
	// ClientSecret is the application's secret.
	ClientSecret string `json:"client_secret"`

	// RedirectURL is the URL to redirect users going through
	// the OAuth flow, after the resource owner's URLs.
	RedirectURL string `json:"redirect_url"`
	AuthURL     string `json:"auth_url"`
	TokenURL    string `json:"token_url"`
}

type LLMConfig struct {
	// LLM type, optional value "ollama" or "openai"
	Type          string        `config:"type" json:"type"`
	Endpoint      string        `config:"endpoint" json:"endpoint"`
	DefaultModel  string        `config:"default_model" json:"default_model,omitempty"`
	Parameters    LLMParameters `config:"parameters" json:"parameters"`
	Keepalive     string        `config:"keepalive" json:"keepalive,omitempty"`
	ContextLength uint64        `config:"context_length" json:"context_length,omitempty"`
	Token         string        `config:"token" json:"token,omitempty"`

	IntentAnalysisModel string `config:"intent_analysis_model" json:"intent_analysis_model,omitempty"`
	PickingDocModel     string `config:"picking_doc_model" json:"picking_doc_model,omitempty"`
	AnsweringModel      string `config:"answering_model" json:"answering_model,omitempty"`
}

type LLMParameters struct {
	Temperature       float64 `config:"temperature" json:"temperature"`
	TopP              float64 `config:"top_p" json:"top_p"`
	MaxTokens         int     `config:"max_tokens" json:"max_tokens"`
	PresencePenalty   float64 `config:"presence_penalty" json:"presence_penalty"`
	FrequencyPenalty  float64 `config:"frequency_penalty" json:"frequency_penalty"`
	EnhancedInference bool    `config:"enhanced_inference" json:"enhanced_inference"`
	MaxLength         int     `config:"max_length" json:"max_length"`
}

const WEBSOCKET_USER_SESSION = "websocket-user-session"
const WEBSOCKET_SESSION_USER = "websocket-session-user"
