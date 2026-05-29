/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"fmt"
	"sync"
)

// Model describes a model's static, immutable properties as defined by the
// model provider. These fields are determined when the model is trained and
// do not change at runtime.
//
// This is distinct from ModelConfig, which describes how a model is *used*
// (runtime settings like temperature, max tokens, etc.).
type Model struct {
	Name string  `json:"name"` // model ID / name
	Type LLMType `json:"type,omitempty"` // LLMTypeLanguage, LLMTypeVision, LLMTypeEmbedding

	// SupportReasoning reports whether this model is capable of reasoning mode.
	// Only meaningful for language models (Type == LLMTypeLanguage).
	SupportReasoning bool `json:"support_reasoning,omitempty"`
}

type ModelProvider struct {
	CombinedFullText

	Name        string  `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext,fields:{text: {type: text}, pinyin: {type: text, analyzer: pinyin_analyzer}}}"`
	APIKey      string  `json:"api_key" elastic_mapping:"api_key:{type:keyword}"`                                // API key of the model provider
	APIType     string  `json:"api_type" elastic_mapping:"api_type:{type:keyword}"`                              // API type of the model provider, possible values: openai,gemini, anthropic
	Icon        string  `json:"icon" elastic_mapping:"icon:{enabled:false}"`                                     // Icon of the model provider
	Models      []Model `json:"models" elastic_mapping:"models:{type:object,enabled:false}"`                     // Models provided by the model provider
	BaseURL     string  `json:"base_url" elastic_mapping:"base_url:{enabled:false}"`                             // Base URL of the model provider
	Enabled     bool    `json:"enabled" elastic_mapping:"enabled:{type:boolean}"`                                // Whether the model provider is enabled
	Builtin     bool    `json:"builtin" elastic_mapping:"builtin:{type:boolean}"`                                // Whether the model provider is builtin
	Description string  `json:"description" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"` // Description of the model provider
	Website     string  `json:"website" elastic_mapping:"website:{type:keyword}"`                                // Website of the model provider

	models    map[string]*Model
	getLocker sync.RWMutex
}

// LLMType identifies the kind of LLM a caller needs. It only enumerates real
// model categories; use cases like intent analysis or answering are not model
// types and have their own fallback handled at their call sites.
type LLMType string

const (
	LLMTypeLanguage  LLMType = "language"
	LLMTypeVision    LLMType = "vision"
	LLMTypeEmbedding LLMType = "embedding"
)

// GetModel returns the static model definition for the given model name.
// Returns nil if the model is not found in this provider.
func (provider *ModelProvider) GetModel(name string) *Model {
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
		provider.models = make(map[string]*Model)
	}

	for i := range provider.Models {
		m := &provider.Models[i]
		provider.models[m.Name] = m
	}

	return provider.models[name]
}

// Validate rejects a model definition that sets SupportReasoning on a
// non-language model type. When Type is unset the model is assumed to be a
// language model and the flag is allowed.
func (m *Model) Validate() error {
	if m.SupportReasoning && m.Type != "" && m.Type != LLMTypeLanguage {
		return fmt.Errorf("model %q: support_reasoning is only valid for language models", m.Name)
	}
	return nil
}

// ValidateModels calls Validate on every entry in provider.Models and returns
// the first error encountered.
func (provider *ModelProvider) ValidateModels() error {
	for i := range provider.Models {
		if err := provider.Models[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}
