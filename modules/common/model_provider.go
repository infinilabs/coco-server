/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"time"

	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
)

const (
	ModelProviderCachePrimary = "model_provider"
)

// GetModelProvider retrieves the model provider object from the cache or database.
func GetModelProvider(providerID string) (*core.ModelProvider, error) {
	item := GeneralObjectCache.Get(ModelProviderCachePrimary, providerID)
	var provider *core.ModelProvider
	if item != nil && !item.Expired() {
		var ok bool
		if provider, ok = item.Value().(*core.ModelProvider); ok {
			return provider, nil
		}
	}
	provider = &core.ModelProvider{}
	provider.ID = providerID
	_, err := orm.Get(provider)
	if err != nil {
		return nil, err
	}
	// Cache the provider object
	GeneralObjectCache.Set(ModelProviderCachePrimary, providerID, provider, time.Duration(30)*time.Minute)
	return provider, nil
}

// ModelSupportsReasoning checks if the specified model supports reasoning mode.
// Returns false if the provider or model is not found, or if the model doesn't
// support reasoning.
func ModelSupportsReasoning(providerID, modelName string) bool {
	provider, err := GetModelProvider(providerID)
	if err != nil || provider == nil {
		return false
	}
	model := provider.GetModel(modelName)
	if model == nil {
		return false
	}
	return model.SupportReasoning
}
