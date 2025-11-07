/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"sync"
	"time"

	"infini.sh/coco/core"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
)

var (
	config   *Config
	configMu sync.Mutex
)

func AppConfigFromFile() (*Config, error) {
	cocoConfig := Config{
		ServerInfo: &ServerInfo{Version: Version{Number: global.Env().GetVersion()}, Updated: time.Now()},
	}

	ok, err := env.ParseConfig("coco", &cocoConfig)
	if ok && err != nil {
		return nil, errors.New("invalid config")
	}

	return &cocoConfig, nil
}

func AppConfig() Config {

	if config == nil {
		reloadConfig()
	}

	//double check
	if config == nil {
		panic("invalid coco config")
	}
	retCfg := *config

	if retCfg.ServerInfo.AuthProvider.SSO.URL == "" || util.PrefixStr(retCfg.ServerInfo.AuthProvider.SSO.URL, "/") {
		retCfg.ServerInfo.AuthProvider.SSO.URL = util.JoinPath(retCfg.ServerInfo.Endpoint, "/#/login")
	} else if !util.PrefixStr(retCfg.ServerInfo.AuthProvider.SSO.URL, retCfg.ServerInfo.Endpoint) {
		retCfg.ServerInfo.AuthProvider.SSO.URL = util.JoinPath(retCfg.ServerInfo.Endpoint, "/#/login")
	}

	return *config
}

func reloadConfig() {
	v, err := AppConfigFromFile()
	if v != nil && err == nil {
		config = v
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
			config.ServerInfo.Version = Version{global.Env().GetVersion()}
		}
	}
	buf, _ = kv.GetValue(core.DefaultSettingBucketKey, []byte(core.DefaultAppSettingsKey))
	if buf != nil {
		appSettings := &AppSettings{}
		err := util.FromJSONBytes(buf, appSettings)
		if err == nil {
			config.AppSettings = appSettings
		}
	}
	buf, _ = kv.GetValue(core.DefaultSettingBucketKey, []byte(core.DefaultSearchSettingsKey))
	if buf != nil {
		searchSettings := &SearchSettings{}
		err := util.FromJSONBytes(buf, searchSettings)
		if err == nil {
			config.SearchSettings = searchSettings
		}
	}

	filebasedConfig, _ := AppConfigFromFile()
	if filebasedConfig != nil {
		//protect fields on managed mode
		if filebasedConfig.ServerInfo != nil {
			if global.Env().SystemConfig.WebAppConfig.Security.Managed {
				config.ServerInfo.Managed = global.Env().SystemConfig.WebAppConfig.Security.Managed
				config.ServerInfo.AuthProvider = filebasedConfig.ServerInfo.AuthProvider
				config.ServerInfo.Provider = filebasedConfig.ServerInfo.Provider
				config.ServerInfo.Endpoint = filebasedConfig.ServerInfo.Endpoint
				config.ServerInfo.Public = filebasedConfig.ServerInfo.Public
				config.ServerInfo.Version = filebasedConfig.ServerInfo.Version
			}

			config.ServerInfo.EncodeIconToBase64 = filebasedConfig.ServerInfo.EncodeIconToBase64
		}
	}
}

func SetAppConfig(c *Config) {
	configMu.Lock()
	defer configMu.Unlock()
	//save server's config
	err := kv.AddValue(core.DefaultSettingBucketKey, []byte(core.DefaultServerConfigKey), util.MustToJSONBytes(c.ServerInfo))
	if err != nil {
		panic(err)
	}
	//save chat start page's config
	err = kv.AddValue(core.DefaultSettingBucketKey, []byte(core.DefaultAppSettingsKey), util.MustToJSONBytes(c.AppSettings))
	if err != nil {
		panic(err)
	}
	//save search's config
	err = kv.AddValue(core.DefaultSettingBucketKey, []byte(core.DefaultSearchSettingsKey), util.MustToJSONBytes(c.SearchSettings))
	if err != nil {
		panic(err)
	}
	config = nil
	reloadConfig()
}

type Config struct {
	ServerInfo     *ServerInfo     `config:"server" json:"server,omitempty"`
	AppSettings    *AppSettings    `config:"app_settings" json:"app_settings,omitempty"`
	SearchSettings *SearchSettings `config:"search_settings" json:"search_settings,omitempty"`
}

const OLLAMA = "ollama"
const OPENAI = "openai"

type AppSettings struct {
	Chat *ChatConfig `json:"chat,omitempty" config:"chat" `
}

type ChatConfig struct {
	ChatStartPageConfig *core.ChatStartPageConfig `config:"start_page" json:"start_page,omitempty"`
}

type SearchSettings struct {
	Enabled     bool   `json:"enabled"`
	Integration string `json:"integration"`
}
