/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

type ModelProvider struct {
	CombinedFullText

	Name        string   `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"`     // Name of the model provider
	APIKey      string   `json:"api_key" elastic_mapping:"api_key:{type:keyword}"`                         // API key of the model provider
	APIEndpoint string   `json:"api_endpoint" elastic_mapping:"api_endpoint:{type:keyword}"`               // API endpoint of the model provider
	Icon        string   `json:"icon" elastic_mapping:"icon:{type:keyword}"`                               // Icon of the model provider
	Models      []string `json:"models" elastic_mapping:"models:{type:keyword,copy_to:combined_fulltext}"` // Models provided by the model provider
	Enabled     bool     `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`                         // Whether the model provider is enabled
	Builtin     bool     `json:"builtin" elastic_mapping:"builtin:{type:keyword}"`                         // Whether the model provider is builtin
}
