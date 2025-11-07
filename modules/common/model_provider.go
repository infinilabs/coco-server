/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/framework/core/orm"
	"time"
)

type ModelProvider struct {
	CombinedFullText

	Name        string        `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"`            // Name of the model provider
	APIKey      string        `json:"api_key" elastic_mapping:"api_key:{type:keyword}"`                                // API key of the model provider
	APIType     string        `json:"api_type" elastic_mapping:"api_type:{type:keyword}"`                              // API type of the model provider, possible values: openai,gemini, anthropic
	Icon        string        `json:"icon" elastic_mapping:"icon:{enabled:false}"`                                     // Icon of the model provider
	Models      []ModelConfig `json:"models" elastic_mapping:"models:{type:object,enabled:false}"`                     // Models provided by the model provider
	BaseURL     string        `json:"base_url" elastic_mapping:"base_url:{enabled:false}"`                             // Base URL of the model provider
	Enabled     bool          `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`                                // Whether the model provider is enabled
	Builtin     bool          `json:"builtin" elastic_mapping:"builtin:{type:keyword}"`                                // Whether the model provider is builtin
	Description string        `json:"description" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"` // Description of the model provider
	Website     string        `json:"website" elastic_mapping:"website:{type:keyword}"`                                // Website of the model provider
}

const (
	ModelProviderCachePrimary = "model_provider"
)

// GetModelProvider retrieves the model provider object from the cache or database.
func GetModelProvider(providerID string) (*ModelProvider, error) {
	item := GeneralObjectCache.Get(ModelProviderCachePrimary, providerID)
	var provider *ModelProvider
	if item != nil && !item.Expired() {
		var ok bool
		if provider, ok = item.Value().(*ModelProvider); ok {
			return provider, nil
		}
	}
	provider = &ModelProvider{}
	provider.ID = providerID
	_, err := orm.Get(provider)
	if err != nil {
		return nil, err
	}
	// Cache the provider object
	GeneralObjectCache.Set(ModelProviderCachePrimary, providerID, provider, time.Duration(30)*time.Minute)
	return provider, nil
}
