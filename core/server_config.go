/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "time"

// ServerInfo represents the main structure for server configuration.
type ServerInfo struct {
	Name                 string       `json:"name" config:"name"`                                               // Config key for the server name
	Endpoint             string       `json:"endpoint" config:"endpoint"`                                       // Config key for the server endpoint
	Provider             Provider     `json:"provider" config:"provider"`                                       // Config key for the provider
	Version              Version      `json:"version" config:"version"`                                         // Config key for the version
	MinimalClientVersion Version      `json:"minimal_client_version,omitempty" config:"minimal_client_version"` // Config key for the version
	Updated              time.Time    `json:"updated" config:"updated"`                                         // Config key for the updated time
	Public               bool         `json:"public,omitempty" config:"public"`                                 // Config key for public visibility
	AuthProvider         AuthProvider `json:"auth_provider,omitempty" config:"auth_provider"`                   // Config key for auth provider, link used by the APP
	Managed              bool         `json:"managed,omitempty" config:"managed" `                              // An alias to global.Env().SystemConfig.WebAppConfig.Security.Managed
	EncodeIconToBase64   bool         `json:"encode_icon_to_base64,omitempty" config:"encode_icon_to_base64" `
	Store                StoreConfig  `json:"store,omitempty" config:"store"` // Config key for store configuration
}

// Provider represents the "provider" section of the configuration.
type Provider struct {
	Name          string       `json:"name" config:"name"`                             // Config key for provider name
	Icon          string       `json:"icon" config:"icon"`                             // Config key for provider icon
	Website       string       `json:"website" config:"website"`                       // Config key for provider website
	EULA          string       `json:"eula" config:"eula"`                             // Config key for provider EULA
	PrivacyPolicy string       `json:"privacy_policy" config:"privacy_policy"`         // Config key for privacy policy
	Banner        string       `json:"banner" config:"banner"`                         // Config key for provider banner
	Description   string       `json:"description" config:"description"`               // Config key for provider description
	AuthProvider  AuthProvider `json:"auth_provider,omitempty" config:"auth_provider"` // Config key for auth provider
}

// Version represents the "version" section of the configuration.
type Version struct {
	Number string `json:"number" config:"number"` // Config key for version number
}

// AuthProvider represents the "auth_provider" section of the configuration.
type AuthProvider struct {
	SSO SSO `json:"sso" config:"sso"` // Config key for SSO
}

// SSO represents the "sso" section under "auth_provider".
type SSO struct {
	URL string `json:"url" config:"url"` // Config key for SSO URL
}

type StoreConfig struct {
	Endpoint string `json:"endpoint" config:"endpoint"` // store service endpoint
	Local    bool   `json:"local" config:"local"`       // whether use local store service
}
