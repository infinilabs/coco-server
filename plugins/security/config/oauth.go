/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package config

type OAuthConfig struct {
	Enabled      bool                `config:"enabled"`
	ClientID     string              `config:"client_id"`
	ClientSecret string              `config:"client_secret"`
	DefaultRoles []string            `config:"default_roles"`
	RoleMapping  map[string][]string `config:"role_mapping"`
	AuthorizeUrl string              `config:"authorize_url"`
	TokenUrl     string              `config:"token_url"`
	RedirectUrl  string              `config:"redirect_url"`
	Scopes       []string            `config:"scopes"`

	SuccessPage string `config:"success_page"`
	FailedPage  string `config:"failed_page"`
}
