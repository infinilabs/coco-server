/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package config

type Config struct {
	Enabled        bool                 `config:"enabled"`
	Authentication AuthenticationConfig `config:"authc"`
}

type RealmConfig struct {
	Enabled bool `config:"enabled"`
	Order   int  `config:"order"`
}

type RealmsConfig struct {
	OAuth map[string]OAuthConfig `config:"oauth"`
}

type AuthenticationConfig struct {
	Realms RealmsConfig `config:"realms"`
}
