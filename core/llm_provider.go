/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "sync"

type ModelProvider struct {
	CombinedFullText

	Name        string        `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext,fields:{text: {type: text}, pinyin: {type: text, analyzer: pinyin_analyzer}}}"`
	APIKey      string        `json:"api_key" elastic_mapping:"api_key:{type:keyword}"`                                // API key of the model provider
	APIType     string        `json:"api_type" elastic_mapping:"api_type:{type:keyword}"`                              // API type of the model provider, possible values: openai,gemini, anthropic
	Icon        string        `json:"icon" elastic_mapping:"icon:{enabled:false}"`                                     // Icon of the model provider
	Models      []ModelConfig `json:"models" elastic_mapping:"models:{type:object,enabled:false}"`                     // Models provided by the model provider
	BaseURL     string        `json:"base_url" elastic_mapping:"base_url:{enabled:false}"`                             // Base URL of the model provider
	Enabled     bool          `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`                                // Whether the model provider is enabled
	Builtin     bool          `json:"builtin" elastic_mapping:"builtin:{type:keyword}"`                                // Whether the model provider is builtin
	Description string        `json:"description" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"` // Description of the model provider
	Website     string        `json:"website" elastic_mapping:"website:{type:keyword}"`                                // Website of the model provider

	models    map[string]*ModelConfig
	getLocker sync.RWMutex
}

func (provider *ModelProvider) GetModelConfig(name string) *ModelConfig {
	if provider == nil {
		return nil
	}

	provider.getLocker.RLock()
	if v, ok := provider.models[name]; ok {
		provider.getLocker.RUnlock()
		return v
	}
	provider.getLocker.RUnlock()

	provider.getLocker.Lock()
	defer provider.getLocker.Unlock()

	if provider.models == nil {
		provider.models = make(map[string]*ModelConfig)
	}

	for i := range provider.Models {
		m := &provider.Models[i]
		provider.models[m.Name] = m
	}

	return provider.models[name]
}
