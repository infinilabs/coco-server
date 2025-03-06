// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package common

import "time"

// ServerInfo represents the main structure for server configuration.
type ServerInfo struct {
	Name                 string       `json:"name" config:"name"`                                     // Config key for the server name
	Endpoint             string       `json:"endpoint" config:"endpoint"`                             // Config key for the server endpoint
	Provider             Provider     `json:"provider" config:"provider"`                             // Config key for the provider
	Version              Version      `json:"version" config:"version"`                               // Config key for the version
	MinimalClientVersion Version      `json:"minimal_client_version" config:"minimal_client_version"` // Config key for the version
	Updated              time.Time    `json:"updated" config:"updated"`                               // Config key for the updated time
	Public               bool         `json:"public" config:"public"`                                 // Config key for public visibility
	AuthProvider         AuthProvider `json:"auth_provider" config:"auth_provider"`                   // Config key for auth provider
}

// Provider represents the "provider" section of the configuration.
type Provider struct {
	Name          string `json:"name" config:"name"`                     // Config key for provider name
	Icon          string `json:"icon" config:"icon"`                     // Config key for provider icon
	Website       string `json:"website" config:"website"`               // Config key for provider website
	EULA          string `json:"eula" config:"eula"`                     // Config key for provider EULA
	PrivacyPolicy string `json:"privacy_policy" config:"privacy_policy"` // Config key for privacy policy
	Banner        string `json:"banner" config:"banner"`                 // Config key for provider banner
	Description   string `json:"description" config:"description"`       // Config key for provider description
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
